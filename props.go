package htmlc

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// Props is the interface htmlc demands of any value used as component props.
type Props interface {
	Keys() []string
	Get(key string) (any, bool)
}

// MapProps wraps map[string]any and implements Props.
type MapProps struct{ m map[string]any }

func newMapProps(m map[string]any) MapProps { return MapProps{m} }

func (p MapProps) Keys() []string {
	keys := make([]string, 0, len(p.m))
	for k := range p.m {
		keys = append(keys, k)
	}
	return keys
}

func (p MapProps) Get(key string) (any, bool) {
	v, ok := p.m[key]
	return v, ok
}

// StructProps wraps a dereferenced reflect.Value of kind Struct and implements
// Props lazily — no upfront map allocation.
type StructProps struct{ rv reflect.Value }

// newStructProps dereferences any chain of pointers. Returns false if the
// final kind is not Struct or if a nil pointer is encountered.
func newStructProps(v any) (StructProps, bool) {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return StructProps{}, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return StructProps{}, false
	}
	return StructProps{rv}, true
}

// Keys enumerates canonical keys using the two-pass algorithm: direct
// (non-anonymous) fields first, then anonymous embedded fields. The
// first-rune-lowercase alias is never added to Keys — it is a lookup-only
// affordance.
func (p StructProps) Keys() []string {
	seen := make(map[string]struct{})
	return collectStructKeys(p.rv, seen)
}

// Get resolves a key using three-step lookup:
//  1. Exact json tag match (case-sensitive).
//  2. Exact Go field name match (case-sensitive).
//  3. First-rune-lowercased Go field name match — only when no json tag is present.
//
// Returns nil, true when the resolved field value is a typed nil pointer.
func (p StructProps) Get(key string) (any, bool) {
	return structPropsGet(p.rv, key)
}

// structPropsGet implements the two-pass, three-step field resolution for StructProps.Get.
func structPropsGet(rv reflect.Value, key string) (any, bool) {
	rt := rv.Type()
	// First pass: direct (non-anonymous) fields — higher priority.
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() || f.Anonymous {
			continue
		}
		tag := f.Tag.Get("json")
		// Step 1: exact json tag match.
		if tag != "" {
			parts := strings.SplitN(tag, ",", 2)
			if parts[0] == "-" {
				continue
			}
			if parts[0] != "" {
				if parts[0] == key {
					return fieldVal(rv.Field(i)), true
				}
				// Tag is set but doesn't match — skip steps 2 and 3.
				continue
			}
		}
		// Step 2: exact Go field name match.
		if f.Name == key {
			return fieldVal(rv.Field(i)), true
		}
		// Step 3: first-rune-lowercased Go field name match, only when no json tag.
		if tag == "" && len(f.Name) > 0 {
			alias := string(unicode.ToLower(rune(f.Name[0]))) + f.Name[1:]
			if alias == key {
				return fieldVal(rv.Field(i)), true
			}
		}
	}
	// Second pass: recurse into anonymous (embedded) struct fields.
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.Anonymous {
			continue
		}
		tag := f.Tag.Get("json")
		if tag != "" {
			parts := strings.SplitN(tag, ",", 2)
			if parts[0] == "-" {
				continue
			}
			if parts[0] != "" {
				// Embedded field has an explicit json name — not promoted.
				if parts[0] == key && f.IsExported() {
					return rv.Field(i).Interface(), true
				}
				continue
			}
		}
		// Dereference pointer-to-struct embedded fields.
		fv := rv.Field(i)
		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}
		if fv.Kind() != reflect.Struct {
			continue
		}
		if val, ok := structPropsGet(fv, key); ok {
			return val, ok
		}
	}
	return nil, false
}

// fieldVal returns the interface value of fv, converting typed nil pointers to
// untyped nil so that v-if guards work correctly.
func fieldVal(fv reflect.Value) any {
	if fv.Kind() == reflect.Ptr && fv.IsNil() {
		return nil
	}
	return fv.Interface()
}

// collectStructKeys enumerates canonical keys using the two-pass algorithm.
// seen is used to enforce outer-wins shadowing.
func collectStructKeys(rv reflect.Value, seen map[string]struct{}) []string {
	rt := rv.Type()
	var keys []string

	// Pass 1: direct (non-anonymous, exported) fields.
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() || f.Anonymous {
			continue
		}
		key := structFieldKey(f)
		if key == "" {
			continue // json:"-"
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	// Pass 2: anonymous (embedded) fields.
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.Anonymous {
			continue
		}
		tag := f.Tag.Get("json")
		if tag != "" {
			parts := strings.SplitN(tag, ",", 2)
			if parts[0] == "-" {
				continue // skip entirely
			}
			if parts[0] != "" {
				// Explicit json name: treat as named field, do not recurse.
				if f.IsExported() {
					key := parts[0]
					if _, exists := seen[key]; !exists {
						seen[key] = struct{}{}
						keys = append(keys, key)
					}
				}
				continue
			}
		}
		// Dereference pointer-to-struct embedded fields.
		fv := rv.Field(i)
		ft := f.Type
		if ft.Kind() == reflect.Ptr {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
			ft = ft.Elem()
		}
		if ft.Kind() != reflect.Struct {
			continue
		}
		subKeys := collectStructKeys(fv, seen)
		keys = append(keys, subKeys...)
	}

	return keys
}

// structFieldKey returns the map key for a struct field: the json tag name if
// one is present (and not "-"), otherwise the Go field name. Returns "" when
// the field should be omitted (json:"-").
func structFieldKey(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag != "" {
		parts := strings.SplitN(tag, ",", 2)
		if parts[0] == "-" {
			return ""
		}
		if parts[0] != "" {
			return parts[0]
		}
	}
	return f.Name
}

// toProps converts val into a Props implementation, or returns an error for
// unsupported types. This is the sole dispatch point for all spread operations.
//
// Priority order:
//  1. nil → (nil, nil) — nil spread is a no-op.
//  2. Props identity → return as-is.
//  3. map[string]any → MapProps.
//  4. struct / pointer-to-struct → StructProps (nil pointer → no-op).
//  5. anything else → error.
func toProps(val any) (Props, error) {
	if val == nil {
		return nil, nil
	}
	if p, ok := val.(Props); ok {
		return p, nil
	}
	if m, ok := val.(map[string]any); ok {
		return newMapProps(m), nil
	}
	// Dereference pointer chain; nil pointer is a no-op.
	rv := reflect.ValueOf(val)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, nil // nil pointer spread is a no-op
		}
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		return StructProps{rv}, nil
	}
	return nil, fmt.Errorf("expected map or struct, got %T", val)
}
