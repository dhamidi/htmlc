// Package htmx provides small, zero-dependency helpers for working with
// htmx (https://htmx.org) request and response headers from ordinary
// net/http handlers.
//
// Every function in this package operates purely on *http.Request and
// http.ResponseWriter — it has no dependency on htmlc or any third-party
// package, so importing it never pulls in anything beyond the Go standard
// library.
//
// This package covers the common-case htmx header contract, not htmx's full
// response-header surface (e.g. HX-Push-Url, HX-Redirect, HX-Retarget,
// HX-Reswap, and the JSON-payload form of HX-Trigger are intentionally out
// of scope for v1). A caller needing one of those can set the header
// directly via w.Header().Set, exactly as this package's own SetTrigger
// does internally.
package htmx

import "net/http"

// IsHTMXRequest reports whether r was issued by htmx.
//
// htmx sets the request header "HX-Request: true" on every AJAX request it
// issues (https://htmx.org/docs/#request-headers). Because htmx only ever
// sends the literal value "true" and never sends the header with any other
// value, any other value (or the header's absence) is treated as "not an
// htmx request".
func IsHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// IsBoosted reports whether r originated from an element enhanced with
// htmx's hx-boost attribute.
//
// htmx sets the request header "HX-Boosted: true" when the request was
// triggered by a boosted link or form
// (https://htmx.org/docs/#request-headers, https://htmx.org/attributes/hx-boost/).
// As with IsHTMXRequest, htmx only ever sends the literal value "true", so
// any other value or the header's absence means the request was not
// boosted.
func IsBoosted(r *http.Request) bool {
	return r.Header.Get("HX-Boosted") == "true"
}

// SetTrigger sets the "HX-Trigger" response header to event, instructing
// htmx to trigger a client-side DOM event with that name after the response
// is processed (https://htmx.org/headers/hx-trigger/).
//
// SetTrigger covers the common case of a single, bare event name. htmx's
// HX-Trigger header also supports a comma-separated list of event names, and
// a JSON object mapping event names to detail payloads (e.g.
// `{"eventName": {"detail": "value"}}`) so listeners can read data off the
// event. Callers needing either of those forms should call
// w.Header().Set("HX-Trigger", ...) directly with their own fully
// constructed value instead of using this function.
func SetTrigger(w http.ResponseWriter, event string) {
	w.Header().Set("HX-Trigger", event)
}
