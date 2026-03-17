package htmlc

import _ "embed"

// InspectorScript is the source of the htmlc on-page inspector tool.
// It is a self-contained JavaScript module that, when executed, registers
// and appends the <htmlc-inspector> custom element to the document body.
//
// The script is automatically injected before </body> by RenderPage and
// RenderPageContext when the engine is created with Options{Debug: true}.
//
//go:embed htmlc-inspector.js
var InspectorScript string
