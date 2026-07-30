<!--
  HomePage — the radix-demo landing page.

  This page is the first real, end-to-end consumer of ui/radix mounted via
  Options.Mounts (RFC 014 §6 Examples 1-3). It deliberately mixes every
  documented reference form from RFC 014 §5's Syntax Summary table for a
  mounted component, on one page, so it doubles as living documentation:

    1. Plain unqualified name    <Accordion :items="faqItems" />
    2. PascalCase alias          <RadixTabs :items="tabItems" />
    3. kebab-case alias          <radix-dialog :open="false">...</radix-dialog>
    4. Explicit qualified form   <component is="radix/Accordion">...</component>

  components/ (this demo's ComponentDir) has no local Accordion.vue, Tabs.vue,
  or Dialog.vue of its own, so form 1 above resolves unambiguously to the
  vendored ui/radix component with no special syntax needed — exactly RFC 014
  §6 Example 2. The genuine cross-mount collision scenario (RFC 014 §6
  Example 3) is deliberately NOT reproduced here — it lives in its own
  isolated test (collision_test.go) so the live demo app itself stays
  collision-free.

  Every component reference below uses explicit open/close tags
  (<Accordion>...</Accordion>, not <Accordion ... />) rather than
  self-closing syntax — matching RFC 014 §5's own Syntax Summary table,
  which documents the open/close form. This is deliberate, not just a style
  preference: a self-closing custom-component tag is auto-corrected by the
  parser (commit b8de838 fixed this for the lowercase built-in <component>
  tag specifically) but every auto-correction, self-closing <component> or
  not, still records a Component.Warnings entry, which ValidateAll()
  surfaces as a ValidationError — and this demo's own tests assert
  ValidateAll() returns zero errors for the live app. The self-closing
  <component> bugfix itself is exercised and confirmed separately, in
  main_test.go, without tripping that assertion.
-->
<template>
  <!DOCTYPE html>
  <html lang="en">
    <head>
      <meta charset="UTF-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
      <title>{{ title }}</title>
    </head>
    <body>
      <h1>{{ title }}</h1>
      <p>
        A single page demonstrating every documented reference form for a
        component mounted via <code>Options.Mounts</code> (RFC 014).
      </p>

      <section>
        <h2>1. Accordion — plain unqualified name</h2>
        <p><code>&lt;Accordion :items="faqItems"&gt;&lt;/Accordion&gt;</code></p>
        <Accordion :items="faqItems"></Accordion>
      </section>

      <section>
        <h2>2. Tabs — PascalCase alias</h2>
        <p><code>&lt;RadixTabs :items="tabItems"&gt;&lt;/RadixTabs&gt;</code></p>
        <RadixTabs :items="tabItems"></RadixTabs>
      </section>

      <section>
        <h2>3. Dialog — kebab-case alias</h2>
        <p><code>&lt;radix-dialog :open="false"&gt;...&lt;/radix-dialog&gt;</code></p>
        <radix-dialog :open="false">
          <h3>Welcome</h3>
          <p>
            This dialog is referenced via its auto-registered kebab-case
            alias, <code>radix-dialog</code>. It renders closed
            (<code>:open="false"</code> omits the <code>open</code>
            attribute) until its <code>&lt;script customelement&gt;</code>
            enhancement or a caller-set prop opens it.
          </p>
        </radix-dialog>
      </section>

      <section>
        <h2>4. Accordion again — explicit qualified form</h2>
        <p>
          <code>&lt;component is="radix/Accordion"&gt;...&lt;/component&gt;</code>
        </p>
        <component is="radix/Accordion" :items="moreFaqItems"></component>
        <p class="after-component">
          This paragraph renders after the explicit
          <code>&lt;component&gt;</code> tag above.
        </p>
      </section>
    </body>
  </html>
</template>
