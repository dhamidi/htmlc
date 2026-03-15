package htmlc

import (
	"encoding/json"
	"testing"
)

// FuzzScopeCSS verifies that ScopeCSS never panics on arbitrary CSS and scope
// attribute strings. Errors are not applicable since ScopeCSS returns a string.
func FuzzScopeCSS(f *testing.F) {
	f.Add("p { color: red }", "[data-v-abcdef01]")
	f.Add(".card > .body { margin: 0 }", "[data-v-abcdef01]")
	f.Add("@media (max-width:600px){ .x{} }", "[data-v-abcdef01]")
	f.Add("", "[data-v-abcdef01]")
	f.Add("p{}", "")
	f.Add("a, b, c { display: block }", "[data-v-00000001]")
	f.Add("::before { content: '' }", "[data-v-abcdef01]")
	f.Add("@keyframes spin { 0%{} 100%{} }", "[data-v-abcdef01]")
	f.Add("h1 h2 h3 { font-weight: bold }", "[data-v-abcdef01]")
	f.Add(".a .b > .c + .d ~ .e { }", "[data-v-abcdef01]")
	f.Add("/* comment */ p { color: blue }", "[data-v-abcdef01]")
	f.Add("p { color: red", "[data-v-abcdef01]") // unclosed brace
	f.Add("{{{", "[data-v-abcdef01]")            // malformed CSS
	f.Add("p { color: red }", "")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, css, scopeAttr string) {
		// Must not panic.
		out := ScopeCSS(css, scopeAttr)
		_ = out
	})
}

// FuzzParseFile verifies that ParseFile never panics on arbitrary .vue source
// content. Parse errors are acceptable; crashes are not.
func FuzzParseFile(f *testing.F) {
	f.Add("<template><div>hello</div></template>")
	f.Add("<template><div v-if=\"x\">a</div></template>")
	f.Add("<template></template><script>export default {}</script>")
	f.Add("<template></template><style scoped>p{}</style>")
	f.Add("")
	f.Add("<template><ul><li v-for=\"item in items\">{{ item }}</li></ul></template>")
	f.Add("<template><div :class=\"cls\">{{ msg }}</div></template>")
	f.Add("<template><MyComp :prop=\"val\" /></template>")
	f.Add("<template></template><script>export default { props: ['name'] }</script><style scoped>.x{}</style>")
	f.Add("<template><div v-show=\"visible\">shown</div></template>")
	f.Add("not a vue file at all")
	f.Add("<template><")        // truncated
	f.Add("<<>>{{")            // malformed
	f.Add("<style scoped></style>") // missing template

	f.Fuzz(func(t *testing.T, src string) {
		// Must not panic. Parse errors are acceptable.
		c, err := ParseFile("fuzz.vue", src)
		_, _ = c, err
	})
}

// FuzzRenderString verifies that RenderString never panics when given arbitrary
// template bodies and scope data. Parse or render errors are acceptable.
func FuzzRenderString(f *testing.F) {
	f.Add("<div>{{ name }}</div>", `{"name":"world"}`)
	f.Add("<ul><li v-for=\"x in items\">{{ x }}</li></ul>", `{"items":["a","b"]}`)
	f.Add("<div v-if=\"show\">yes</div>", `{"show":true}`)
	f.Add("<div>{{ a + b }}</div>", `{"a":1,"b":2}`)
	f.Add("<span :class=\"cls\">text</span>", `{"cls":"active"}`)
	f.Add("<div v-show=\"flag\">visible</div>", `{"flag":false}`)
	f.Add("<p>{{ msg }}</p>", `{"msg":"hello"}`)
	f.Add("<div>{{ items.length }}</div>", `{"items":[1,2,3]}`)
	f.Add("<div v-if=\"x\">a</div><div v-else>b</div>", `{"x":false}`)
	f.Add("", `{}`)
	f.Add("<div>{{ </div>", `{}`)            // broken expression
	f.Add("<div v-for=\"\">x</div>", `{}`)   // empty v-for
	f.Add("{{ a.b.c.d.e }}", `{}`)          // deep missing prop

	f.Fuzz(func(t *testing.T, tmpl, scopeJSON string) {
		var scope map[string]any
		if err := json.Unmarshal([]byte(scopeJSON), &scope); err != nil {
			// Invalid JSON scope: skip (not the system under test).
			t.Skip()
		}
		src := "<template>" + tmpl + "</template>"
		c, err := ParseFile("fuzz.vue", src)
		if err != nil {
			return // parse errors are fine
		}
		out, err := RenderString(c, scope)
		_, _ = out, err
		// Must not panic.
	})
}
