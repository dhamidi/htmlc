// Package datastar wires htmlc's [htmlc.Engine.RenderFragmentSession] to
// Datastar's (https://data-star.dev) SSE-driven DOM-patching protocol.
//
// Unlike its two siblings (github.com/dhamidi/htmlc/hypermedia/htmx and
// github.com/dhamidi/htmlc/hypermedia/turbo, both zero-dependency), this
// module has a hard dependency on both the root github.com/dhamidi/htmlc
// module (for *htmlc.Engine / *htmlc.RenderSession, RFC 014 §4.5) and the
// third-party github.com/starfederation/datastar-go module — because
// Datastar's entire backend contract is a single long-lived
// text/event-stream response over which the server emits zero-to-many
// datastar-patch-elements SSE events (RFC 014 §1.3), and driving that
// protocol correctly requires datastar-go's own
// *datastar.ServerSentEventGenerator. A project that only uses htmx or
// Turbo never imports this module and so never resolves datastar-go's
// dependency graph at all (RFC 014 §4.6).
//
// Note on the package-name collision: datastar-go's own SSE package is
// itself named "datastar" (github.com/starfederation/datastar-go/datastar),
// the same name as this package. This file imports it under the alias
// "dsdatastar" to keep the two unambiguous — matching the alias RFC 014 §6
// Example 6 itself uses for the same reason. Downstream callers importing
// both this package and datastar-go's will need their own alias for at
// least one of the two; dsdatastar is the suggested convention.
package datastar

import (
	"context"
	"strings"

	htmlc "github.com/dhamidi/htmlc"
	dsdatastar "github.com/starfederation/datastar-go/datastar"
)

// PatchElementsFragment renders name via engine.RenderFragmentSession into an
// internal buffer, using sess to carry style/script deduplication state
// across the life of the SSE connection sse is attached to (RFC 014 §4.5),
// and then sends the rendered HTML to the client as a single
// "datastar-patch-elements" SSE event via sse.PatchElements.
//
// This combines Engine.RenderFragmentSession (cross-call <style scoped>/
// custom-element-script dedup for a long-lived connection) with datastar-go's
// own PatchElements in one call, so a caller driving a Datastar SSE loop
// (RFC 014 §4.6, §6 Example 6) does not have to hand-write the
// buffer-then-patch wiring itself: every tick becomes one
// PatchElementsFragment call instead of a render-into-buffer step followed by
// a separate sse.PatchElements call.
//
// selector, when non-empty, is passed to datastar-go as
// dsdatastar.WithSelector(selector), targeting the patch at that CSS
// selector. When selector is empty, PatchElementsFragment does not pass
// WithSelector at all, leaving Datastar to fall back to its own default
// targeting mode (matching by the patched element's own id attribute) —
// datastar-go's PatchElements only ever emits a "selector" data line when its
// internal Selector option is non-empty (confirmed by reading
// datastar-go@v1.2.2's elements.go), so calling WithSelector("") would in
// practice be a no-op identical to omitting the option; PatchElementsFragment
// still guards on selector != "" explicitly rather than relying on that
// implementation detail, so the intent — "no selector means default
// targeting" — is visible at this call site rather than depending on
// datastar-go's internals not changing.
//
// If rendering name fails (for example, name references an unknown
// component), PatchElementsFragment returns that error immediately and never
// calls sse.PatchElements — no SSE event is sent for a failed tick, so a
// caller retrying the same *htmlc.RenderSession on the next tick never
// observes a corrupted or partial "datastar-patch-elements" event on the
// wire. A subsequent error from sse.PatchElements itself (for example, the
// connection having been closed by the client) is returned as-is.
func PatchElementsFragment(sse *dsdatastar.ServerSentEventGenerator, engine *htmlc.Engine, sess *htmlc.RenderSession, ctx context.Context, name string, data map[string]any, selector string) error {
	var buf strings.Builder
	if err := engine.RenderFragmentSession(ctx, &buf, name, data, sess); err != nil {
		return err
	}

	opts := make([]dsdatastar.PatchElementOption, 0, 1)
	if selector != "" {
		opts = append(opts, dsdatastar.WithSelector(selector))
	}

	return sse.PatchElements(buf.String(), opts...)
}
