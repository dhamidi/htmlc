package radix

import (
	"io/fs"
	"testing"
)

func TestFS_ReturnsNonNil(t *testing.T) {
	got := FS()
	if got == nil {
		t.Fatal("FS() returned nil")
	}
}

func TestFS_PlaceholderComponentReadable(t *testing.T) {
	data, err := fs.ReadFile(FS(), "components/Placeholder.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Placeholder.vue) failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("components/Placeholder.vue read back empty content")
	}
}
