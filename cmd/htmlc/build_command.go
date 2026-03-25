package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/dhamidi/htmlc"
)

const helpBuild = `build — render a page tree to an output directory

SYNOPSIS
  htmlc build [-strict] [-dir <path>] [-pages <path>] [-out <path>] [-layout <name>] [-debug] [-dev <addr>]

DESCRIPTION
  Walks the pages directory recursively, renders every .vue file as a full
  HTML page, and writes the results to the output directory. The directory
  hierarchy is preserved: pages/posts/hello.vue becomes out/posts/hello.html.

  Props for each page are loaded by merging JSON data files in order:

    1. pages/_data.json          — root defaults (all pages)
    2. pages/subdir/_data.json   — subdirectory defaults (pages in that dir)
    3. pages/subdir/hello.json   — page-level props (highest priority)

  Each level is shallow-merged so page-level values always win. If no data
  files exist the page is rendered with no props.

FLAGS
  -strict         Enable strict mode: missing props abort with an error and all
                  components are validated before rendering.
  -dir string     Directory containing shared .vue components. (default ".")
  -pages string   Root of the page tree. (default "./pages")
  -out string     Output directory. Created if it does not exist. (default "./out")
  -layout string  Layout component (from -dir) to wrap every page. (default: none)
  -debug          Annotate output with data-htmlc-* attributes on each
                  component's root element (component name, source file, props).
  -dev string     Start a development server at <addr> (e.g. :8080) that serves
                  the output directory and automatically rebuilds when source
                  files change. The server runs until interrupted (Ctrl-C).
                  Build flags (-dir, -pages, -out, -layout, -debug) are still
                  honoured.

EXAMPLES
  # Build all pages using defaults
  htmlc build

  # Build with an explicit component dir, pages dir, and output dir
  htmlc build -dir ./templates -pages ./pages -out ./dist

  # Build with a shared layout
  htmlc build -dir ./templates -pages ./pages -out ./dist -layout AppLayout

  # Serve the built site with live rebuilds on port 8080
  htmlc build -dir ./templates -pages ./pages -out ./dist -dev :8080
`

// pageEntry describes a single page found during page discovery.
type pageEntry struct {
	// relPath is the path relative to the pages root, e.g. "posts/hello.vue"
	relPath string
	// absPath is the absolute path to the .vue file
	absPath string
	// dataPath is the path to the matching .json data file, or "" if none
	dataPath string
	// outPath is the resolved output path relative to -out, e.g. "posts/hello.html"
	outPath string
}

// discoverPages walks pagesDir recursively and returns a sorted slice of
// pageEntry for every .vue file found. Files whose base name starts with "_"
// are skipped (they are treated as shared partials, not pages).
func discoverPages(pagesDir string) ([]pageEntry, error) {
	var entries []pageEntry
	err := filepath.WalkDir(pagesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".vue" {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasPrefix(base, "_") {
			return nil
		}
		rel, err := filepath.Rel(pagesDir, path)
		if err != nil {
			return err
		}
		outPath := strings.TrimSuffix(rel, ".vue") + ".html"
		dataPath := strings.TrimSuffix(path, ".vue") + ".json"
		if _, statErr := os.Stat(dataPath); statErr != nil {
			dataPath = ""
		}
		entries = append(entries, pageEntry{
			relPath:  rel,
			absPath:  path,
			dataPath: dataPath,
			outPath:  outPath,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].relPath < entries[j].relPath
	})
	return entries, nil
}

// readJSONFile reads path and unmarshals its contents into a map.
// It returns a descriptive error if the file contains invalid JSON.
func readJSONFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%s: invalid JSON: %w", path, err)
	}
	return m, nil
}

// loadPageData loads and shallow-merges props for entry.
//
// It collects ancestor _data.json files from pagesRoot down to the page's
// parent directory (inclusive), then the page's own .json file
// (entry.dataPath), and shallow-merges them in that order so that
// page-level values take highest priority.
//
// Missing files are silently skipped. An error is returned only when a
// file that exists contains invalid JSON.
func loadPageData(entry pageEntry, pagesRoot string) (map[string]any, error) {
	pageDir := filepath.Dir(entry.absPath)
	rel, err := filepath.Rel(pagesRoot, pageDir)
	if err != nil {
		return nil, fmt.Errorf("resolving page directory: %w", err)
	}

	result := map[string]any{}

	// Collect _data.json paths from pagesRoot down to pageDir.
	// First check pagesRoot itself, then descend one component at a time.
	var dataDirs []string
	dataDirs = append(dataDirs, pagesRoot)
	if rel != "." {
		current := pagesRoot
		for _, part := range strings.Split(rel, string(filepath.Separator)) {
			current = filepath.Join(current, part)
			dataDirs = append(dataDirs, current)
		}
	}

	for _, dir := range dataDirs {
		candidate := filepath.Join(dir, "_data.json")
		if _, statErr := os.Stat(candidate); statErr != nil {
			continue
		}
		m, err := readJSONFile(candidate)
		if err != nil {
			return nil, err
		}
		for k, v := range m {
			result[k] = v
		}
	}

	// Apply page-level data last (highest priority).
	if entry.dataPath != "" {
		m, err := readJSONFile(entry.dataPath)
		if err != nil {
			return nil, err
		}
		for k, v := range m {
			result[k] = v
		}
	}

	return result, nil
}

// dirHash returns a hex digest summarising the mtimes of all files under dirs.
func dirHash(dirs ...string) (string, error) {
	h := sha256.New()
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			fmt.Fprintf(h, "%s\t%d\n", path, info.ModTime().UnixNano())
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// dirHashFS returns a hex digest summarising the mtimes of all files under dirs in fsys.
func dirHashFS(fsys fs.FS, dirs ...string) (string, error) {
	h := sha256.New()
	for _, dir := range dirs {
		err := fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			fmt.Fprintf(h, "%s\t%d\n", path, info.ModTime().UnixNano())
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// writeScripts writes the collected custom element scripts to
// <scriptsDir>/<hash>.js. It creates the directory only when collector has at
// least one script. It is a no-op when the collector is empty.
func writeScripts(collector *htmlc.CustomElementCollector, scriptsDir string) error {
	if collector.Len() == 0 {
		return nil
	}
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return fmt.Errorf("creating scripts directory: %w", err)
	}
	fsys := collector.ScriptsFS()
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(scriptsDir, path), data, 0644)
	})
}

// runDevServer starts an HTTP file server on addr that serves the out directory
// and rebuilds when source files change on each incoming request. Scripts
// collected from custom element components are served in-memory at /scripts/.
func runDevServer(addr, dir, pages, out, layout string, debug bool, strict bool, stdout, stderr io.Writer) error {
	var mu sync.Mutex
	lastHash, _ := dirHash(dir, pages)
	activeCollector := htmlc.NewCustomElementCollector()

	rebuild := func() {
		mu.Lock()
		defer mu.Unlock()
		h, err := dirHash(dir, pages)
		if err != nil || h == lastHash {
			return
		}
		lastHash = h
		fmt.Fprintf(stdout, "htmlc dev: change detected — rebuilding…\n")
		buildArgs := []string{"-dir", dir, "-pages", pages, "-out", out}
		if layout != "" {
			buildArgs = append(buildArgs, "-layout", layout)
		}
		if debug {
			buildArgs = append(buildArgs, "-debug")
		}
		var newCollector *htmlc.CustomElementCollector
		if err := runBuildCollect(buildArgs, stdout, stderr, strict, &newCollector); err != nil && err != errSilent {
			fmt.Fprintf(stderr, "htmlc dev: rebuild error: %v\n", err)
		}
		if newCollector != nil {
			activeCollector = newCollector
		}
	}

	staticFS := http.FileServer(http.Dir(out))
	mux := http.NewServeMux()

	// Serve custom element scripts from the in-memory collector at /scripts/.
	mux.HandleFunc("/scripts/", func(w http.ResponseWriter, r *http.Request) {
		rebuild()
		mu.Lock()
		coll := activeCollector
		mu.Unlock()
		http.StripPrefix("/scripts/", htmlc.NewScriptFSServer(coll)).ServeHTTP(w, r)
	})

	// Serve all other paths from the output directory.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rebuild()
		staticFS.ServeHTTP(w, r)
	})

	srv := &http.Server{Addr: addr, Handler: mux}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	fmt.Fprintf(stdout, "htmlc dev: serving %s on http://%s\n", out, addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("dev server: %w", err)
	}
	return nil
}

// runBuild is the entry point for the "build" subcommand.
func runBuild(args []string, stdout, stderr io.Writer, strict bool) error {
	return runBuildCollect(args, stdout, stderr, strict, nil)
}

// runBuildCollect implements the "build" subcommand. When collOut is non-nil,
// the CustomElementCollector populated during rendering is stored there so
// callers (e.g. the dev server) can access it after the build.
func runBuildCollect(args []string, stdout, stderr io.Writer, strict bool, collOut **htmlc.CustomElementCollector) error {
	fset := flag.NewFlagSet("build", flag.ContinueOnError)
	fset.SetOutput(stderr)
	dir := fset.String("dir", ".", "directory containing shared .vue components")
	pages := fset.String("pages", "./pages", "root of the page tree")
	out := fset.String("out", "./out", "output directory")
	layoutFlag := fset.String("layout", "", "layout component to wrap every page")
	debugFlag := fset.Bool("debug", false, "enable debug render mode")
	devAddr := fset.String("dev", "", "serve output directory and rebuild on changes (e.g. :8080)")
	if err := fset.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(stdout, helpBuild)
			return nil
		}
		return err
	}

	if _, statErr := os.Stat(*pages); statErr != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("build", fmt.Sprintf("cannot find pages directory %q", *pages),
			"  The pages directory does not exist. Create it and add .vue page files.",
			"",
			"  EXAMPLE",
			"    mkdir pages",
			fmt.Sprintf("    htmlc build -pages %s", *pages),
		))
		return errSilent
	}

	if _, statErr := os.Stat(*dir); statErr != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("build", fmt.Sprintf("cannot load components from %q", *dir),
			"  No such directory. Create the directory and add .vue component files.",
		))
		return errSilent
	}

	discovered, err := discoverPages(*pages)
	if err != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("build", fmt.Sprintf("page discovery failed: %v", err)))
		return errSilent
	}

	engine, err := htmlc.New(htmlc.Options{ComponentDir: *dir, Debug: *debugFlag})
	if err != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("build", fmt.Sprintf("failed to initialise engine: %v", err)))
		return errSilent
	}

	extDirs, err := discoverDirectives(*dir)
	if err != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("build", fmt.Sprintf("directive discovery: %v", err)))
		return errSilent
	}
	var extDirectives []*externalDirective
	for dname, dpath := range extDirs {
		ed := &externalDirective{name: dname, path: dpath, stderr: stderr}
		if startErr := ed.start(); startErr != nil {
			fmt.Fprintf(stderr, "htmlc build: directive %q: failed to start: %v\n", dname, startErr)
			continue
		}
		extDirectives = append(extDirectives, ed)
		engine.RegisterDirective(dname, ed)
	}
	defer func() {
		for _, ed := range extDirectives {
			ed.stop()
		}
	}()

	if strict {
		engine.WithMissingPropHandler(htmlc.ErrorOnMissingProp)
		if errs := engine.ValidateAll(); len(errs) > 0 {
			for _, ve := range errs {
				fmt.Fprintf(stderr, "htmlc build: validation error in %s: %s\n", ve.Component, ve.Message)
			}
			return errSilent
		}
	}

	if *layoutFlag != "" && !engine.Has(*layoutFlag) {
		fmt.Fprintln(stderr, cmdErrorMsg("build", fmt.Sprintf("layout component %q not found", *layoutFlag),
			fmt.Sprintf("  No component named %q was found in %q.", *layoutFlag, *dir),
			"  Create the layout component or check the -dir and -layout flags.",
		))
		return errSilent
	}

	if err := os.MkdirAll(*out, 0755); err != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("build", fmt.Sprintf("cannot create output directory %q: %v", *out, err)))
		return errSilent
	}

	verbose := isTerminal(stdout)
	failed := 0
	total := len(discovered)
	collector := htmlc.NewCustomElementCollector()

	for _, e := range discovered {
		name := strings.TrimSuffix(filepath.Base(e.relPath), ".vue")

		// Register the page component when the pages dir differs from the components dir.
		if *pages != *dir {
			if regErr := engine.Register(name, e.absPath); regErr != nil {
				fmt.Fprintf(stderr, "htmlc build: %s: %v\n", e.relPath, regErr)
				if verbose {
					fmt.Fprintf(stdout, "  ERROR  %s  (%v)\n", e.outPath, regErr)
				}
				failed++
				continue
			}
		}

		data, err := loadPageData(e, *pages)
		if err != nil {
			fmt.Fprintf(stderr, "htmlc build: %s: failed to load data: %v\n", e.relPath, err)
			if verbose {
				fmt.Fprintf(stdout, "  ERROR  %s  (%v)\n", e.outPath, err)
			}
			failed++
			continue
		}

		outFile := filepath.Join(*out, e.outPath)
		if mkErr := os.MkdirAll(filepath.Dir(outFile), 0755); mkErr != nil {
			fmt.Fprintf(stderr, "htmlc build: %s: cannot create output directory: %v\n", e.relPath, mkErr)
			if verbose {
				fmt.Fprintf(stdout, "  ERROR  %s  (%v)\n", e.outPath, mkErr)
			}
			failed++
			continue
		}

		f, createErr := os.Create(outFile)
		if createErr != nil {
			fmt.Fprintf(stderr, "htmlc build: %s: cannot create output file: %v\n", e.relPath, createErr)
			if verbose {
				fmt.Fprintf(stdout, "  ERROR  %s  (%v)\n", e.outPath, createErr)
			}
			failed++
			continue
		}

		var renderErr error
		if *layoutFlag != "" {
			content, fragErr := engine.RenderFragmentStringWithCollector(context.Background(), name, data, collector)
			if fragErr != nil {
				renderErr = fragErr
			} else {
				layoutData := make(map[string]any, len(data)+1)
				for k, v := range data {
					layoutData[k] = v
				}
				layoutData["content"] = content
				renderErr = engine.RenderPageWithCollector(context.Background(), f, *layoutFlag, layoutData, collector)
			}
		} else {
			renderErr = engine.RenderPageWithCollector(context.Background(), f, name, data, collector)
		}
		f.Close()

		if renderErr != nil {
			fmt.Fprintf(stderr, "htmlc build: %s: %v\n", e.relPath, renderErr)
			if verbose {
				fmt.Fprintf(stdout, "  ERROR  %s  (%v)\n", e.outPath, renderErr)
			}
			os.Remove(outFile)
			failed++
			continue
		}

		if verbose {
			fmt.Fprintf(stdout, "  built  %s\n", e.outPath)
		}
	}

	// Write collected custom element scripts to <out>/scripts/.
	if writeErr := writeScripts(collector, filepath.Join(*out, "scripts")); writeErr != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("build", fmt.Sprintf("failed to write scripts: %v", writeErr)))
	}

	fmt.Fprintf(stdout, "Build complete: %d pages, %d errors.\n", total, failed)
	if failed > 0 {
		return errSilent
	}
	if collOut != nil {
		*collOut = collector
	}
	if *devAddr != "" {
		return runDevServer(*devAddr, *dir, *pages, *out, *layoutFlag, *debugFlag, strict, stdout, stderr)
	}
	return nil
}
