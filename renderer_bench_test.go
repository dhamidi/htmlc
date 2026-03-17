package htmlc

import (
	"testing"
)

// benchStruct is a representative struct used in benchmarks.
type benchStruct struct {
	Name    string
	Email   string `json:"email"`
	Age     int    `json:"age"`
	Role    string `json:"role"`
	Active  bool
	Score   float64 `json:"score"`
	benchEmbed
}

type benchEmbed struct {
	Department string
	Location   string
}

var benchVal = benchStruct{
	Name:       "Alice",
	Email:      "alice@example.com",
	Age:        30,
	Role:       "admin",
	Active:     true,
	Score:      9.5,
	benchEmbed: benchEmbed{Department: "Engineering", Location: "NYC"},
}

// BenchmarkStructProps_Get benchmarks lazy StructProps.Get — a single-field
// reflection lookup without allocating a full map[string]any.
func BenchmarkStructProps_Get(b *testing.B) {
	sp, ok := newStructProps(benchVal)
	if !ok {
		b.Fatal("newStructProps returned false")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sp.Get("email")
	}
}

// BenchmarkStructProps_Keys benchmarks StructProps.Keys to confirm it enumerates
// fields without pre-computing a map.
func BenchmarkStructProps_Keys(b *testing.B) {
	sp, ok := newStructProps(benchVal)
	if !ok {
		b.Fatal("newStructProps returned false")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sp.Keys()
	}
}

// BenchmarkToStringMap_EagerMap benchmarks the old eager-map path via toProps
// on a map[string]any to establish a baseline for comparison.
// The eager allocation cost is visible in allocs/op when using a struct path.
func BenchmarkToStringMap_EagerMap(b *testing.B) {
	// Simulate the old eager approach: convert struct to map[string]any once per call.
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sp, ok := newStructProps(benchVal)
		if !ok {
			b.Fatal("newStructProps returned false")
		}
		seen := make(map[string]struct{})
		_ = collectStructKeys(sp.rv, seen)
	}
}
