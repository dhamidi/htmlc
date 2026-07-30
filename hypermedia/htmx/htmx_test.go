package htmx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsHTMXRequest(t *testing.T) {
	tests := []struct {
		name   string
		header string
		set    bool
		want   bool
	}{
		{name: "header set to true", header: "true", set: true, want: true},
		{name: "header absent", set: false, want: false},
		{name: "header set to false", header: "false", set: true, want: false},
		{name: "header set to garbage", header: "yes", set: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.set {
				r.Header.Set("HX-Request", tt.header)
			}
			if got := IsHTMXRequest(r); got != tt.want {
				t.Errorf("IsHTMXRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsBoosted(t *testing.T) {
	tests := []struct {
		name   string
		header string
		set    bool
		want   bool
	}{
		{name: "header set to true", header: "true", set: true, want: true},
		{name: "header absent", set: false, want: false},
		{name: "header set to false", header: "false", set: true, want: false},
		{name: "header set to garbage", header: "yes", set: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.set {
				r.Header.Set("HX-Boosted", tt.header)
			}
			if got := IsBoosted(r); got != tt.want {
				t.Errorf("IsBoosted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetTrigger(t *testing.T) {
	w := httptest.NewRecorder()
	SetTrigger(w, "counter-updated")

	if got := w.Header().Get("HX-Trigger"); got != "counter-updated" {
		t.Errorf("HX-Trigger header = %q, want %q", got, "counter-updated")
	}
}

func TestNeitherHeaderSet(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	if IsHTMXRequest(r) {
		t.Error("IsHTMXRequest() = true, want false for a plain navigation request")
	}
	if IsBoosted(r) {
		t.Error("IsBoosted() = true, want false for a plain navigation request")
	}
}
