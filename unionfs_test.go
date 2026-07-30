package htmlc

import (
	"errors"
	"io/fs"
	"sort"
	"testing"
	"testing/fstest"
)

func TestUnionFS_SatisfiesInterfaces(t *testing.T) {
	var (
		_ fs.FS        = (*unionFS)(nil)
		_ fs.StatFS    = (*unionFS)(nil)
		_ fs.ReadDirFS = (*unionFS)(nil)
	)
}

func twoMountUnion() *unionFS {
	root := fstest.MapFS{
		"HomePage.vue": &fstest.MapFile{Data: []byte("<template><div/></template>")},
		"Card.vue":     &fstest.MapFile{Data: []byte("<template><div/></template>")},
	}
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte("<template><div>radix accordion</div></template>")},
		"Tabs.vue":      &fstest.MapFile{Data: []byte("<template><div>radix tabs</div></template>")},
	}
	return newUnionFS([]fsMount{
		{prefix: "", fs: root},
		{prefix: "radix", fs: radix},
	})
}

func TestUnionFS_OpenRoutesByPrefix(t *testing.T) {
	u := twoMountUnion()

	f, err := u.Open("HomePage.vue")
	if err != nil {
		t.Fatalf("Open(HomePage.vue): %v", err)
	}
	f.Close()

	f, err = u.Open("radix/Accordion.vue")
	if err != nil {
		t.Fatalf("Open(radix/Accordion.vue): %v", err)
	}
	data := make([]byte, 512)
	n, _ := f.Read(data)
	f.Close()
	if got := string(data[:n]); got == "" || !containsString(got, "radix accordion") {
		t.Fatalf("Open(radix/Accordion.vue) content = %q, want it to contain %q", got, "radix accordion")
	}

	if _, err := u.Open("radix/DoesNotExist.vue"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Open(radix/DoesNotExist.vue) err = %v, want fs.ErrNotExist", err)
	}
}

func containsString(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func TestUnionFS_StatRoutesByPrefix(t *testing.T) {
	u := twoMountUnion()

	info, err := u.Stat("HomePage.vue")
	if err != nil {
		t.Fatalf("Stat(HomePage.vue): %v", err)
	}
	if info.IsDir() {
		t.Fatalf("Stat(HomePage.vue).IsDir() = true, want false")
	}

	info, err = u.Stat("radix/Accordion.vue")
	if err != nil {
		t.Fatalf("Stat(radix/Accordion.vue): %v", err)
	}
	if info.IsDir() {
		t.Fatalf("Stat(radix/Accordion.vue).IsDir() = true, want false")
	}

	// The mount prefix itself resolves to the mounted fs's root directory.
	info, err = u.Stat("radix")
	if err != nil {
		t.Fatalf("Stat(radix): %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("Stat(radix).IsDir() = false, want true")
	}

	if _, err := u.Stat("nope.vue"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Stat(nope.vue) err = %v, want fs.ErrNotExist", err)
	}
}

func TestUnionFS_ReadDirRoot_SynthesizesMountEntry(t *testing.T) {
	u := twoMountUnion()

	entries, err := u.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir(.): %v", err)
	}

	names := dirEntryNames(entries)
	want := []string{"Card.vue", "HomePage.vue", "radix"}
	assertNamesEqual(t, names, want)

	// Confirm the synthesized "radix" entry reports itself as a directory.
	for _, e := range entries {
		if e.Name() == "radix" && !e.IsDir() {
			t.Fatalf("synthesized radix entry IsDir() = false, want true")
		}
	}
}

func TestUnionFS_ReadDirMount_ListsUnderlyingContents(t *testing.T) {
	u := twoMountUnion()

	entries, err := u.ReadDir("radix")
	if err != nil {
		t.Fatalf("ReadDir(radix): %v", err)
	}
	names := dirEntryNames(entries)
	assertNamesEqual(t, names, []string{"Accordion.vue", "Tabs.vue"})
}

func TestUnionFS_NestedSubdirectoriesInsideMount(t *testing.T) {
	root := fstest.MapFS{
		"HomePage.vue": &fstest.MapFile{Data: []byte("<template/>")},
	}
	radix := fstest.MapFS{
		"Accordion.vue":          &fstest.MapFile{Data: []byte("<template/>")},
		"dialog/Trigger.vue":     &fstest.MapFile{Data: []byte("<template/>")},
		"dialog/Content.vue":     &fstest.MapFile{Data: []byte("<template/>")},
		"dialog/nested/Deep.vue": &fstest.MapFile{Data: []byte("<template/>")},
	}
	u := newUnionFS([]fsMount{
		{prefix: "", fs: root},
		{prefix: "radix", fs: radix},
	})

	entries, err := u.ReadDir("radix/dialog")
	if err != nil {
		t.Fatalf("ReadDir(radix/dialog): %v", err)
	}
	assertNamesEqual(t, dirEntryNames(entries), []string{"Content.vue", "Trigger.vue", "nested"})

	if _, err := u.Open("radix/dialog/nested/Deep.vue"); err != nil {
		t.Fatalf("Open(radix/dialog/nested/Deep.vue): %v", err)
	}

	info, err := u.Stat("radix/dialog")
	if err != nil {
		t.Fatalf("Stat(radix/dialog): %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("Stat(radix/dialog).IsDir() = false, want true")
	}
}

func TestUnionFS_MultiSegmentPrefix_SynthesizesIntermediateAncestor(t *testing.T) {
	root := fstest.MapFS{
		"HomePage.vue": &fstest.MapFile{Data: []byte("<template/>")},
	}
	vendorRadix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte("<template/>")},
	}
	u := newUnionFS([]fsMount{
		{prefix: "", fs: root},
		{prefix: "vendor/radix", fs: vendorRadix},
	})

	// Listing the root must synthesize the "vendor" intermediate segment.
	entries, err := u.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir(.): %v", err)
	}
	assertNamesEqual(t, dirEntryNames(entries), []string{"HomePage.vue", "vendor"})

	// Listing "vendor" must synthesize the "radix" segment.
	entries, err = u.ReadDir("vendor")
	if err != nil {
		t.Fatalf("ReadDir(vendor): %v", err)
	}
	assertNamesEqual(t, dirEntryNames(entries), []string{"radix"})

	// Listing "vendor/radix" reaches the real mounted contents.
	entries, err = u.ReadDir("vendor/radix")
	if err != nil {
		t.Fatalf("ReadDir(vendor/radix): %v", err)
	}
	assertNamesEqual(t, dirEntryNames(entries), []string{"Accordion.vue"})

	if _, err := u.Open("vendor/radix/Accordion.vue"); err != nil {
		t.Fatalf("Open(vendor/radix/Accordion.vue): %v", err)
	}

	// Stat on the purely-synthetic intermediate ancestor must report a directory.
	info, err := u.Stat("vendor")
	if err != nil {
		t.Fatalf("Stat(vendor): %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("Stat(vendor).IsDir() = false, want true")
	}

	// Open on the purely-synthetic intermediate ancestor must also succeed
	// and behave as a directory.
	f, err := u.Open("vendor")
	if err != nil {
		t.Fatalf("Open(vendor): %v", err)
	}
	defer f.Close()
	rdf, ok := f.(fs.ReadDirFile)
	if !ok {
		t.Fatalf("Open(vendor) did not return an fs.ReadDirFile")
	}
	des, err := rdf.ReadDir(-1)
	if err != nil {
		t.Fatalf("synthetic dir ReadDir(-1): %v", err)
	}
	assertNamesEqual(t, dirEntryNames(des), []string{"radix"})
}

func TestUnionFS_PathNotOwnedByAnyMount(t *testing.T) {
	u := twoMountUnion()

	if _, err := u.Open("nowhere.vue"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Open(nowhere.vue) err = %v, want fs.ErrNotExist", err)
	}
	var pe *fs.PathError
	if _, err := u.Open("nowhere.vue"); !errors.As(err, &pe) {
		t.Fatalf("Open(nowhere.vue) err type = %T, want *fs.PathError", err)
	}

	if _, err := u.Stat("also/nowhere.vue"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Stat(also/nowhere.vue) err = %v, want fs.ErrNotExist", err)
	}

	if _, err := u.ReadDir("also/nowhere"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadDir(also/nowhere) err = %v, want fs.ErrNotExist", err)
	}
}

// TestUnionFS_RealEntryWinsOverSyntheticOnNameCollision confirms that, if a
// mount's own root fs happens to already contain a real entry with the same
// name as another mount's prefix segment (a misconfiguration the RFC says
// callers must avoid, but which unionFS itself must still degrade
// predictably on rather than nondeterministically), ReadDir deterministically
// keeps the real entry rather than flipping between the two across calls.
func TestUnionFS_RealEntryWinsOverSyntheticOnNameCollision(t *testing.T) {
	root := fstest.MapFS{
		"radix/Local.vue": &fstest.MapFile{Data: []byte("<template/>")},
	}
	radixMount := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte("<template/>")},
	}
	u := newUnionFS([]fsMount{
		{prefix: "", fs: root},
		{prefix: "radix", fs: radixMount},
	})

	for i := 0; i < 5; i++ {
		entries, err := u.ReadDir(".")
		if err != nil {
			t.Fatalf("ReadDir(.): %v", err)
		}
		names := dirEntryNames(entries)
		if len(names) != 1 || names[0] != "radix" {
			t.Fatalf("ReadDir(.) names = %v, want exactly [\"radix\"] (deduplicated, deterministic)", names)
		}
	}
}

// TestUnionFS_PrefixVsRootFileCollision guards against a false-positive
// string-prefix match: a mount prefix "radix" must not match a root-mount
// file literally named "radix.vue" (whose fs.FS path is "radix.vue", not
// "radix/...").
func TestUnionFS_PrefixVsRootFileCollision(t *testing.T) {
	root := fstest.MapFS{
		"radix.vue": &fstest.MapFile{Data: []byte("<template><div>root radix.vue</div></template>")},
	}
	radixMount := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte("<template><div>mounted accordion</div></template>")},
	}
	u := newUnionFS([]fsMount{
		{prefix: "", fs: root},
		{prefix: "radix", fs: radixMount},
	})

	f, err := u.Open("radix.vue")
	if err != nil {
		t.Fatalf("Open(radix.vue): %v", err)
	}
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	f.Close()
	if got := string(buf[:n]); !containsString(got, "root radix.vue") {
		t.Fatalf("Open(radix.vue) content = %q, want it routed to the root mount's radix.vue", got)
	}

	// The mount itself is still reachable under its own prefix.
	if _, err := u.Open("radix/Accordion.vue"); err != nil {
		t.Fatalf("Open(radix/Accordion.vue): %v", err)
	}

	// Listing root must show both the literal file and the synthesized
	// mount-prefix directory entry, without confusing the two.
	entries, err := u.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir(.): %v", err)
	}
	assertNamesEqual(t, dirEntryNames(entries), []string{"radix", "radix.vue"})
	for _, e := range entries {
		switch e.Name() {
		case "radix.vue":
			if e.IsDir() {
				t.Fatalf("radix.vue entry IsDir() = true, want false")
			}
		case "radix":
			if !e.IsDir() {
				t.Fatalf("radix entry IsDir() = false, want true")
			}
		}
	}
}

// TestUnionFS_TrailingSlash confirms trailing-slash paths are rejected with
// a proper error rather than silently normalized or causing a panic.
func TestUnionFS_TrailingSlash(t *testing.T) {
	u := twoMountUnion()

	for _, name := range []string{"radix/", "HomePage.vue/", "/"} {
		if _, err := u.Open(name); err == nil {
			t.Fatalf("Open(%q) err = nil, want an error for an invalid path", name)
		}
		if _, err := u.Stat(name); err == nil {
			t.Fatalf("Stat(%q) err = nil, want an error for an invalid path", name)
		}
		if _, err := u.ReadDir(name); err == nil {
			t.Fatalf("ReadDir(%q) err = nil, want an error for an invalid path", name)
		}
	}
}

// TestUnionFS_WalkDir_VisitsEveryFileExactlyOnce is the most important test
// in this file: it exercises fs.WalkDir over a unionFS exactly the way
// Engine.discoverInto will (in a later commit), and confirms every file
// across every mount is visited exactly once, with directories (including
// synthesized ones) correctly descended into.
func TestUnionFS_WalkDir_VisitsEveryFileExactlyOnce(t *testing.T) {
	root := fstest.MapFS{
		"HomePage.vue":  &fstest.MapFile{Data: []byte("<template/>")},
		"Card.vue":      &fstest.MapFile{Data: []byte("<template/>")},
		"blog/Post.vue": &fstest.MapFile{Data: []byte("<template/>")},
	}
	radix := fstest.MapFS{
		"Accordion.vue":      &fstest.MapFile{Data: []byte("<template/>")},
		"Tabs.vue":           &fstest.MapFile{Data: []byte("<template/>")},
		"dialog/Trigger.vue": &fstest.MapFile{Data: []byte("<template/>")},
		"dialog/Content.vue": &fstest.MapFile{Data: []byte("<template/>")},
	}
	hypermedia := fstest.MapFS{
		"Frame.vue": &fstest.MapFile{Data: []byte("<template/>")},
	}
	u := newUnionFS([]fsMount{
		{prefix: "", fs: root},
		{prefix: "radix", fs: radix},
		{prefix: "hypermedia", fs: hypermedia},
	})

	visited := map[string]int{}
	var dirsVisited []string
	err := fs.WalkDir(u, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirsVisited = append(dirsVisited, p)
			return nil
		}
		visited[p]++
		return nil
	})
	if err != nil {
		t.Fatalf("fs.WalkDir: %v", err)
	}

	wantFiles := []string{
		"HomePage.vue",
		"Card.vue",
		"blog/Post.vue",
		"radix/Accordion.vue",
		"radix/Tabs.vue",
		"radix/dialog/Trigger.vue",
		"radix/dialog/Content.vue",
		"hypermedia/Frame.vue",
	}
	if len(visited) != len(wantFiles) {
		t.Fatalf("visited %d files, want %d: got %v", len(visited), len(wantFiles), visited)
	}
	for _, f := range wantFiles {
		if visited[f] != 1 {
			t.Errorf("file %q visited %d times, want exactly 1", f, visited[f])
		}
	}

	wantDirs := []string{".", "blog", "radix", "radix/dialog", "hypermedia"}
	for _, d := range wantDirs {
		if !containsStringSlice(dirsVisited, d) {
			t.Errorf("directory %q not visited by WalkDir; visited dirs = %v", d, dirsVisited)
		}
	}
}

func containsStringSlice(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func dirEntryNames(entries []fs.DirEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return names
}

func assertNamesEqual(t *testing.T, got, want []string) {
	t.Helper()
	gotSorted := append([]string(nil), got...)
	wantSorted := append([]string(nil), want...)
	sort.Strings(gotSorted)
	sort.Strings(wantSorted)
	if len(gotSorted) != len(wantSorted) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	for i := range gotSorted {
		if gotSorted[i] != wantSorted[i] {
			t.Fatalf("names = %v, want %v", got, want)
		}
	}
}
