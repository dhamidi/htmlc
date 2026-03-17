package htmlc

import (
	"encoding/json"

	"golang.org/x/net/html"
)

// debugAttrOrder defines the fixed emission order of data-htmlc-* debug
// attributes. Attributes are always emitted in this order, appended after all
// existing element attributes on the component root element.
var debugAttrOrder = []string{
	"data-htmlc-component",
	"data-htmlc-file",
	"data-htmlc-props",
	"data-htmlc-props-error",
}

// propsToJSON serialises props through the Props interface into a JSON object.
// Nested Props values are recursed into and emitted as nested JSON objects.
// If p is nil, the returned bytes represent an empty JSON object "{}".
func propsToJSON(p Props) ([]byte, error) {
	if p == nil {
		return []byte("{}"), nil
	}
	keys := p.Keys()
	m := make(map[string]any, len(keys))
	for _, k := range keys {
		v, _ := p.Get(k)
		if nested, ok := v.(Props); ok {
			raw, err := propsToJSON(nested)
			if err != nil {
				return nil, err
			}
			m[k] = json.RawMessage(raw)
		} else {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

// isFragmentTemplate reports whether the component template has no single
// element root — either it has multiple element children or only text nodes.
// Fragment templates cannot carry debug attributes and are silently skipped.
func isFragmentTemplate(tmpl *html.Node) bool {
	var count int
	for c := tmpl.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			count++
			if count > 1 {
				return true
			}
		}
	}
	return count == 0
}
