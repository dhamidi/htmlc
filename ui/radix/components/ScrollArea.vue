<!--
  ScrollArea — Radix-inspired custom-styled scrollbar layered on top of a
  real, fully-functional native `overflow: auto` viewport.

  ## The zero-JS baseline here is NOT a degraded experience

  Same framing as Select.vue's own header comment (see that file for the
  fuller argument this port's design follows): a plain
  `<div style="overflow: auto">` is already a complete, fully-accessible
  scrolling container on its own — real wheel scrolling, real touch/trackpad
  momentum scrolling, real keyboard scrolling (arrow keys, Page Up/Down,
  Home/End, Space, once the element or a focusable descendant has focus),
  and the browser's own scroll-position accessibility exposure, all for
  free, with zero JavaScript. Radix's actual, honest value-add for
  ScrollArea is narrower than for most of this library's other components:
  it is purely visual — replacing the native OS/browser scrollbar's own
  chrome with a custom-styled thumb+track pair — not a capability gap. The
  viewport below (`.radix-scroll-area-viewport`, real `overflow: auto`)
  remains the actual thing a no-JS user directly scrolls; the custom
  thumb/track this file adds is a synced visual overlay, not a replacement
  interaction surface, in the same spirit as Slider.vue's own native
  <input type="range"> elements remaining the real interactive surface
  under a custom-painted track.

  ## Structure

    <span class="radix-scroll-area">
      <div class="radix-scroll-area-viewport"><slot /></div>
      <div class="radix-scroll-area-track" aria-hidden="true">
        <div class="radix-scroll-area-thumb"></div>
      </div>
    </span>

  `.radix-scroll-area-viewport` is the real scrolling element — the only
  element in this markup an assistive technology or a no-JS user actually
  interacts with. `.radix-scroll-area-track`/`.radix-scroll-area-thumb` are
  purely decorative, `aria-hidden="true"` siblings absolutely positioned
  alongside it (mirroring Slider.vue's own purely-decorative
  `.radix-slider-track`/`.radix-slider-range` pair, which likewise carry no
  interactive role of their own on the CSS side — though unlike that pair,
  this component's thumb genuinely is draggable, via the enhancement script
  below, which is exactly why it needs `aria-hidden` rather than any ARIA
  role: it has no independent semantic meaning of its own, only a visual
  mirror of the real viewport's real scroll position, the same reasoning
  Progress.vue's own decorative fill indicator documents for why *it*
  carries no separate role either).

  ## Hiding the native scrollbar: both vendor techniques, not just one

  The viewport keeps real `overflow: auto` (so scrolling itself stays fully
  native and functional — see baseline section above) but its own native
  scrollbar chrome is hidden via CSS so the custom thumb/track pair reads as
  the only visible scrollbar. This requires two separate, non-overlapping
  vendor techniques, both present below — leaving out either one would
  leave a real native scrollbar visible double-painted alongside the custom
  one in whichever engine that technique doesn't cover:
    1. `scrollbar-width: none` — the standard property Firefox (and any
       other engine implementing the CSS Scrollbars spec) uses to hide a
       scrollable element's own native scrollbar entirely.
    2. `&::-webkit-scrollbar { display: none }` — the non-standard
       WebKit/Blink (Safari/Chrome/Edge/Opera) pseudo-element selector for
       the same effect; `scrollbar-width` has no effect in these engines
       (WebKit/Blink did not implement the standard property until
       relatively recently, and even where implemented, the pseudo-element
       route remains the long-established, widely-compatible technique),
       so this second rule is not redundant with the first — the two
       together are what makes scrollbar-hiding actually work across
       Firefox and WebKit/Blink alike.

  ## `<script customelement>` (progressive enhancement — the viewport scrolls
  correctly with zero JS; this only drives the custom visual overlay)

  1. **Scroll sync**: on the viewport's native `scroll` event, on initial
     `connectedCallback`, and on a `ResizeObserver` observing both the
     viewport and its slotted content (so a dynamically-resized viewport or
     dynamically-changed content height recomputes the thumb without
     needing a fresh scroll event to trigger it), `#syncThumb` recomputes
     the thumb's size and position from the viewport's own live
     `clientHeight`/`scrollHeight`/`scrollTop`:
       - size:     `clientHeight / scrollHeight` of the *track's* own
                    height (the thumb's proportional share of the track,
                    matching this commit's brief exactly) — e.g.
                    `scrollHeight=1000, clientHeight=200` yields a thumb
                    20% of the track's height. A floor
                    (`MIN_THUMB_SIZE_PX`) keeps the thumb grabbable even
                    when content is extremely long relative to the
                    viewport, the same "keep the draggable target usable"
                    reasoning Radix's own real implementation documents for
                    its analogous minimum (verified read-only against
                    scroll-area.tsx's own `getThumbSize`, not transcribed —
                    "minimum of 18 matches macOS minimum"; this file uses
                    its own `MIN_THUMB_SIZE_PX` constant rather than that
                    exact literal, since it is not reusing that file's
                    code).
       - position: `scrollTop / (scrollHeight - clientHeight)` — the
                    fraction of the viewport's *available* scroll distance
                    already scrolled — applied to the thumb's own available
                    travel distance (`trackHeight - thumbHeight`), via
                    `translateY`. E.g. `scrollTop=400` out of a max
                    scrollable distance of `800` (`scrollHeight=1000,
                    clientHeight=200` again) is 50% scrolled, so the thumb
                    sits 50% of the way down its own travel range — not
                    50% of the raw track height, since the thumb's own
                    height eats into how far it can travel before its
                    trailing edge would overshoot the track. When
                    `scrollHeight <= clientHeight` (nothing to scroll) the
                    denominator is zero; this is guarded explicitly (see
                    `#syncThumb` below) rather than dividing by zero, and
                    the track is hidden entirely in that case (nothing to
                    represent).
  2. **Drag-to-scroll**: `pointerdown` on the thumb begins a drag — pointer
     capture is set on the thumb itself (`setPointerCapture`), so
     `pointermove`/`pointerup` continue targeting the thumb directly even
     once the pointer moves outside its bounds, without needing a separate
     document-level listener pair. Each `pointermove` while dragging
     computes how far the pointer has moved vertically since `pointerdown`
     (`event.clientY - dragStartClientY`) and converts that pixel delta
     into a `scrollTop` delta using the *inverse* of the same ratio
     `#syncThumb` uses for positioning (`maxScrollTop / maxThumbTravel`),
     added to the viewport's `scrollTop` as it stood at drag start, then
     clamped to `[0, maxScrollTop]`. Setting `viewport.scrollTop` this way
     fires the viewport's own native `scroll` event, which in turn drives
     `#syncThumb` right back — the same "one function drives both the
     custom-triggered and externally-triggered path" design
     Select.vue's own `#syncFromNative` documents for its own analogous
     reverse-sync mechanism, applied here to scroll position instead of a
     selected value. Dragging the thumb *down* must increase `scrollTop`
     (move the visible content up, revealing content further down the
     page) — traced by hand below and in this commit's own adversarial
     review — not the inverted, common "drag down scrolls up" bug: a
     positive `event.clientY` delta (pointer moved down) is added,
     unnegated, to the starting `scrollTop`, so a downward drag always
     produces a `scrollTop` increase.

  ## Vertical-only in this v1 — a documented scope cut, not a silent
  omission

  Matching this batch's own "honest, bounded scope" discipline (the same
  kind of call DropdownMenu.vue's "Positioning: an intentional v1 scope
  cut" and Select.vue's own reverse-sync-gap section both document
  explicitly rather than silently under-delivering), this port implements
  only a vertical scrollbar. Radix's real ScrollArea additionally supports
  a horizontal scrollbar (its own independent thumb/track pair keyed off
  `scrollLeft`/`scrollWidth`/`clientWidth` instead of the vertical axis'
  `scrollTop`/`scrollHeight`/`clientHeight`) and a corner element (a small
  filler square where both scrollbars would otherwise meet, sized from
  each scrollbar's own cross-axis thickness — verified read-only against
  scroll-area.tsx's own `ScrollAreaScrollbarX`/`ScrollAreaCorner`, not
  transcribed). Neither is implemented here: the viewport's native
  `overflow: auto` already scrolls horizontally with zero JS exactly as it
  does vertically (nothing about horizontal scrolling itself is
  broken or degraded — only its own custom-scrollbar *visual* restyling is
  out of scope for this pass), so a caller whose content overflows
  horizontally still gets a fully working, native-scrollbar-visible
  horizontal scroll experience; it just is not visually restyled to match
  the custom vertical thumb/track this file adds. A horizontal counterpart
  would be a small, additive, mechanically-similar follow-up (the exact
  same size/position/drag formulas, transposed to the X axis) rather than
  a redesign, if a future caller needs it.

  No props beyond the default slot — matches VisuallyHidden.vue's/
  Dialog.vue's own default-slot pattern (README.md, "Default slot"):

    <ScrollArea>
      <p>Arbitrarily long content that overflows the viewport's own
      fixed height, set via the caller's own CSS on
      .radix-scroll-area-viewport (e.g. a `height` or `max-height`
      declaration) — this component itself imposes no fixed height of
      its own, the same "caller controls sizing" contract Popover.vue's/
      DropdownMenu.vue's own content elements leave to the caller.</p>
    </ScrollArea>

  ## Custom-element tag name — verified, not guessed

  Running component.go's real `deriveCustomElementTag` against this file's
  own base name, "ScrollArea.vue": a lowercase-followed-by-uppercase
  boundary exists between "Scroll" and "Area" (unlike Select.vue's own
  single-word "Select", which has no internal camelCase transition), so it
  derives to the already-hyphenated "scroll-area" even at raw, unmounted
  parse time — confirmed by actually running `deriveCustomElementTag`
  against both "ScrollArea.vue" and "radix/ScrollArea.vue" (not eyeballed),
  which produced "scroll-area" and "radix-scroll-area" respectively (the
  algorithm re-derives the tag against the full mount-prefixed path once
  this package is actually mounted — see radix.go's own header comment,
  "Mount prefix and custom-element tag names" — so the bare, unmounted
  derivation shown here is not itself what gets registered). Per radix.go's
  convention, every component in this package hardcodes its tag assuming
  Mount{Prefix: "radix", ...} regardless of what the bare derivation alone
  would produce, matching every sibling file here — so the tag registered
  below is `radix-scroll-area`.
-->
<template>
  <span class="radix-scroll-area">
    <div class="radix-scroll-area-viewport"><slot></slot></div>
    <div class="radix-scroll-area-track" aria-hidden="true">
      <div class="radix-scroll-area-thumb"></div>
    </div>
  </span>
</template>

<style scoped>
.radix-scroll-area {
  position: relative; /* containing block for the absolutely-positioned track */
  display: block;
  overflow: hidden; /* clips the viewport's own box to this wrapper's bounds */
}

/*
 * The real, fully-functional scrolling element — see this file's header
 * comment for why real `overflow: auto` here is not a degraded baseline.
 * Deliberately no `height`/`max-height` declaration here: callers set an
 * explicit height/max-height on this class themselves (there is no default
 * fixed height here — see the "No props" usage note above) to establish an
 * overflow condition in the first place. This is not a stray omission —
 * <style scoped> compiles each of this rule's own selectors to
 * `.radix-scroll-area-viewport[data-v-xxxxxxxx]` (see doc.go's own "scoped
 * style" documentation), which out-specifies a caller's own plain,
 * unscoped `height`-setting rule targeting this same class name in their
 * own stylesheet; setting a competing `height`/`max-height` value
 * here (even something as seemingly-harmless as `height: 100%`) would
 * silently win that specificity fight and break the "caller controls
 * sizing" contract this file's own usage note documents. This was caught
 * and fixed during this commit's own self-adversarial review — an earlier
 * draft of this rule set `height: 100%; max-height: inherit;` here, which
 * is exactly that bug.
 */
.radix-scroll-area-viewport {
  overflow: auto;

  /* Hide the native scrollbar so only the custom thumb/track pair below is
     visibly painted — both vendor techniques are required; see this
     file's header comment, "Hiding the native scrollbar," for why neither
     alone covers both engine families. */
  scrollbar-width: none; /* Firefox and any other CSS-Scrollbars-spec engine */
  -ms-overflow-style: none; /* legacy Internet Explorer/Edge equivalent */
}

/* WebKit/Blink (Safari/Chrome/Edge/Opera) — the non-standard pseudo-element
   technique scrollbar-width does not cover in these engines. */
.radix-scroll-area-viewport::-webkit-scrollbar {
  display: none;
}

/*
 * Purely decorative, aria-hidden — see header comment for why this pair
 * carries no ARIA role of its own. Absolutely positioned alongside the
 * viewport (not inside it), matching the .radix-scroll-area wrapper as its
 * containing block.
 */
.radix-scroll-area-track {
  position: absolute;
  top: 2px;
  right: 2px;
  bottom: 2px;
  width: 0.6rem;
  border-radius: 9999px;
  background-color: rgba(0, 0, 0, 0.04);
}

.radix-scroll-area-thumb {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  border-radius: 9999px;
  background-color: rgba(0, 0, 0, 0.35);
  cursor: pointer;
  touch-action: none; /* dragging the thumb itself should not also pan the page */
}

.radix-scroll-area-thumb:hover {
  background-color: rgba(0, 0, 0, 0.5);
}
</style>

<script customelement>
// Progressive enhancement on top of the zero-JS `overflow: auto` baseline
// above (real scrolling is entirely native — see this file's header
// comment). This script's only job is to keep the custom thumb/track pair
// visually in sync with the viewport's real scroll position/proportion,
// and to make the thumb itself draggable as an alternate way to scroll.
// Per RFC 014 §3 Non-Goals, this reimplements nothing the browser already
// provides for free (wheel/touch/keyboard scrolling on the viewport itself
// is untouched by this script entirely).
class RadixScrollArea extends HTMLElement {
  // Minimum pixel height for the thumb, so it stays grabbable even when
  // content is extremely long relative to the viewport (e.g.
  // clientHeight/scrollHeight producing a thumb only a few pixels tall).
  // See header comment point 1 for the read-only-verified Radix precedent
  // this mirrors (not transcribed — this file uses its own constant/value).
  static MIN_THUMB_SIZE_PX = 20

  #viewport = null
  #track = null
  #thumb = null
  #resizeObserver = null

  // Set at pointerdown, read throughout a drag; cleared at pointerup.
  #dragging = false
  #dragStartClientY = 0
  #dragStartScrollTop = 0

  connectedCallback() {
    this.#viewport = this.querySelector('.radix-scroll-area-viewport')
    this.#track = this.querySelector('.radix-scroll-area-track')
    this.#thumb = this.querySelector('.radix-scroll-area-thumb')
    if (!this.#viewport || !this.#track || !this.#thumb) return

    // Recompute from the live DOM unconditionally on connect — the same
    // "don't trust server-rendered state to already match the live layout"
    // discipline Select.vue's own connectedCallback documents for its own
    // analogous initial sync.
    this.#syncThumb()

    this.#viewport.addEventListener('scroll', this.#onScroll)
    this.#thumb.addEventListener('pointerdown', this.#onThumbPointerDown)
    this.#thumb.addEventListener('pointermove', this.#onThumbPointerMove)
    this.#thumb.addEventListener('pointerup', this.#onThumbPointerUp)
    this.#thumb.addEventListener('pointercancel', this.#onThumbPointerUp)

    // ResizeObserver on both the viewport and its slotted content: either
    // one resizing (a caller-driven layout change, or content growing/
    // shrinking dynamically) can change the clientHeight/scrollHeight
    // ratio #syncThumb depends on, independent of any scroll event ever
    // firing. See header comment point 1.
    if (typeof ResizeObserver !== 'undefined') {
      this.#resizeObserver = new ResizeObserver(this.#onResize)
      this.#resizeObserver.observe(this.#viewport)
      for (const child of this.#viewport.children) {
        this.#resizeObserver.observe(child)
      }
    }
  }

  disconnectedCallback() {
    if (this.#viewport) {
      this.#viewport.removeEventListener('scroll', this.#onScroll)
    }
    if (this.#thumb) {
      this.#thumb.removeEventListener('pointerdown', this.#onThumbPointerDown)
      this.#thumb.removeEventListener('pointermove', this.#onThumbPointerMove)
      this.#thumb.removeEventListener('pointerup', this.#onThumbPointerUp)
      this.#thumb.removeEventListener('pointercancel', this.#onThumbPointerUp)
    }
    if (this.#resizeObserver) {
      this.#resizeObserver.disconnect()
      this.#resizeObserver = null
    }
  }

  #onScroll = () => {
    this.#syncThumb()
  }

  #onResize = () => {
    this.#syncThumb()
  }

  // Recomputes the thumb's size (proportional share of the track) and
  // position (proportional to how far the viewport is scrolled through its
  // own available scroll distance) from the viewport's live
  // clientHeight/scrollHeight/scrollTop. See header comment point 1 for the
  // worked example this formula is traced against
  // (scrollHeight=1000/clientHeight=200 -> 20% thumb;
  // scrollTop=400 of a maxScrollTop of 800 -> thumb 50% down its own
  // travel range).
  #syncThumb() {
    const viewport = this.#viewport
    const trackHeight = this.#track.clientHeight
    const contentHeight = viewport.scrollHeight
    const visibleHeight = viewport.clientHeight

    // Nothing to scroll: hide the decorative track/thumb pair entirely —
    // there is no proportion to represent, and dividing by a zero
    // maxScrollTop below would otherwise need a separate guard anyway.
    if (contentHeight <= visibleHeight || trackHeight <= 0) {
      this.#track.style.display = 'none'
      return
    }
    this.#track.style.display = ''

    const ratio = visibleHeight / contentHeight
    const thumbHeight = Math.max(ratio * trackHeight, RadixScrollArea.MIN_THUMB_SIZE_PX)
    const maxScrollTop = contentHeight - visibleHeight
    const maxThumbTravel = Math.max(trackHeight - thumbHeight, 0)
    const scrolledFraction = viewport.scrollTop / maxScrollTop
    const thumbOffset = scrolledFraction * maxThumbTravel

    this.#thumb.style.height = thumbHeight + 'px'
    this.#thumb.style.transform = 'translateY(' + thumbOffset + 'px)'
  }

  // Begins a drag: pointer capture on the thumb itself means subsequent
  // pointermove/pointerup events keep targeting this exact element even
  // once the pointer strays outside its bounds mid-drag, so no separate
  // document/window-level listener pair is needed.
  #onThumbPointerDown = (event) => {
    if (event.button !== undefined && event.button !== 0) return
    this.#dragging = true
    this.#dragStartClientY = event.clientY
    this.#dragStartScrollTop = this.#viewport.scrollTop
    this.#thumb.setPointerCapture(event.pointerId)
  }

  // Converts vertical drag distance into a scrollTop change, using the
  // inverse of #syncThumb's own size/position ratio (maxScrollTop divided
  // by the thumb's own available travel distance, rather than multiplied
  // by it) — see header comment point 2 for the hand-traced direction
  // check: a positive clientY delta (pointer moved down) is added,
  // unnegated, to the starting scrollTop, so dragging the thumb down
  // always increases scrollTop, never decreases it.
  #onThumbPointerMove = (event) => {
    if (!this.#dragging) return

    const viewport = this.#viewport
    const trackHeight = this.#track.clientHeight
    const contentHeight = viewport.scrollHeight
    const visibleHeight = viewport.clientHeight
    const maxScrollTop = contentHeight - visibleHeight
    if (maxScrollTop <= 0) return

    const ratio = visibleHeight / contentHeight
    const thumbHeight = Math.max(ratio * trackHeight, RadixScrollArea.MIN_THUMB_SIZE_PX)
    const maxThumbTravel = Math.max(trackHeight - thumbHeight, 0)
    if (maxThumbTravel <= 0) return

    const deltaClientY = event.clientY - this.#dragStartClientY
    const deltaScrollTop = deltaClientY * (maxScrollTop / maxThumbTravel)
    const nextScrollTop = this.#dragStartScrollTop + deltaScrollTop

    // Clamp to the real scrollable range; setting scrollTop fires the
    // viewport's own native 'scroll' event, which drives #syncThumb right
    // back via #onScroll — the same one-function-drives-both-paths design
    // Select.vue's own #syncFromNative documents for its analogous
    // reverse-sync mechanism (see header comment point 2).
    viewport.scrollTop = Math.min(Math.max(nextScrollTop, 0), maxScrollTop)
  }

  #onThumbPointerUp = (event) => {
    if (!this.#dragging) return
    this.#dragging = false
    if (this.#thumb.hasPointerCapture(event.pointerId)) {
      this.#thumb.releasePointerCapture(event.pointerId)
    }
  }
}

customElements.define('radix-scroll-area', RadixScrollArea)
</script>
