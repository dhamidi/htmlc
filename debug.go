package htmlc

import (
	"fmt"
	"io"
)

// debugWriter wraps an io.Writer and provides helpers for emitting structured
// HTML debug comments. It is used by Renderer when Debug mode is active.
type debugWriter struct {
	w     io.Writer
	depth int
}

func newDebugWriter(w io.Writer) *debugWriter { return &debugWriter{w: w} }

func (d *debugWriter) Write(p []byte) (int, error) { return d.w.Write(p) }

func (d *debugWriter) comment(format string, args ...any) {
	fmt.Fprintf(d.w, "<!-- [htmlc:debug] %s -->\n", fmt.Sprintf(format, args...))
}

func (d *debugWriter) componentStart(name, file string) {
	d.comment("component=%s file=%s", name, file)
	d.depth++
}

func (d *debugWriter) componentEnd(name string) {
	d.depth--
	d.comment("/component=%s", name)
}

func (d *debugWriter) exprValue(expr string, value any) {
	d.comment("expr=%q value=%v", expr, value)
}

func (d *debugWriter) vifSkipped(expr string, result bool) {
	d.comment("v-if=%q \u2192 %v: node skipped", expr, result)
}

func (d *debugWriter) slotStart(name string, nodeCount int) {
	d.comment("slot=%s nodes=%d", name, nodeCount)
}

func (d *debugWriter) slotEnd(name string) {
	d.comment("/slot=%s", name)
}
