package htmlc

import "io"

// countingWriter wraps an io.Writer and counts bytes written.
// It is embedded by value in Renderer and Engine so no heap allocation
// is needed per component dispatch.
type countingWriter struct {
	w io.Writer
	n int64
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.n += int64(n)
	return n, err
}

// Reset reinitialises the counter and redirects writes to w.
func (cw *countingWriter) Reset(w io.Writer) {
	cw.w = w
	cw.n = 0
}

// MsgComponentRendered is the slog message emitted at LevelDebug when a
// component renders successfully.
const MsgComponentRendered = "component rendered"

// MsgComponentFailed is the slog message emitted at LevelError when a
// component render fails.
const MsgComponentFailed = "component render failed"
