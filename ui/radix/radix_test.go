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

func TestFS_AccordionComponentReadable(t *testing.T) {
	data, err := fs.ReadFile(FS(), "components/Accordion.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Accordion.vue) failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("components/Accordion.vue read back empty content")
	}
}
