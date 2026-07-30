// Package turbo provides small, zero-dependency helpers for working with
// Turbo (https://turbo.hotwired.dev) request/response headers and the
// <turbo-stream> response format from ordinary net/http handlers.
//
// Every function in this package operates purely on *http.Request,
// io.Writer, and strings — it has no dependency on htmlc or any
// third-party package, so importing it never pulls in anything beyond the
// Go standard library.
//
// This package covers Turbo's request/response shapes only: Drive
// (WantsStream), Frames (FrameID), and single-response Streams
// (WriteStream). Turbo's WebSocket-based real-time broadcast path
// (<turbo-stream-source>, streaming the same update to multiple
// already-connected clients) is intentionally out of scope — see RFC 014
// §3 and §4.6.
package turbo

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
)

// StreamContentType is the MIME type Turbo Streams responses must be served
// with (https://turbo.hotwired.dev/handbook/streams). Turbo Drive and Turbo
// Frames send this value in their Accept header to indicate they can accept
// a <turbo-stream> response in place of a full HTML document.
const StreamContentType = "text/vnd.turbo-stream.html"

// WantsStream reports whether r's Accept header indicates the client wants
// a Turbo Stream response.
//
// A real Accept header is a comma-separated list of media types, each
// optionally followed by parameters (e.g. ";q=0.9"), such as
// "text/html, text/vnd.turbo-stream.html, application/xhtml+xml" — the
// value Turbo Drive/Frames actually send alongside its other accepted
// types. WantsStream splits the header on commas, strips any
// parameters, and compares each resulting media type against
// StreamContentType case-insensitively, rather than testing whether
// StreamContentType appears anywhere in the raw header value: a plain
// substring search risks a false positive against an unrelated media type
// that merely contains StreamContentType as part of a longer token (e.g.
// a hypothetical vendor type or parameter value), and it cannot correctly
// ignore per-type parameters that legitimately follow a match. Parsing the
// list into distinct media types avoids both problems.
func WantsStream(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if accept == "" {
		return false
	}
	for _, part := range strings.Split(accept, ",") {
		part = strings.TrimSpace(part)
		if i := strings.IndexByte(part, ';'); i != -1 {
			part = part[:i]
		}
		part = strings.TrimSpace(part)
		if strings.EqualFold(part, StreamContentType) {
			return true
		}
	}
	return false
}

// FrameID reports the value of the "Turbo-Frame" request header, which
// Turbo Frames sets to the requesting frame's id attribute on any
// navigation originating from within a <turbo-frame>
// (https://turbo.hotwired.dev/handbook/frames).
//
// FrameID follows Go's standard "value, ok" idiom, distinguishing the
// header's absence from its presence: ok is false only when the header is
// not sent at all. A request that sends "Turbo-Frame:" with an empty value
// is treated as present-but-empty — ok is true and id is "" — exactly as a
// map lookup or http.Header.Values would distinguish a key mapped to a zero
// value from a key that was never set. Turbo itself never sends the header
// with an empty value in practice, so this distinction mainly protects a
// caller from misreading a deliberately-sent empty header as "no frame
// request" when it is, in fact, a (malformed or synthetic) frame request.
func FrameID(r *http.Request) (id string, ok bool) {
	values := r.Header.Values("Turbo-Frame")
	if len(values) == 0 {
		return "", false
	}
	return values[0], true
}

// WriteStream writes a single well-formed <turbo-stream> element to w:
//
//	<turbo-stream action="ACTION" target="TARGET"><template>FRAGMENT</template></turbo-stream>
//
// matching Turbo Streams' real, documented markup
// (https://turbo.hotwired.dev/handbook/streams). The <template> wrapper
// around fragmentHTML is load-bearing, not optional: content inside a
// <template> element is inert HTML — it is parsed but never rendered or
// executed — until Turbo's own JavaScript moves it into the live DOM under
// target using action. Omitting the wrapper would make fragmentHTML render
// immediately wherever the raw response happens to be inserted, defeating
// the whole mechanism.
//
// action and target are passed through as plain strings rather than
// restricted to an enum, so WriteStream keeps working with a future Turbo
// action this package's v1 doesn't need to know about by name. Turbo's own
// well-known action values, as of this writing, are:
//
//   - "append"  — insert fragmentHTML after target's last child
//   - "prepend" — insert fragmentHTML before target's first child
//   - "replace" — replace target itself with fragmentHTML
//   - "update"  — replace target's children with fragmentHTML
//   - "remove"  — remove target (fragmentHTML is ignored by Turbo for this action)
//   - "before"  — insert fragmentHTML immediately before target
//   - "after"   — insert fragmentHTML immediately after target
//
// action and target are escaped for safe use inside a double-quoted HTML
// attribute; fragmentHTML is written verbatim, since it is itself markup
// destined for the DOM, not attribute text.
//
// WriteStream performs no validation that fragmentHTML actually contains a
// matching `<turbo-frame id="target">` (or anything else) — it is a pure,
// unbuffered formatter. Validating the response shape would require
// buffering and re-parsing fragmentHTML, which conflicts with the
// streaming-friendly design used elsewhere in htmlc; Turbo itself performs
// no such validation either (RFC 014 §10 item 8).
//
// Two independent WriteStream calls into the same w concatenate cleanly
// with no special framing required between them — each call writes one
// complete, self-contained <turbo-stream> element — matching the pattern
// used to deliver multiple actions in a single response (RFC 014 §6
// Example 5).
func WriteStream(w io.Writer, action, target, fragmentHTML string) error {
	if _, err := fmt.Fprintf(w, `<turbo-stream action="%s" target="%s"><template>`,
		html.EscapeString(action), html.EscapeString(target)); err != nil {
		return err
	}
	if _, err := io.WriteString(w, fragmentHTML); err != nil {
		return err
	}
	_, err := io.WriteString(w, `</template></turbo-stream>`)
	return err
}
