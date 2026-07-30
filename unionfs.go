package htmlc

import (
	"errors"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"
)

// errIsDirectory is returned from a synthesized directory's Read method,
// mirroring the error real filesystems return when Read is called on a
// directory file.
var errIsDirectory = errors.New("is a directory")

// fsMount pairs a source filesystem with the path prefix it is mounted at
// within a unionFS. An empty prefix mounts fs at the union's root.
//
// prefix is stored normalized: no leading or trailing slash, "" for the
// root mount. Callers constructing a unionFS may pass prefixes with or
// without slashes; newUnionFS normalizes them.
type fsMount struct {
	prefix string
	fs     fs.FS
}

// unionFS presents N source fs.FS values, each mounted at a distinct path
// prefix, as a single logical filesystem tree. For example, mounting fsA at
// "" (root) and fsB at "radix" makes fsB's "Accordion.vue" reachable as
// "radix/Accordion.vue" while fsA's own files are reachable unprefixed.
//
// unionFS implements fs.FS, fs.StatFS, and fs.ReadDirFS — the exact minimal
// surface fs.WalkDir needs to walk a filesystem (it uses ReadDirFS directly
// when present, and Stat for the walk root), which is in turn the exact
// surface Engine.discoverInto relies on.
//
// Routing uses longest-prefix matching on path *segments*, never on raw
// string prefixes, so a mount prefix "radix" never falsely matches a
// same-mount file literally named "radix.vue" (its path is "radix.vue", not
// "radix/..."). At most one mount is expected to carry the empty prefix
// (the root); it is consulted only when no more specific mount matches.
//
// Because a mounted fs.FS's own contents have no idea they are mounted
// under a prefix, listing a directory that is a strict ancestor of a mount
// prefix (e.g. listing "." when a mount's prefix is "radix", or listing
// "vendor" when a mount's prefix is "vendor/radix") synthesizes a directory
// entry for the next path segment so that fs.WalkDir actually descends into
// the mounted subtree.
//
// unionFS uses the "path" package (forward-slash), never "filepath",
// throughout: fs.FS paths are always slash-separated regardless of host OS.
type unionFS struct {
	// mounts is sorted by descending prefix depth (segment count) so the
	// most specific mount is always tried before a less specific one; the
	// root mount ("" — zero segments) is therefore always tried last.
	mounts []fsMount
}

var (
	_ fs.FS        = (*unionFS)(nil)
	_ fs.StatFS    = (*unionFS)(nil)
	_ fs.ReadDirFS = (*unionFS)(nil)
)

// newUnionFS builds a unionFS from mounts. Prefixes are normalized (leading
// and trailing slashes trimmed; "." treated as "") and mounts are sorted by
// descending specificity for routing. newUnionFS does not validate that
// prefixes are unique or non-overlapping — callers that need that guarantee
// (e.g. Engine.New validating Mount.Prefix uniqueness) enforce it
// separately; when two mounts do share a prefix, the one appearing earlier
// in mounts wins routing ties.
func newUnionFS(mounts []fsMount) *unionFS {
	normalized := make([]fsMount, len(mounts))
	for i, m := range mounts {
		p := strings.Trim(m.prefix, "/")
		if p == "." {
			p = ""
		}
		normalized[i] = fsMount{prefix: p, fs: m.fs}
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		return len(pathSegments(normalized[i].prefix)) > len(pathSegments(normalized[j].prefix))
	})
	return &unionFS{mounts: normalized}
}

// pathSegments splits a clean, slash-separated fs.FS path into its
// components. The root "." (and "") has zero segments.
func pathSegments(p string) []string {
	if p == "" || p == "." {
		return nil
	}
	return strings.Split(p, "/")
}

// match finds the mount that owns name (an already fs.ValidPath-valid path,
// e.g. "." or "radix/Accordion.vue") and returns that mount plus the path
// to pass to its underlying fs.FS after stripping the matched prefix.
func (u *unionFS) match(name string) (fsMount, string, bool) {
	for _, m := range u.mounts {
		if m.prefix == "" {
			return m, name, true
		}
		if name == m.prefix {
			return m, ".", true
		}
		if strings.HasPrefix(name, m.prefix+"/") {
			return m, name[len(m.prefix)+1:], true
		}
	}
	return fsMount{}, "", false
}

// syntheticChildren returns the immediate child path segments, sorted and
// deduplicated, that must be synthesized as directory entries when listing
// name because a mount prefix descends through name without name itself
// belonging to any single mount's real underlying tree.
func (u *unionFS) syntheticChildren(name string) []string {
	nameSegs := pathSegments(name)
	seen := make(map[string]bool)
	var out []string
	for _, m := range u.mounts {
		if m.prefix == "" {
			continue
		}
		mSegs := pathSegments(m.prefix)
		if len(mSegs) <= len(nameSegs) {
			continue
		}
		matches := true
		for i, s := range nameSegs {
			if mSegs[i] != s {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		child := mSegs[len(nameSegs)]
		if !seen[child] {
			seen[child] = true
			out = append(out, child)
		}
	}
	sort.Strings(out)
	return out
}

// Open implements fs.FS.
func (u *unionFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	if m, sub, ok := u.match(name); ok {
		f, err := m.fs.Open(sub)
		if err == nil {
			return f, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	if name == "." || len(u.syntheticChildren(name)) > 0 {
		entries, err := u.ReadDir(name)
		if err != nil {
			return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
		}
		return &syntheticDir{name: name, entries: entries}, nil
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

// Stat implements fs.StatFS.
func (u *unionFS) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrInvalid}
	}
	if m, sub, ok := u.match(name); ok {
		info, err := fs.Stat(m.fs, sub)
		if err == nil {
			return info, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	if name == "." || len(u.syntheticChildren(name)) > 0 {
		return syntheticFileInfo(path.Base(name)), nil
	}
	return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
}

// ReadDir implements fs.ReadDirFS.
func (u *unionFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}

	var entries []fs.DirEntry
	found := false

	if m, sub, ok := u.match(name); ok {
		got, err := fs.ReadDir(m.fs, sub)
		switch {
		case err == nil:
			found = true
			entries = append(entries, got...)
		case errors.Is(err, fs.ErrNotExist):
			// No real directory at this path in the owning mount; synthetic
			// children (checked below) may still make this a valid listing.
		default:
			return nil, err
		}
	}

	for _, seg := range u.syntheticChildren(name) {
		found = true
		entries = append(entries, syntheticDirEntry(seg))
	}

	if !found {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}

	return dedupeSortedDirEntries(entries), nil
}

// dedupeSortedDirEntries sorts entries by name and removes duplicates
// (keeping the first occurrence), guarding against the case where a real
// mount entry and a synthesized ancestor segment share the same name. The
// sort is stable and real entries are always appended before synthetic ones
// by ReadDir, so a real entry deterministically wins such a collision.
func dedupeSortedDirEntries(entries []fs.DirEntry) []fs.DirEntry {
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	out := entries[:0:0]
	seen := false
	var last string
	for _, e := range entries {
		if seen && e.Name() == last {
			continue
		}
		out = append(out, e)
		last = e.Name()
		seen = true
	}
	return out
}

// syntheticDirEntry is a fs.DirEntry for a mount-prefix segment that has no
// corresponding real entry in any single underlying mount (e.g. "radix" as
// seen from the union root, when only "radix/Accordion.vue" etc. actually
// exist inside the mounted fs).
type syntheticDirEntry string

func (s syntheticDirEntry) Name() string      { return string(s) }
func (s syntheticDirEntry) IsDir() bool       { return true }
func (s syntheticDirEntry) Type() fs.FileMode { return fs.ModeDir }
func (s syntheticDirEntry) Info() (fs.FileInfo, error) {
	return syntheticFileInfo(s), nil
}

// syntheticFileInfo is the fs.FileInfo counterpart of syntheticDirEntry.
type syntheticFileInfo string

func (s syntheticFileInfo) Name() string       { return string(s) }
func (s syntheticFileInfo) Size() int64        { return 0 }
func (s syntheticFileInfo) Mode() fs.FileMode  { return fs.ModeDir | 0o555 }
func (s syntheticFileInfo) ModTime() time.Time { return time.Time{} }
func (s syntheticFileInfo) IsDir() bool        { return true }
func (s syntheticFileInfo) Sys() any           { return nil }

// syntheticDir is the fs.File (and fs.ReadDirFile) returned by Open for a
// synthesized ancestor directory — one that exists only because a mount
// prefix descends through it, not because any single underlying mount has a
// real directory there.
type syntheticDir struct {
	name    string
	entries []fs.DirEntry
	offset  int
}

func (d *syntheticDir) Stat() (fs.FileInfo, error) {
	return syntheticFileInfo(path.Base(d.name)), nil
}

func (d *syntheticDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.name, Err: errIsDirectory}
}

func (d *syntheticDir) Close() error { return nil }

// ReadDir implements fs.ReadDirFile.
func (d *syntheticDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if n <= 0 {
		rest := d.entries[d.offset:]
		d.offset = len(d.entries)
		return rest, nil
	}
	if d.offset >= len(d.entries) {
		return nil, io.EOF
	}
	end := d.offset + n
	if end > len(d.entries) {
		end = len(d.entries)
	}
	batch := d.entries[d.offset:end]
	d.offset = end
	return batch, nil
}

var _ fs.ReadDirFile = (*syntheticDir)(nil)
