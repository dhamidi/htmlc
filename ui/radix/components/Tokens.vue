<!--
  Tokens — pure CSS-custom-property carrier for this package's shared
  design tokens (--radix-sand-*/--radix-brown-*/--radix-ruby-*/
  --radix-radius-*/--radix-space-*) plus one shared utility class,
  .radix-visually-hidden-input. It has no template content of its own:
  `<template></template>` is empty on purpose, because this component
  exists solely to contribute a <style> block to the page — it renders no
  markup, so an empty template is the correct, honest shape for it, not a
  placeholder waiting to be filled in.

  Every other component in this package (see each file's own <template>)
  instantiates <Tokens></Tokens> as a child, once, near the top of its own
  markup. This is deliberate and is what actually makes the shared block
  reach the final rendered page exactly once no matter how many of this
  package's components a given page mounts — see style.go's
  StyleCollector.Add for the precise mechanism: two StyleContributions
  dedupe when their (ScopeID, CSS) pair matches exactly, where CSS is a
  component's *entire* <style> block text. A single shared block pasted
  into N different components' own <style> sections alongside each
  component's own distinct rules does NOT dedupe under that rule, because
  the surrounding text differs per file, so the combined block text is
  never byte-identical across files (this package's previous rollout made
  exactly that mistake — confirmed by rendering the real demo and counting
  32 separate copies of the block in the output). Routing every consumer
  through one shared *child* component sidesteps that entirely: Tokens.vue
  is a single file with a single, fixed <style> block, so every render of
  it — regardless of which parent mounts it, or how many different parents
  do — contributes the exact same (ScopeID, CSS) pair, which is exactly
  the case StyleCollector.Add already collapses to one entry. Verified
  directly (not assumed) with a throwaway fstest.MapFS-based render before
  adopting this pattern package-wide: a two-component page each mounting a
  shared empty-template token component produced exactly one copy of the
  shared block in the rendered output, with each component's own distinct
  rule intact and separate.

  This file is therefore the single canonical source of truth for the
  token values themselves — edit them here, not in any consuming
  component's own <style> section (none of them carry a copy anymore; see
  radix.go's own header comment, "Design tokens", for the cross-reference).
-->
<template></template>

<style>
/*
 * ui/radix design tokens — an earthy palette built from real Radix Colors
 * scales (@radix-ui/colors, sand/brown/ruby, light mode, fetched from the
 * published package's own CSS source, not guessed), chosen deliberately
 * over the library's own default gray/blue/red trio: sand (a warm neutral)
 * replaces gray for surfaces/text, brown (a coffee/terracotta accent)
 * replaces blue for primary/interactive/focus, and ruby (a warm rose-red)
 * replaces plain red for danger/destructive — still unambiguously "this is
 * dangerous," just warmer than a stock red. Paired with Radix Themes'
 * default "medium" radius scale / scaling=1 spacing scale (fetched from
 * @radix-ui/themes/.../tokens/{radius,space}.css). Prefixed --radix-*
 * (unlike upstream's own unprefixed --sand-1/--brown-9/--radius-3/--space-4)
 * because this is a mounted library, not an app shell: an unprefixed
 * custom property would leak into, and could collide with, a consuming
 * project's own global CSS custom properties of the same generic name.
 *
 * Every other component in this package instantiates this component
 * (<Tokens></Tokens>) as a child instead of carrying its own copy of this
 * block. htmlc's StyleCollector dedupes style contributions by exact
 * (scope, CSS text) match (see style.go's StyleCollector.Add) — global/
 * unscoped contributions use an empty scope, so this file's fixed <style>
 * text collapses into a single entry in the final page's stylesheet no
 * matter how many of the 30+ components in this package a given page
 * mounts, since each of them renders this exact same child component
 * rather than duplicating its CSS text inline. This is the mechanism, not
 * an assumption: confirmed by reading style.go's Add directly, and by
 * rendering a throwaway fixture, before adopting this pattern.
 *
 * This block is the single canonical source of truth (edit it here, in
 * Tokens.vue, only — see radix.go's own header comment, "Design tokens",
 * for the cross-reference). Space tokens are defined for completeness/
 * future use; not yet adopted for every padding/margin/gap declaration
 * across this package (a deliberate v1 scope cut).
 */
:root {
  /* Radix Colors, sand scale (warm neutral), light mode */
  --radix-sand-1: #fdfdfc;
  --radix-sand-2: #f9f9f8;
  --radix-sand-3: #f1f0ef;
  --radix-sand-4: #e9e8e6;
  --radix-sand-5: #e2e1de;
  --radix-sand-6: #dad9d6;
  --radix-sand-7: #cfceca;
  --radix-sand-8: #bcbbb5;
  --radix-sand-9: #8d8d86;
  --radix-sand-10: #82827c;
  --radix-sand-11: #63635e;
  --radix-sand-12: #21201c;

  /* Radix Colors, brown scale (accent), light mode */
  --radix-brown-1: #fefdfc;
  --radix-brown-2: #fcf9f6;
  --radix-brown-3: #f6eee7;
  --radix-brown-4: #f0e4d9;
  --radix-brown-5: #ebdaca;
  --radix-brown-6: #e4cdb7;
  --radix-brown-7: #dcbc9f;
  --radix-brown-8: #cea37e;
  --radix-brown-9: #ad7f58;
  --radix-brown-10: #a07553;
  --radix-brown-11: #815e46;
  --radix-brown-12: #3e332e;

  /* Radix Colors, ruby scale (danger/destructive), light mode */
  --radix-ruby-1: #fffcfd;
  --radix-ruby-2: #fff7f8;
  --radix-ruby-3: #feeaed;
  --radix-ruby-4: #ffdce1;
  --radix-ruby-5: #ffced6;
  --radix-ruby-6: #f8bfc8;
  --radix-ruby-7: #efacb8;
  --radix-ruby-8: #e592a3;
  --radix-ruby-9: #e54666;
  --radix-ruby-10: #dc3b5d;
  --radix-ruby-11: #ca244d;
  --radix-ruby-12: #64172b;

  /* Radix Themes radius scale, radius-factor=1 ("medium"), scaling=1 */
  --radix-radius-1: 3px;
  --radix-radius-2: 4px;
  --radix-radius-3: 6px;
  --radix-radius-4: 8px;
  --radix-radius-5: 12px;
  --radix-radius-6: 16px;
  --radix-radius-thumb: 9999px;

  /* Radix Themes spacing scale, scaling=1 */
  --radix-space-1: 4px;
  --radix-space-2: 8px;
  --radix-space-3: 12px;
  --radix-space-4: 16px;
  --radix-space-5: 24px;
  --radix-space-6: 32px;
  --radix-space-7: 40px;
  --radix-space-8: 48px;
  --radix-space-9: 64px;
}

/*
 * Shared visually-hidden-input technique (was previously repeated
 * verbatim, under a different class name, in Checkbox.vue/RadioGroup.vue/
 * Select.vue/Switch.vue/Tabs.vue — same dedup mechanism as above). A
 * component using this pairs it as a *second* class alongside its own
 * component-specific class (e.g. class="radix-checkbox-input
 * radix-visually-hidden-input"), so this class only ever contributes the
 * hiding declarations; every compound/sibling selector a component needs
 * (:checked + .foo, :focus-visible + .foo, etc.) keeps using its own
 * specific class, unaffected by this one.
 */
.radix-visually-hidden-input {
  position: absolute;
  opacity: 0;
  width: 1px;
  height: 1px;
  margin: -1px;
  padding: 0;
  overflow: hidden;
  clip-path: inset(50%);
  white-space: nowrap;
  border: 0;
}
</style>
