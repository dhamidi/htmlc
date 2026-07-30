package turbo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStreamContentType(t *testing.T) {
	const want = "text/vnd.turbo-stream.html"
	if StreamContentType != want {
		t.Errorf("StreamContentType = %q, want %q", StreamContentType, want)
	}
}

func TestWantsStream(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		set    bool
		want   bool
	}{
		{
			name:   "exact match, sole value",
			accept: "text/vnd.turbo-stream.html",
			set:    true,
			want:   true,
		},
		{
			name:   "part of a realistic comma-separated list",
			accept: "text/html, text/vnd.turbo-stream.html, application/xhtml+xml",
			set:    true,
			want:   true,
		},
		{
			name:   "part of a comma-separated list with q parameters",
			accept: "text/html;q=0.9, text/vnd.turbo-stream.html;q=1.0, */*;q=0.1",
			set:    true,
			want:   true,
		},
		{
			name: "header absent",
			set:  false,
			want: false,
		},
		{
			name:   "header present but does not mention it",
			accept: "text/html",
			set:    true,
			want:   false,
		},
		{
			name:   "header present but empty",
			accept: "",
			set:    true,
			want:   false,
		},
		{
			name:   "similar but distinct media type does not false-positive",
			accept: "text/html, application/vnd.turbo-stream.html+json",
			set:    true,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.set {
				r.Header.Set("Accept", tt.accept)
			}
			if got := WantsStream(r); got != tt.want {
				t.Errorf("WantsStream() = %v, want %v (Accept: %q)", got, tt.want, tt.accept)
			}
		})
	}
}

func TestFrameID(t *testing.T) {
	t.Run("header set", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Turbo-Frame", "todo-list")

		id, ok := FrameID(r)
		if !ok {
			t.Fatal("FrameID() ok = false, want true")
		}
		if id != "todo-list" {
			t.Errorf("FrameID() id = %q, want %q", id, "todo-list")
		}
	})

	t.Run("header absent", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		id, ok := FrameID(r)
		if ok {
			t.Error("FrameID() ok = true, want false for absent header")
		}
		if id != "" {
			t.Errorf("FrameID() id = %q, want %q", id, "")
		}
	})

	// Documented choice: a header explicitly sent with an empty value is
	// "present but empty", distinct from "absent" — ok is true, matching
	// Go's usual "value, ok" idiom for presence (like a map lookup or
	// http.Header.Values), not a truthiness check on the value itself.
	t.Run("header present but empty", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Turbo-Frame", "")

		id, ok := FrameID(r)
		if !ok {
			t.Error("FrameID() ok = false, want true for an empty-but-present header")
		}
		if id != "" {
			t.Errorf("FrameID() id = %q, want %q", id, "")
		}
	})
}

func TestWriteStream(t *testing.T) {
	var b strings.Builder
	if err := WriteStream(&b, "replace", "todo-list", "<li>Buy milk</li>"); err != nil {
		t.Fatalf("WriteStream() error = %v", err)
	}

	want := `<turbo-stream action="replace" target="todo-list"><template><li>Buy milk</li></template></turbo-stream>`
	if got := b.String(); got != want {
		t.Errorf("WriteStream() wrote:\n%s\nwant:\n%s", got, want)
	}
}

func TestWriteStream_EscapesActionAndTarget(t *testing.T) {
	var b strings.Builder
	if err := WriteStream(&b, `re"place`, `todo"list`, "<p>x</p>"); err != nil {
		t.Fatalf("WriteStream() error = %v", err)
	}

	got := b.String()
	if strings.Contains(got, `action="re"place"`) || strings.Contains(got, `target="todo"list"`) {
		t.Errorf("WriteStream() did not escape quotes in action/target: %s", got)
	}
	if !strings.Contains(got, "<template><p>x</p></template>") {
		t.Errorf("WriteStream() fragment not wrapped/placed as expected: %s", got)
	}
}

func TestWriteStream_ConcatenatesCleanly(t *testing.T) {
	// RFC 014 §1.3/§6 Example 5: multiple <turbo-stream> elements written
	// into the same response require no special framing between them.
	var b strings.Builder

	if err := WriteStream(&b, "append", "todo-list", "<li>Buy milk</li>"); err != nil {
		t.Fatalf("first WriteStream() error = %v", err)
	}
	if err := WriteStream(&b, "update", "todo-count", "<span>1</span>"); err != nil {
		t.Fatalf("second WriteStream() error = %v", err)
	}

	want := `<turbo-stream action="append" target="todo-list"><template><li>Buy milk</li></template></turbo-stream>` +
		`<turbo-stream action="update" target="todo-count"><template><span>1</span></template></turbo-stream>`
	if got := b.String(); got != want {
		t.Errorf("concatenated WriteStream() output =\n%s\nwant:\n%s", got, want)
	}

	// Sanity check: exactly two well-formed elements, nothing extra between
	// or around them.
	if n := strings.Count(b.String(), "<turbo-stream "); n != 2 {
		t.Errorf("expected exactly 2 <turbo-stream> elements, found %d", n)
	}
}
