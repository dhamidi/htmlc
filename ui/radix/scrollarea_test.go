package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like Select.vue's own tests, this module has no dependency on the root
// htmlc package, so a full render-based test (mounting ScrollArea.vue into a
// real htmlc.Engine and checking rendered HTML) is out of scope here — that
// proof is deliberately deferred to the examples/radix-demo commit, which
// does depend on root htmlc. These are content-sanity checks: they confirm
// the component's source file contains the markers this commit's design
// depends on.
func TestScrollArea_ContainsZeroJSBaseline(t *testing.T) {
	src := readScrollArea(t)
	tpl := scrollAreaTemplateBlock(t, src)

	for _, marker := range []string{
		`class="radix-scroll-area-viewport"`,
		"<slot></slot>",
	} {
		if !strings.Contains(tpl, marker) {
			t.Errorf("ScrollArea.vue <template> missing expected baseline marker %q; template was:\n%s", marker, tpl)
		}
	}

	// The real, functional overflow:auto baseline lives in the scoped
	// style rule, not inline on the template tag (see header comment) —
	// confirm it is there and is genuinely "auto", not "hidden"/"scroll"
	// only, and not accidentally display:none'd out of existence.
	rule := scrollAreaCSSRule(t, src, ".radix-scroll-area-viewport {")
	if !strings.Contains(rule, "overflow: auto") && !strings.Contains(rule, "overflow:auto") {
		t.Errorf(".radix-scroll-area-viewport rule missing expected `overflow: auto`; rule was:\n%s", rule)
	}
	if strings.Contains(rule, "display: none") || strings.Contains(rule, "display:none") {
		t.Errorf(".radix-scroll-area-viewport rule must not use display:none; rule was:\n%s", rule)
	}
}

// TestScrollArea_HidesNativeScrollbarBothVendorTechniques confirms both
// vendor-specific scrollbar-hiding techniques are present: Firefox's
// standard `scrollbar-width: none` and WebKit/Blink's non-standard
// `::-webkit-scrollbar { display: none }` pseudo-element rule. Neither one
// alone covers both engine families — see this file's header comment,
// "Hiding the native scrollbar."
func TestScrollArea_HidesNativeScrollbarBothVendorTechniques(t *testing.T) {
	src := readScrollArea(t)

	viewportRule := scrollAreaCSSRule(t, src, ".radix-scroll-area-viewport {")
	if !strings.Contains(viewportRule, "scrollbar-width: none") && !strings.Contains(viewportRule, "scrollbar-width:none") {
		t.Errorf(".radix-scroll-area-viewport rule missing expected Firefox `scrollbar-width: none`; rule was:\n%s", viewportRule)
	}

	webkitRule := scrollAreaCSSRule(t, src, ".radix-scroll-area-viewport::-webkit-scrollbar {")
	if !strings.Contains(webkitRule, "display: none") && !strings.Contains(webkitRule, "display:none") {
		t.Errorf("::-webkit-scrollbar rule missing expected `display: none`; rule was:\n%s", webkitRule)
	}
}

// TestScrollArea_CustomThumbAndTrackStructure confirms the custom visual
// overlay's markup: an aria-hidden track element containing a thumb
// element, absolutely positioned alongside the viewport (not nested inside
// it — see header comment for why this pair sits as the viewport's own
// sibling).
func TestScrollArea_CustomThumbAndTrackStructure(t *testing.T) {
	src := readScrollArea(t)
	tpl := scrollAreaTemplateBlock(t, src)

	for _, marker := range []string{
		`class="radix-scroll-area-track"`,
		`aria-hidden="true"`,
		`class="radix-scroll-area-thumb"`,
	} {
		if !strings.Contains(tpl, marker) {
			t.Errorf("ScrollArea.vue <template> missing expected thumb/track marker %q; template was:\n%s", marker, tpl)
		}
	}

	// The thumb must be nested inside the track in the markup (the track
	// is the thumb's containing block for its absolute positioning).
	trackIdx := strings.Index(tpl, "radix-scroll-area-track")
	thumbIdx := strings.Index(tpl, "radix-scroll-area-thumb")
	if trackIdx == -1 || thumbIdx == -1 || thumbIdx < trackIdx {
		t.Errorf("ScrollArea.vue <template> thumb must be nested inside track; template was:\n%s", tpl)
	}

	// Both the track and the thumb must be styled position:absolute
	// against the outer .radix-scroll-area wrapper's containing block.
	trackRule := scrollAreaCSSRule(t, src, ".radix-scroll-area-track {")
	if !strings.Contains(trackRule, "position: absolute") {
		t.Errorf(".radix-scroll-area-track rule missing expected `position: absolute`; rule was:\n%s", trackRule)
	}
	thumbRule := scrollAreaCSSRule(t, src, ".radix-scroll-area-thumb {")
	if !strings.Contains(thumbRule, "position: absolute") {
		t.Errorf(".radix-scroll-area-thumb rule missing expected `position: absolute`; rule was:\n%s", thumbRule)
	}

	wrapperRule := scrollAreaCSSRule(t, src, ".radix-scroll-area {")
	if !strings.Contains(wrapperRule, "position: relative") {
		t.Errorf(".radix-scroll-area wrapper rule missing expected `position: relative` (containing block for the absolutely-positioned track); rule was:\n%s", wrapperRule)
	}
}

func TestScrollArea_ContainsScopedStyle(t *testing.T) {
	src := readScrollArea(t)

	if !strings.Contains(src, "<style scoped>") {
		t.Error("ScrollArea.vue missing expected <style scoped> block")
	}
}

func TestScrollArea_ContainsCustomElementEnhancement(t *testing.T) {
	src := readScrollArea(t)

	for _, marker := range []string{
		"<script customelement>",
		// The tag name must be exactly what component.go's real
		// deriveCustomElementTag derives for "radix/ScrollArea.vue"
		// (verified by actually running the algorithm, not guessed — see
		// the file's header comment), matching every sibling component's
		// own Mount{Prefix: "radix"} convention.
		"customElements.define('radix-scroll-area'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("ScrollArea.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestScrollArea_ScrollSyncMarkers confirms the script recomputes the
// thumb's size/position on the viewport's own native 'scroll' event, on
// initial connect, and via a ResizeObserver watching for layout/content
// changes independent of any scroll event — see header comment point 1.
func TestScrollArea_ScrollSyncMarkers(t *testing.T) {
	src := readScrollArea(t)
	script := scrollAreaScriptBlock(t, src)

	for _, marker := range []string{
		"connectedCallback()",
		"addEventListener('scroll', this.#onScroll)",
		"#syncThumb()",
		"ResizeObserver",
		"disconnectedCallback()",
		"removeEventListener('scroll', this.#onScroll)",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("ScrollArea.vue <script customelement> missing expected scroll-sync marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestScrollArea_ThumbSizeAndPositionFormulas is the process-mandated
// adversarial check on the actual size/position math: thumb size must be
// proportional to clientHeight/scrollHeight, and thumb position must be
// proportional to scrollTop/(scrollHeight - clientHeight) applied to the
// thumb's own available travel range. See header comment point 1 for the
// worked hand-trace (scrollHeight=1000/clientHeight=200 -> 20% thumb;
// scrollTop=400 of maxScrollTop=800 -> thumb 50% down its own travel
// range).
func TestScrollArea_ThumbSizeAndPositionFormulas(t *testing.T) {
	src := readScrollArea(t)
	script := scrollAreaScriptBlock(t, src)

	for _, marker := range []string{
		"#syncThumb()",
		"viewport.scrollHeight",
		"viewport.clientHeight",
		"viewport.scrollTop",
		"visibleHeight / contentHeight",
		"contentHeight - visibleHeight",
		"style.height = thumbHeight",
		"style.transform = 'translateY(",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("ScrollArea.vue <script customelement> missing expected thumb size/position marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestScrollArea_DragToScrollMarkers confirms the thumb is made draggable
// via pointerdown/pointermove/pointerup with pointer capture, and that the
// drag math both exists and is the correctly-signed inverse of the
// size/position formula — see header comment point 2.
func TestScrollArea_DragToScrollMarkers(t *testing.T) {
	src := readScrollArea(t)
	script := scrollAreaScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('pointerdown', this.#onThumbPointerDown)",
		"addEventListener('pointermove', this.#onThumbPointerMove)",
		"addEventListener('pointerup', this.#onThumbPointerUp)",
		"setPointerCapture(event.pointerId)",
		"releasePointerCapture(event.pointerId)",
		"#dragStartClientY",
		"#dragStartScrollTop",
		"event.clientY - this.#dragStartClientY",
		"maxScrollTop / maxThumbTravel",
		"this.#dragStartScrollTop + deltaScrollTop",
		"viewport.scrollTop = Math.min(Math.max(nextScrollTop, 0), maxScrollTop)",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("ScrollArea.vue <script customelement> missing expected drag-to-scroll marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestScrollArea_DragDirectionIsNotInverted is the adversarial check
// specifically targeting the "drag down scrolls up" inversion bug this
// commit's process calls out by name: dragging the thumb down (a positive
// clientY delta) must produce a positive scrollTop delta (scroll content
// down), not a negated one.
func TestScrollArea_DragDirectionIsNotInverted(t *testing.T) {
	src := readScrollArea(t)
	script := scrollAreaScriptBlock(t, src)

	// The delta must be added, not subtracted, when deriving the next
	// scrollTop — a `-deltaScrollTop` or `dragStartScrollTop - deltaScrollTop`
	// would silently invert the drag direction.
	if !strings.Contains(script, "this.#dragStartScrollTop + deltaScrollTop") {
		t.Errorf("ScrollArea.vue drag math must add deltaScrollTop to the starting scrollTop (dragging down increases scrollTop); script was:\n%s", script)
	}
	if strings.Contains(script, "this.#dragStartScrollTop - deltaScrollTop") {
		t.Error("ScrollArea.vue drag math appears inverted: subtracting deltaScrollTop would make dragging down scroll content up")
	}

	// The clientY delta itself must not be negated before use.
	if strings.Contains(script, "-(event.clientY - this.#dragStartClientY)") ||
		strings.Contains(script, "this.#dragStartClientY - event.clientY") {
		t.Error("ScrollArea.vue drag math appears to negate the pointer delta, which would invert drag direction")
	}
}

// TestScrollArea_VerticalOnlyScopeDocumented confirms the intentional
// vertical-only v1 scope cut (horizontal scrollbar + corner element) is
// documented in the header comment, matching this batch's "honest,
// bounded scope" convention rather than a silent omission.
func TestScrollArea_VerticalOnlyScopeDocumented(t *testing.T) {
	src := readScrollArea(t)

	header := scrollAreaHeaderComment(t, src)
	for _, marker := range []string{
		"Vertical-only",
		"horizontal",
		"corner",
	} {
		if !strings.Contains(strings.ToLower(header), strings.ToLower(marker)) {
			t.Errorf("ScrollArea.vue header comment missing expected scope-cut documentation mentioning %q; header was:\n%s", marker, header)
		}
	}
}

// TestScrollArea_DoesNotCopyRadixSource is a light textual guard against
// accidentally transcribing identifiers unique to Radix's real, unported
// source (read only, per this commit's process) rather than writing fresh
// code, e.g. its internal state-machine/context-scope plumbing which this
// port has no equivalent of and should never need to name.
func TestScrollArea_DoesNotCopyRadixSource(t *testing.T) {
	src := readScrollArea(t)

	for _, forbidden := range []string{
		"createScrollAreaContext",
		"createScrollAreaScope",
		"useStateMachine",
		"ScrollAreaProvider",
		"getThumbOffsetFromScroll",
		"getScrollPositionFromPointer",
		"linearScale",
		"addUnlinkedScrollListener",
	} {
		if strings.Contains(src, forbidden) {
			t.Errorf("ScrollArea.vue contains %q, an identifier unique to Radix's real (read-only, never-transcribed) source", forbidden)
		}
	}
}

func scrollAreaHeaderComment(t *testing.T, src string) string {
	t.Helper()
	end := strings.Index(src, "-->")
	if end == -1 {
		t.Fatalf("ScrollArea.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	return src[:end]
}

func scrollAreaTemplateBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.Index(src, "-->")
	if afterComment == -1 {
		t.Fatalf("ScrollArea.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<template>")
	end := strings.Index(body, "</template>")
	if start == -1 || end == -1 {
		t.Fatalf("ScrollArea.vue missing <template>...</template> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</template>")]
}

func scrollAreaScriptBlock(t *testing.T, src string) string {
	t.Helper()
	start := strings.Index(src, "<script customelement>")
	end := strings.Index(src, "</script>")
	if start == -1 || end == -1 {
		t.Fatalf("ScrollArea.vue missing <script customelement>...</script> block; source was:\n%s", src)
	}
	return src[start : end+len("</script>")]
}

// scrollAreaCSSRule extracts a single CSS rule's source text (from its
// opening "selector {" marker through the matching closing "}") so
// assertions can be scoped to just that rule. Searches only the portion of
// the source after the header comment's own "-->" closer, since the header
// comment's prose itself quotes CSS-selector-shaped example text (e.g.
// while explaining the scoped-style specificity pitfall this file's
// .radix-scroll-area-viewport rule works around) that would otherwise be
// found first by a naive whole-file search.
func scrollAreaCSSRule(t *testing.T, src, selectorOpen string) string {
	t.Helper()
	afterComment := strings.Index(src, "-->")
	if afterComment == -1 {
		t.Fatalf("ScrollArea.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, selectorOpen)
	if start == -1 {
		t.Fatalf("ScrollArea.vue missing expected CSS rule starting with %q; source was:\n%s", selectorOpen, body)
	}
	end := strings.Index(body[start:], "}")
	if end == -1 {
		t.Fatalf("ScrollArea.vue could not find end of CSS rule %q; source was:\n%s", selectorOpen, body)
	}
	return body[start : start+end+1]
}

func readScrollArea(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/ScrollArea.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/ScrollArea.vue) failed: %v", err)
	}
	return string(data)
}
