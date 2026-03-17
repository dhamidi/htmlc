package htmlc

import (
	"sort"
	"testing"
)

// --- MapProps tests ---

func TestMapProps_GetAndKeys(t *testing.T) {
	m := map[string]any{"name": "Alice", "age": float64(30)}
	p := newMapProps(m)

	keys := p.Keys()
	sort.Strings(keys)
	if len(keys) != 2 {
		t.Fatalf("Keys() len = %d, want 2", len(keys))
	}
	if keys[0] != "age" || keys[1] != "name" {
		t.Errorf("Keys() = %v, want [age name]", keys)
	}

	v, ok := p.Get("name")
	if !ok || v != "Alice" {
		t.Errorf("Get(name) = %v, %v; want Alice, true", v, ok)
	}
	v, ok = p.Get("age")
	if !ok || v != float64(30) {
		t.Errorf("Get(age) = %v, %v; want 30, true", v, ok)
	}
	_, ok = p.Get("missing")
	if ok {
		t.Errorf("Get(missing) should return false")
	}
}

// --- StructProps tests ---

type spPlain struct {
	Name  string
	Email string
}

type spTagged struct {
	ID    int     `json:"id"`
	Title string  `json:"title"`
	Price float64 `json:"price"`
}

type spInner struct {
	Street string
	City   string
}

type spEmbedded struct {
	Name   string
	spInner // anonymous embedded
}

type spEmbeddedNamed struct {
	Name    string
	spInner `json:"addr"` // embedded with explicit json name — not promoted
}

type spOuter struct {
	City    string // shadows promoted City from spInner
	spInner        // anonymous embedded
}

type spWithNilPtr struct {
	Name    string
	Address *spInner // may be nil
}

func TestStructProps_PlainStruct(t *testing.T) {
	sp, ok := newStructProps(spPlain{Name: "Alice", Email: "alice@example.com"})
	if !ok {
		t.Fatal("newStructProps returned false")
	}

	keys := sp.Keys()
	sort.Strings(keys)
	if len(keys) != 2 || keys[0] != "Email" || keys[1] != "Name" {
		t.Errorf("Keys() = %v, want [Email Name]", keys)
	}

	v, ok := sp.Get("Name")
	if !ok || v != "Alice" {
		t.Errorf("Get(Name) = %v, %v; want Alice, true", v, ok)
	}
	v, ok = sp.Get("Email")
	if !ok || v != "alice@example.com" {
		t.Errorf("Get(Email) = %v, %v; want alice@example.com, true", v, ok)
	}
}

func TestStructProps_JsonTags(t *testing.T) {
	sp, ok := newStructProps(spTagged{ID: 1, Title: "Widget", Price: 9.99})
	if !ok {
		t.Fatal("newStructProps returned false")
	}

	keys := sp.Keys()
	sort.Strings(keys)
	if len(keys) != 3 || keys[0] != "id" || keys[1] != "price" || keys[2] != "title" {
		t.Errorf("Keys() = %v, want [id price title]", keys)
	}

	v, ok := sp.Get("title")
	if !ok || v != "Widget" {
		t.Errorf("Get(title) = %v, %v; want Widget, true", v, ok)
	}
	v, ok = sp.Get("id")
	if !ok || v != 1 {
		t.Errorf("Get(id) = %v, %v; want 1, true", v, ok)
	}
}

func TestStructProps_PointerToStruct(t *testing.T) {
	sp, ok := newStructProps(&spPlain{Name: "Bob"})
	if !ok {
		t.Fatal("newStructProps returned false for pointer")
	}
	v, ok := sp.Get("Name")
	if !ok || v != "Bob" {
		t.Errorf("Get(Name) = %v, %v; want Bob, true", v, ok)
	}
}

func TestStructProps_NilPointer(t *testing.T) {
	var p *spPlain
	_, ok := newStructProps(p)
	if ok {
		t.Error("newStructProps should return false for nil pointer")
	}
}

func TestStructProps_EmbeddedFlattening(t *testing.T) {
	sp, ok := newStructProps(spEmbedded{
		Name:    "Alice",
		spInner: spInner{Street: "123 Main", City: "NYC"},
	})
	if !ok {
		t.Fatal("newStructProps returned false")
	}

	keys := sp.Keys()
	sort.Strings(keys)
	// Expect: City, Name, Street (promoted from spInner)
	if len(keys) != 3 {
		t.Fatalf("Keys() = %v, want 3 keys", keys)
	}

	v, ok := sp.Get("Street")
	if !ok || v != "123 Main" {
		t.Errorf("Get(Street) = %v, %v; want 123 Main, true", v, ok)
	}
	v, ok = sp.Get("City")
	if !ok || v != "NYC" {
		t.Errorf("Get(City) = %v, %v; want NYC, true", v, ok)
	}
}

func TestStructProps_EmbeddedExplicitJsonName(t *testing.T) {
	// Use an exported embedded type so the json-named field is accessible.
	type Inner struct {
		Street string
		City   string
	}
	type Outer struct {
		Name  string
		Inner `json:"addr"` // embedded with explicit json name — not promoted
	}
	sp, ok := newStructProps(Outer{
		Name:  "Alice",
		Inner: Inner{Street: "123 Main", City: "NYC"},
	})
	if !ok {
		t.Fatal("newStructProps returned false")
	}

	keys := sp.Keys()
	sort.Strings(keys)
	// Expect: Name, addr (embedded is not promoted, stored as "addr")
	if len(keys) != 2 {
		t.Fatalf("Keys() = %v, want [Name addr]", keys)
	}
	if keys[0] != "Name" || keys[1] != "addr" {
		t.Errorf("Keys() = %v, want [Name addr]", keys)
	}

	// Street and City should not be accessible at the top level.
	_, ok = sp.Get("Street")
	if ok {
		t.Error("Get(Street) should be false when embedded has explicit json name")
	}
	_, ok = sp.Get("City")
	if ok {
		t.Error("Get(City) should be false when embedded has explicit json name")
	}
	// The embedded struct itself should be accessible under "addr".
	v, ok := sp.Get("addr")
	if !ok || v == nil {
		t.Errorf("Get(addr) = %v, %v; want non-nil, true", v, ok)
	}
}

func TestStructProps_OuterShadowsEmbedded(t *testing.T) {
	sp, ok := newStructProps(spOuter{
		City:    "OUTER",
		spInner: spInner{Street: "123 Main", City: "INNER"},
	})
	if !ok {
		t.Fatal("newStructProps returned false")
	}

	v, ok := sp.Get("City")
	if !ok || v != "OUTER" {
		t.Errorf("Get(City) = %v, %v; want OUTER, true (outer should win)", v, ok)
	}
}

func TestStructProps_NilPtrField(t *testing.T) {
	sp, ok := newStructProps(spWithNilPtr{Name: "Alice", Address: nil})
	if !ok {
		t.Fatal("newStructProps returned false")
	}

	v, ok := sp.Get("Address")
	if !ok {
		t.Error("Get(Address) should return true (nil ptr field found)")
	}
	if v != nil {
		t.Errorf("Get(Address) = %v, want nil (typed nil → untyped nil)", v)
	}
}

func TestStructProps_FirstRuneLowercase(t *testing.T) {
	sp, ok := newStructProps(spPlain{Name: "Alice", Email: "alice@example.com"})
	if !ok {
		t.Fatal("newStructProps returned false")
	}

	// "name" should match field "Name" via first-rune alias.
	v, ok := sp.Get("name")
	if !ok || v != "Alice" {
		t.Errorf("Get(name) = %v, %v; want Alice, true (first-rune alias)", v, ok)
	}
	// "email" should match field "Email".
	v, ok = sp.Get("email")
	if !ok || v != "alice@example.com" {
		t.Errorf("Get(email) = %v, %v; want alice@example.com, true", v, ok)
	}

	// The alias must not appear in Keys().
	for _, k := range sp.Keys() {
		if k == "name" || k == "email" {
			t.Errorf("Keys() must not include lowercase alias %q", k)
		}
	}
}

func TestStructProps_FirstRuneTagPreempts(t *testing.T) {
	// Field Name has json:"fullName" — "name" should NOT match via alias.
	type withTag struct {
		Name string `json:"fullName"`
	}
	sp, ok := newStructProps(withTag{Name: "Alice"})
	if !ok {
		t.Fatal("newStructProps returned false")
	}

	_, ok = sp.Get("name")
	if ok {
		t.Error("Get(name) should be false when field has json tag 'fullName'")
	}
	v, ok := sp.Get("fullName")
	if !ok || v != "Alice" {
		t.Errorf("Get(fullName) = %v, %v; want Alice, true", v, ok)
	}
}

// --- ToProps dispatch tests ---

func TestToProps_Dispatch(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		p, err := toProps(nil)
		if err != nil || p != nil {
			t.Errorf("toProps(nil) = %v, %v; want nil, nil", p, err)
		}
	})

	t.Run("map", func(t *testing.T) {
		m := map[string]any{"x": 1}
		p, err := toProps(m)
		if err != nil {
			t.Fatalf("toProps(map) err = %v", err)
		}
		if p == nil {
			t.Fatal("toProps(map) returned nil Props")
		}
		v, ok := p.Get("x")
		if !ok || v != 1 {
			t.Errorf("Get(x) = %v, %v; want 1, true", v, ok)
		}
	})

	t.Run("struct", func(t *testing.T) {
		p, err := toProps(spPlain{Name: "Bob"})
		if err != nil {
			t.Fatalf("toProps(struct) err = %v", err)
		}
		if p == nil {
			t.Fatal("toProps(struct) returned nil Props")
		}
		v, ok := p.Get("Name")
		if !ok || v != "Bob" {
			t.Errorf("Get(Name) = %v, %v; want Bob, true", v, ok)
		}
	})

	t.Run("ptr-to-struct", func(t *testing.T) {
		p, err := toProps(&spPlain{Name: "Carol"})
		if err != nil {
			t.Fatalf("toProps(ptr) err = %v", err)
		}
		if p == nil {
			t.Fatal("toProps(ptr) returned nil Props")
		}
		v, ok := p.Get("Name")
		if !ok || v != "Carol" {
			t.Errorf("Get(Name) = %v, %v; want Carol, true", v, ok)
		}
	})

	t.Run("props-identity", func(t *testing.T) {
		mp := newMapProps(map[string]any{"k": "v"})
		p, err := toProps(mp)
		if err != nil {
			t.Fatalf("toProps(Props) err = %v", err)
		}
		// Should dispatch via Props identity path — verify via behavior.
		v, ok := p.Get("k")
		if !ok || v != "v" {
			t.Errorf("props identity: Get(k) = %v, %v; want v, true", v, ok)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		_, err := toProps(42)
		if err == nil {
			t.Error("toProps(int) should return error")
		}
	})
}
