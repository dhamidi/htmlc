package expr

import (
	"sort"
	"testing"
)

func TestCollectIdentifiers(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"simple identifier", "x", []string{"x"}},
		{"dot-notation member access", "user.name", []string{"user"}},
		{"bracket-notation member access", "items[idx]", []string{"items", "idx"}},
		{"binary expression", "a + b", []string{"a", "b"}},
		{"ternary expression", "x ? y : z", []string{"x", "y", "z"}},
		{"unary expression", "!flag", []string{"flag"}},
		{"call expression", "len(items)", []string{"len", "items"}},
		{"array literal", "[a, b, 1]", []string{"a", "b"}},
		{"object literal", "{key: val}", []string{"val"}},
		{"nested member + bracket", "a.b[c].d", []string{"a", "c"}},
		{"duplicate removal", "x + x", []string{"x"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CollectIdentifiers(tt.src)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			sort.Strings(got)
			sort.Strings(tt.want)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestCollectIdentifiers_CompileError(t *testing.T) {
	_, err := CollectIdentifiers("a +")
	if err == nil {
		t.Fatal("expected error for invalid expression, got nil")
	}
}
