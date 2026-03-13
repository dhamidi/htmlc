package main

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

// directiveNameRe matches valid directive names: lower-kebab-case starting with a letter.
var directiveNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// discoverDirectives walks dir and returns a map of directive name → absolute
// executable path for every file matching the naming convention.
//
// A file is registered as an external directive when:
//   - Its base name (without extension) matches v-<directive-name>
//   - The directive name matches [a-z][a-z0-9-]*
//   - The file has at least one executable bit set (os.FileMode & 0111 != 0)
//
// Files inside hidden directories (name starts with ".") are skipped.
func discoverDirectives(dir string) (map[string]string, error) {
	result := make(map[string]string)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories.
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		// Strip extension to get the base name for matching.
		base := d.Name()
		if i := strings.IndexByte(base, '.'); i >= 0 {
			base = base[:i]
		}

		// Must start with "v-".
		if !strings.HasPrefix(base, "v-") {
			return nil
		}
		directiveName := base[2:]

		// Directive name must be lower-kebab-case.
		if !directiveNameRe.MatchString(directiveName) {
			return nil
		}

		// Must be executable.
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&0111 == 0 {
			return nil
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		result[directiveName] = absPath
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
