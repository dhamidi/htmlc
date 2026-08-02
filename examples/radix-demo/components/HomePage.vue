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

  Below the RFC 014 reference-forms section is a "Component Gallery": one
  instance of every remaining component in ui/radix (28 of the package's 31
  components — Accordion/Tabs/Dialog are already covered above), grouped
  into "Forms & Inputs", "Overlays & Menus", and "Layout & Display",
  matching the categories on Radix Primitives' own site. Every prop value
  is an inline literal at the call site (no Go-side data plumbing), mirroring
  the "Usage:" example already documented in each component's own header
  comment — this page exists to prove those examples actually render, not
  to invent new usage patterns.

  ## Page chrome (this file's own <style>, not ui/radix's)

  The page-level styling below deliberately reuses the same --radix-gray-*/
  --radix-blue-*/--radix-red-* custom properties every ui/radix component
  already renders with (defined once, by Tokens.vue, and available here for
  free: this page mounts many ui/radix components, each of which pulls in
  <Tokens></Tokens> as its own first template child, and CSS custom
  properties resolve from wherever in the document a :root rule defining
  them happens to live — order relative to this file's own <style> block
  does not matter, confirmed by checking the rendered output after writing
  this). Dark-mode values for the same three scales (fetched from
  @radix-ui/colors' own gray-dark.css/blue-dark.css/red-dark.css, not
  guessed) are layered in via `@media (prefers-color-scheme: dark)`,
  re-declaring the identical custom-property names so every component's own
  `var(--radix-blue-9)`-style reference picks up the dark value automatically
  with no JS theme toggle and no per-component change.

  That override is written `:root:root { ... }`, not plain `:root`, and
  that doubling is load-bearing, not decorative. This file's own <style>
  block is rendered before the aggregated ui/radix component stylesheet in
  the final page, and Tokens.vue's own light-mode values are declared as a
  plain, unconditional `:root { ... }` (no media query) later in that
  aggregated block. A plain `:root` here would have equal specificity to
  Tokens.vue's rule, and per the cascade's normal tie-break (last matching
  declaration in source order wins) Tokens.vue's later, always-applicable
  light values would win even when this media query matches, silently
  defeating dark mode. Confirmed empirically: with a plain `:root`,
  `matchMedia('(prefers-color-scheme: dark)').matches` correctly reported
  `true` under CDP emulation, yet
  `getComputedStyle(document.body).backgroundColor` still came back as the
  light value. Doubling the selector to `:root:root` raises this rule's
  specificity above Tokens.vue's plain `:root`, so it wins whenever the
  media query matches, regardless of source order.

  The left-hand <nav class="sidebar"> is a plain anchor-link list (href="#c5"
  etc., matching the `id` this file gives each <section>/category <h2>) —
  native in-page navigation, zero JS, consistent with every other zero-JS-
  baseline-first choice in this package. It is hidden below a 960px
  viewport width (see the media query) rather than made collapsible,
  because a hamburger-triggered off-canvas nav needs JS to open/close and
  this page has none; the single-column layout below that width was already
  verified to read fine without a nav aid earlier in this project's own
  mobile-viewport review.
-->
<template>
  <!DOCTYPE html>
  <html lang="en">
    <head>
      <meta charset="UTF-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
      <title>{{ title }}</title>
      <style>
        @media (prefers-color-scheme: dark) {
          :root:root {
            --radix-gray-1: #111111;
            --radix-gray-2: #191919;
            --radix-gray-3: #222222;
            --radix-gray-4: #2a2a2a;
            --radix-gray-5: #313131;
            --radix-gray-6: #3a3a3a;
            --radix-gray-7: #484848;
            --radix-gray-8: #606060;
            --radix-gray-9: #6e6e6e;
            --radix-gray-10: #7b7b7b;
            --radix-gray-11: #b4b4b4;
            --radix-gray-12: #eeeeee;
            --radix-blue-1: #0d1520;
            --radix-blue-2: #111927;
            --radix-blue-3: #0d2847;
            --radix-blue-4: #003362;
            --radix-blue-5: #004074;
            --radix-blue-6: #104d87;
            --radix-blue-7: #205d9e;
            --radix-blue-8: #2870bd;
            --radix-blue-9: #0090ff;
            --radix-blue-10: #3b9eff;
            --radix-blue-11: #70b8ff;
            --radix-blue-12: #c2e6ff;
            --radix-red-1: #191111;
            --radix-red-2: #201314;
            --radix-red-3: #3b1219;
            --radix-red-4: #500f1c;
            --radix-red-5: #611623;
            --radix-red-6: #72232d;
            --radix-red-7: #8c333a;
            --radix-red-8: #b54548;
            --radix-red-9: #e5484d;
            --radix-red-10: #ec5d5e;
            --radix-red-11: #ff9592;
            --radix-red-12: #ffd1d9;
            --page-surface: var(--radix-gray-2);
            --page-shadow: 0 1px 2px rgba(0, 0, 0, 0.4), 0 8px 24px rgba(0, 0, 0, 0.36);
          }
        }
        :root {
          --page-surface: #fff;
          --page-shadow: 0 1px 2px rgba(0, 0, 0, 0.04), 0 8px 24px rgba(0, 0, 0, 0.06);
        }
        * {
          box-sizing: border-box;
        }
        body {
          font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
            Helvetica, Arial, sans-serif;
          background: var(--radix-gray-1);
          color: var(--radix-gray-12);
          margin: 0;
          line-height: 1.6;
          display: grid;
          grid-template-columns: 240px minmax(0, 1fr);
          gap: 3rem;
          max-width: 1200px;
          padding: 3rem 1.5rem;
          margin: 0 auto;
          align-items: start;
        }
        .sidebar {
          position: sticky;
          top: 2rem;
          display: flex;
          flex-direction: column;
          gap: 0.35rem;
          max-height: calc(100vh - 4rem);
          overflow-y: auto;
          font-size: 0.85rem;
          padding-right: 0.5rem;
        }
        .sidebar-title {
          font-weight: 700;
          text-transform: uppercase;
          letter-spacing: 0.04em;
          font-size: 0.7rem;
          color: var(--radix-gray-9);
          margin: 1.25rem 0 0.35rem;
        }
        .sidebar-title:first-child {
          margin-top: 0;
        }
        .sidebar a {
          color: var(--radix-gray-11);
          text-decoration: none;
          padding: 0.2rem 0;
          border-radius: var(--radix-radius-2);
        }
        .sidebar a:hover {
          color: var(--radix-blue-9);
        }
        .sidebar a:focus-visible {
          outline: 2px solid var(--radix-blue-9);
          outline-offset: 2px;
        }
        main {
          min-width: 0;
        }
        h1 {
          font-size: 2rem;
          margin: 0 0 0.75rem;
          letter-spacing: -0.02em;
        }
        h2 {
          font-size: 1.05rem;
          margin: 0 0 0.75rem;
        }
        p {
          color: var(--radix-gray-11);
        }
        a {
          color: var(--radix-blue-9);
        }
        code {
          background: var(--radix-gray-3);
          color: var(--radix-gray-12);
          border-radius: var(--radix-radius-2);
          padding: 0.15em 0.4em;
          font-size: 0.85em;
          font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
        }
        .snippet {
          margin: 0.75rem 0;
        }
        .snippet code {
          display: block;
          padding: 0.6rem 0.8rem;
          overflow-x: auto;
          white-space: pre-wrap;
          word-break: break-word;
        }
        h2.category {
          font-size: 1.6rem;
          letter-spacing: -0.01em;
          margin: 4rem 0 1.5rem;
          padding-top: 2.5rem;
          border-top: 1px solid var(--radix-gray-6);
        }
        h2.category:first-of-type {
          margin-top: 3rem;
        }
        h3 {
          font-size: 0.95rem;
          margin: 0 0 0.6rem;
          color: var(--radix-gray-12);
        }
        section {
          margin: 1.25rem 0;
          padding: 1.25rem 1.5rem 1.5rem;
          background: var(--page-surface);
          border: 1px solid var(--radix-gray-6);
          border-radius: var(--radix-radius-4);
          box-shadow: var(--page-shadow);
          scroll-margin-top: 1.5rem;
        }
        .intro {
          margin: 0 0 2.5rem;
          padding: 0;
          border: 0;
          box-shadow: none;
          background: transparent;
        }
        .demo-row {
          display: flex;
          flex-wrap: wrap;
          align-items: center;
          gap: 1.5rem;
          margin: 0.75rem 0 0;
        }
        .demo-field {
          display: flex;
          flex-direction: column;
          gap: 0.35rem;
        }
        .card {
          padding: 1.5rem;
          border: 1px dashed var(--radix-gray-8);
          border-radius: var(--radix-radius-3);
          text-align: center;
          color: var(--radix-gray-11);
          user-select: none;
        }
        .radix-scroll-area-viewport {
          height: 120px;
        }
        @media (max-width: 960px) {
          body {
            grid-template-columns: minmax(0, 1fr);
            padding: 1.5rem 1rem;
          }
          .sidebar {
            display: none;
          }
        }
      </style>
      <script type="importmap">{{ importMap("/scripts/") }}</script>
      <script type="module" src="/scripts/index.js"></script>
    </head>
    <body>
      <nav class="sidebar" aria-label="Component index">
        <div class="sidebar-title">RFC 014 Forms</div>
        <a href="#c1">1. Accordion</a>
        <a href="#c2">2. Tabs</a>
        <a href="#c3">3. Dialog</a>
        <a href="#c4">4. Accordion (explicit)</a>
        <div class="sidebar-title">Forms &amp; Inputs</div>
        <a href="#c5">5. Label</a>
        <a href="#c6">6. Checkbox</a>
        <a href="#c7">7. RadioGroup</a>
        <a href="#c8">8. Switch</a>
        <a href="#c9">9. Toggle</a>
        <a href="#c10">10. ToggleGroup</a>
        <a href="#c11">11. Slider</a>
        <a href="#c12">12. PasswordToggleField</a>
        <a href="#c13">13. OneTimePasswordField</a>
        <a href="#c14">14. Select</a>
        <a href="#c15">15. Form</a>
        <div class="sidebar-title">Overlays &amp; Menus</div>
        <a href="#c16">16. AlertDialog</a>
        <a href="#c17">17. Popover</a>
        <a href="#c18">18. HoverCard</a>
        <a href="#c19">19. Tooltip</a>
        <a href="#c20">20. DropdownMenu</a>
        <a href="#c21">21. ContextMenu</a>
        <a href="#c22">22. Menubar</a>
        <a href="#c23">23. NavigationMenu</a>
        <a href="#c24">24. Toast</a>
        <div class="sidebar-title">Layout &amp; Display</div>
        <a href="#c25">25. AspectRatio</a>
        <a href="#c26">26. Avatar</a>
        <a href="#c27">27. Progress</a>
        <a href="#c28">28. ScrollArea</a>
        <a href="#c29">29. Separator</a>
        <a href="#c30">30. Toolbar</a>
        <a href="#c31">31. VisuallyHidden</a>
        <a href="#c32">32. Collapsible</a>
        <a href="#c33">33. Button</a>
      </nav>
      <main>
        <h1>{{ title }}</h1>
        <p class="intro">
          A single page demonstrating every documented reference form for a
          component mounted via <code>Options.Mounts</code> (RFC 014), plus a
          full gallery below showcasing every one of the 31 components in
          <code>ui/radix</code>.
        </p>

        <section id="c1">
          <h2>1. Accordion — plain unqualified name</h2>
          <p class="snippet"><code>&lt;Accordion :items="faqItems"&gt;&lt;/Accordion&gt;</code></p>
          <Accordion :items="faqItems"></Accordion>
        </section>

        <section id="c2">
          <h2>2. Tabs — PascalCase alias</h2>
          <p class="snippet"><code>&lt;RadixTabs :items="tabItems"&gt;&lt;/RadixTabs&gt;</code></p>
          <RadixTabs :items="tabItems"></RadixTabs>
        </section>

        <section id="c3">
          <h2>3. Dialog — kebab-case alias</h2>
          <p class="snippet"><code>&lt;radix-dialog :open="false"&gt;...&lt;/radix-dialog&gt;</code></p>
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

        <section id="c4">
          <h2>4. Accordion again — explicit qualified form</h2>
          <p class="snippet">
            <code>&lt;component is="radix/Accordion"&gt;...&lt;/component&gt;</code>
          </p>
          <component is="radix/Accordion" :items="moreFaqItems"></component>
          <p class="after-component">
            This paragraph renders after the explicit
            <code>&lt;component&gt;</code> tag above.
          </p>
        </section>

        <h2 class="category">Forms &amp; Inputs</h2>

        <section id="c5">
          <h3>5. Label</h3>
          <p class="snippet"><code>&lt;Label for="newsletter-email"&gt;Email&lt;/Label&gt;</code></p>
          <div class="demo-field">
            <Label for="newsletter-email">Email</Label>
            <input id="newsletter-email" type="email" placeholder="you@example.com" />
          </div>
        </section>

        <section id="c6">
          <h3>6. Checkbox</h3>
          <p class="snippet"><code>&lt;Checkbox :checked="true" :indeterminate="false" ... /&gt;</code></p>
          <div class="demo-row">
            <div class="demo-field">
              <Label for="terms">Accept the terms</Label>
              <Checkbox id="terms" name="terms" value="on" :checked="true" :indeterminate="false" :disabled="false" :required="true"></Checkbox>
            </div>
            <div class="demo-field">
              <Label for="select-all">Select all (mixed)</Label>
              <Checkbox id="select-all" name="select-all" value="on" :checked="false" :indeterminate="true" :disabled="false" :required="false"></Checkbox>
            </div>
          </div>
        </section>

        <section id="c7">
          <h3>7. RadioGroup</h3>
          <p class="snippet"><code>&lt;RadioGroup name="..." value="daily" :items="[...]" /&gt;</code></p>
          <RadioGroup
            name="notification-frequency"
            value="daily"
            :disabled="false"
            :items="[
              { id: 'daily', value: 'daily', label: 'Daily' },
              { id: 'weekly', value: 'weekly', label: 'Weekly' },
              { id: 'never', value: 'never', label: 'Never' },
            ]"
          ></RadioGroup>
        </section>

        <section id="c8">
          <h3>8. Switch</h3>
          <p class="snippet"><code>&lt;Switch :checked="false" ... /&gt;</code></p>
          <div class="demo-field">
            <Label for="airplane-mode">Airplane mode</Label>
            <Switch id="airplane-mode" name="airplane-mode" value="on" :checked="false" :disabled="false" :required="false"></Switch>
          </div>
        </section>

        <section id="c9">
          <h3>9. Toggle</h3>
          <p class="snippet"><code>&lt;Toggle :pressed="true"&gt;&lt;strong&gt;B&lt;/strong&gt;&lt;/Toggle&gt;</code></p>
          <Toggle :pressed="true" :disabled="false"><strong>B</strong></Toggle>
        </section>

        <section id="c10">
          <h3>10. ToggleGroup</h3>
          <p class="snippet"><code>&lt;ToggleGroup type="single" :items="[...]" /&gt;</code></p>
          <div class="demo-row">
            <ToggleGroup
              type="single"
              value="bold"
              :disabled="false"
              :items="[
                { id: 'bold', label: 'B', pressed: false },
                { id: 'italic', label: 'I', pressed: false },
                { id: 'underline', label: 'U', pressed: false },
              ]"
            ></ToggleGroup>
            <ToggleGroup
              type="multiple"
              value=""
              :disabled="false"
              :items="[
                { id: 'left', label: 'L', pressed: true },
                { id: 'center', label: 'C', pressed: false },
                { id: 'right', label: 'R', pressed: false },
              ]"
            ></ToggleGroup>
          </div>
        </section>

        <section id="c11">
          <h3>11. Slider</h3>
          <p class="snippet"><code>&lt;Slider :values="[50]" :min="0" :max="100" :step="1" /&gt;</code></p>
          <div class="demo-field">
            <Slider :values="[50]" :min="0" :max="100" :step="1"></Slider>
            <Slider :values="[25, 75]" :min="0" :max="100" :step="1"></Slider>
          </div>
        </section>

        <section id="c12">
          <h3>12. PasswordToggleField</h3>
          <p class="snippet"><code>&lt;PasswordToggleField id="pw" name="password" ... /&gt;</code></p>
          <div class="demo-field">
            <Label for="pw">Password</Label>
            <PasswordToggleField id="pw" name="password" :required="true" :disabled="false"></PasswordToggleField>
          </div>
        </section>

        <section id="c13">
          <h3>13. OneTimePasswordField</h3>
          <p class="snippet"><code>&lt;OneTimePasswordField :length="[...]" name="otp" /&gt;</code></p>
          <OneTimePasswordField :length="[null, null, null, null, null, null]" name="otp"></OneTimePasswordField>
        </section>

        <section id="c14">
          <h3>14. Select</h3>
          <p class="snippet"><code>&lt;Select id="fruit" name="fruit" value="apple" :items="[...]" /&gt;</code></p>
          <Select
            id="fruit"
            name="fruit"
            value="apple"
            :items="[
              { value: 'apple', label: 'Apple' },
              { value: 'banana', label: 'Banana' },
              { value: 'cherry', label: 'Cherry', disabled: true },
            ]"
          ></Select>
        </section>

        <section id="c15">
          <h3>15. Form</h3>
          <p class="snippet"><code>&lt;Form id="..." label="..." error="..."&gt;&lt;input .../&gt;&lt;/Form&gt;</code></p>
          <!--
            Deliberately <radix-form>, not <Form>: this page's own literal
            <form v-native> below is real, caller-side HTML, parsed by the
            same html.Parse call as everything else in this <template> — and
            <Form> lowercases to the literal tag name "form" during HTML
            tokenization, same as the wrapping element. golang.org/x/net/html
            implements the WHATWG spec's own nested-<form> suppression rule
            (a <form> start tag is silently dropped whenever the parser's
            form-element pointer is already set — the same rule real browsers
            apply, verified via a standalone html.Parse reproduction before
            writing this comment): the inner <Form>...</Form> tag pair
            vanishes entirely before this engine's own component resolution
            ever sees it, silently promoting the slotted <input> to a direct
            child of the outer form and dropping the Form component's own
            label/message markup with no error or warning. <radix-form> (this
            package's auto-registered kebab-case alias for the same
            component) does not collide with the literal "form" tag name, so
            it survives standard HTML tree construction intact.
          -->
          <form v-native method="post" action="#">
            <radix-form id="signup-email" label="Email" error="">
              <input id="signup-email" type="email" name="email" />
            </radix-form>
            <Button variant="primary" type="submit" :disabled="false">Sign up</Button>
          </form>
        </section>

        <h2 class="category">Overlays &amp; Menus</h2>

        <section id="c16">
          <h3>16. AlertDialog</h3>
          <p class="snippet"><code>&lt;AlertDialog :open="false"&gt;...&lt;template #actions&gt;...&lt;/AlertDialog&gt;</code></p>
          <AlertDialog :open="false">
            <h3>Delete this file?</h3>
            <p>This action cannot be undone.</p>
            <template #actions>
              <Button variant="destructive" type="button" :disabled="false">Delete</Button>
            </template>
          </AlertDialog>
        </section>

        <section id="c17">
          <h3>17. Popover</h3>
          <p class="snippet"><code>&lt;Popover id="..."&gt;...&lt;template #content&gt;...&lt;/Popover&gt;</code></p>
          <Popover id="notifications-popover">
            Notifications
            <template #content>
              <h3>Notifications</h3>
              <p>You're all caught up.</p>
            </template>
          </Popover>
        </section>

        <section id="c18">
          <h3>18. HoverCard</h3>
          <p class="snippet"><code>&lt;HoverCard :openDelayMs="700" :closeDelayMs="300"&gt;...&lt;/HoverCard&gt;</code></p>
          <HoverCard :openDelayMs="700" :closeDelayMs="300">
            <a href="/users/kentcdodds">@kentcdodds</a>
            <template #content>
              <strong>Kent C. Dodds</strong>
              <p>Teaching people development skills.</p>
            </template>
          </HoverCard>
        </section>

        <section id="c19">
          <h3>19. Tooltip</h3>
          <p class="snippet"><code>&lt;Tooltip content="Saved to your library"&gt;Save&lt;/Tooltip&gt;</code></p>
          <Tooltip content="Saved to your library" :delayMs="700">Save</Tooltip>
        </section>

        <section id="c20">
          <h3>20. DropdownMenu</h3>
          <p class="snippet"><code>&lt;DropdownMenu id="..." :items="[...]"&gt;File&lt;/DropdownMenu&gt;</code></p>
          <DropdownMenu
            id="file-menu"
            :items="[
              { type: 'item', id: 'new', label: 'New File' },
              { type: 'item', id: 'open', label: 'Open...' },
              { type: 'separator' },
              { type: 'item', id: 'close', label: 'Close', disabled: true },
            ]"
          >
            File
          </DropdownMenu>
        </section>

        <section id="c21">
          <h3>21. ContextMenu</h3>
          <p class="snippet"><code>&lt;ContextMenu :items="[...]"&gt;...&lt;/ContextMenu&gt;</code></p>
          <ContextMenu
            :items="[
              { type: 'item', id: 'copy', label: 'Copy' },
              { type: 'item', id: 'paste', label: 'Paste' },
              { type: 'separator' },
              { type: 'item', id: 'delete', label: 'Delete', disabled: true },
            ]"
          >
            <div class="card">Right-click me</div>
          </ContextMenu>
        </section>

        <section id="c22">
          <h3>22. Menubar</h3>
          <p class="snippet"><code>&lt;Menubar :menus="[...]" /&gt;</code></p>
          <Menubar
            :menus="[
              {
                id: 'menubar-file-menu',
                label: 'File',
                items: [
                  { type: 'item', id: 'new', label: 'New File' },
                  { type: 'item', id: 'open', label: 'Open...' },
                  { type: 'separator' },
                  { type: 'item', id: 'close', label: 'Close', disabled: true },
                ],
              },
              {
                id: 'menubar-edit-menu',
                label: 'Edit',
                items: [
                  { type: 'item', id: 'undo', label: 'Undo' },
                  { type: 'item', id: 'redo', label: 'Redo' },
                ],
              },
            ]"
          ></Menubar>
        </section>

        <section id="c23">
          <h3>23. NavigationMenu</h3>
          <p class="snippet"><code>&lt;NavigationMenu :items="[...]" /&gt;</code></p>
          <NavigationMenu
            :items="[
              { id: 'home', label: 'Home', href: '/' },
              {
                id: 'products',
                label: 'Products',
                content: '<a href=/widgets>Widgets</a><a href=/gadgets>Gadgets</a>',
              },
              { id: 'about', label: 'About', href: '/about' },
            ]"
          ></NavigationMenu>
        </section>

        <section id="c24">
          <h3>24. Toast</h3>
          <p class="snippet"><code>&lt;Toast id="..." title="..." variant="default" /&gt;</code></p>
          <div class="demo-field">
            <Toast id="toast-saved" title="Saved" description="Your changes have been saved." :duration="5000" variant="default"></Toast>
            <Toast id="toast-error" title="Something went wrong" description="Could not save your changes." :duration="0" variant="destructive"></Toast>
          </div>
        </section>

        <h2 class="category">Layout &amp; Display</h2>

        <section id="c25">
          <h3>25. AspectRatio</h3>
          <p class="snippet"><code>&lt;AspectRatio :ratio="16/9"&gt;...&lt;/AspectRatio&gt;</code></p>
          <div style="max-width: 20rem">
            <AspectRatio :ratio="16/9">
              <div style="width: 100%; height: 100%; background: var(--radix-gray-4); display: flex; align-items: center; justify-content: center; color: var(--radix-gray-11);">16:9</div>
            </AspectRatio>
          </div>
        </section>

        <section id="c26">
          <h3>26. Avatar</h3>
          <p class="snippet"><code>&lt;Avatar :src="..." :alt="..." :delayMs="0"&gt;&lt;template #fallback&gt;...&lt;/template&gt;&lt;/Avatar&gt;</code></p>
          <Avatar src="/no-such-avatar.jpg" alt="Kent C. Dodds" :delayMs="0">
            <template #fallback>KD</template>
          </Avatar>
        </section>

        <section id="c27">
          <h3>27. Progress</h3>
          <p class="snippet"><code>&lt;Progress :value="80" :max="100" :indeterminate="false" /&gt;</code></p>
          <div class="demo-field">
            <Progress :value="80" :max="100" :indeterminate="false"></Progress>
            <Progress :value="0" :max="100" :indeterminate="true"></Progress>
          </div>
        </section>

        <section id="c28">
          <h3>28. ScrollArea</h3>
          <p class="snippet"><code>&lt;ScrollArea&gt;...&lt;/ScrollArea&gt;</code></p>
          <div style="max-width: 20rem">
            <ScrollArea>
              <p>Radix-inspired, headless, zero-JS-baseline components for htmlc.</p>
              <p>Mount the package, then reference its components by any documented form.</p>
              <p>Every component ships a real, functional zero-JS baseline first.</p>
              <p>Progressive enhancement layers JS on top only where genuinely needed.</p>
            </ScrollArea>
          </div>
        </section>

        <section id="c29">
          <h3>29. Separator</h3>
          <p class="snippet"><code>&lt;Separator :orientation="'horizontal'" :decorative="false" /&gt;</code></p>
          <Separator :orientation="'horizontal'" :decorative="false"></Separator>
        </section>

        <section id="c30">
          <h3>30. Toolbar</h3>
          <p class="snippet"><code>&lt;Toolbar orientation="horizontal" :items="[...]" /&gt;</code></p>
          <Toolbar
            orientation="horizontal"
            :items="[
              { type: 'button', id: 'bold', label: 'B' },
              { type: 'button', id: 'italic', label: 'I' },
              { type: 'separator' },
              { type: 'button', id: 'link', label: 'Link', disabled: true },
              { type: 'button', id: 'more', label: 'More' },
            ]"
          ></Toolbar>
        </section>

        <section id="c31">
          <h3>31. VisuallyHidden</h3>
          <p class="snippet"><code>&lt;VisuallyHidden&gt;...&lt;/VisuallyHidden&gt;</code></p>
          <Button variant="default" type="button" :disabled="false">
            <span aria-hidden="true">&#10003;</span>
            <VisuallyHidden>Mark as complete</VisuallyHidden>
          </Button>
        </section>

        <section id="c32">
          <h3>32. Collapsible</h3>
          <p class="snippet"><code>&lt;Collapsible :open="false"&gt;&lt;template #trigger&gt;...&lt;/template&gt;...&lt;/Collapsible&gt;</code></p>
          <Collapsible :open="false">
            <template #trigger>Show details</template>
            <p>Panel body content, composed freely by the caller.</p>
          </Collapsible>
        </section>

        <section id="c33">
          <h3>33. Button</h3>
          <p class="snippet"><code>&lt;Button variant="primary" type="button" :disabled="false"&gt;...&lt;/Button&gt;</code></p>
          <div class="demo-row">
            <Button variant="default" type="button" :disabled="false">Default</Button>
            <Button variant="primary" type="button" :disabled="false">Primary</Button>
            <Button variant="destructive" type="button" :disabled="false">Destructive</Button>
            <Button variant="default" type="button" :disabled="true">Disabled</Button>
          </div>
        </section>
      </main>
    </body>
  </html>
</template>
