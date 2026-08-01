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
  to invent new usage patterns. Verified by an actual rendered screenshot,
  not just a successful render call: that screenshot caught a real bug this
  gallery's addition surfaced in four popover-backed components
  (ContextMenu/DropdownMenu/Menubar/Select), whose scoped
  `.radix-*-content { display: flex; ... }` rule unconditionally overrode
  the browser's native `[popover]:not(:popover-open) { display: none; }`
  default — an author-origin style always wins that cascade over a
  UA-origin one, regardless of selector specificity — making each of those
  four always render open. Fixed at the source (see each file's own
  `:popover-open`-gated rule), not papered over here.
-->
<template>
  <!DOCTYPE html>
  <html lang="en">
    <head>
      <meta charset="UTF-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
      <title>{{ title }}</title>
      <style>
        body {
          font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
            Helvetica, Arial, sans-serif;
          color: #1a1a1a;
          max-width: 720px;
          margin: 3rem auto;
          padding: 0 1.5rem;
          line-height: 1.6;
        }
        h1 {
          font-size: 1.75rem;
        }
        h2 {
          font-size: 1.15rem;
          margin-top: 0;
        }
        section {
          margin: 2.5rem 0;
        }
        p {
          color: #444;
        }
        code {
          background: #f2f2f2;
          border-radius: 3px;
          padding: 0.15em 0.4em;
          font-size: 0.9em;
        }
        h2.category {
          font-size: 1.5rem;
          margin: 3.5rem 0 0.5rem;
          padding-top: 2rem;
          border-top: 2px solid #1a1a1a;
        }
        h3 {
          font-size: 1rem;
          margin: 0 0 0.25rem;
        }
        .demo-row {
          display: flex;
          flex-wrap: wrap;
          align-items: center;
          gap: 1.5rem;
          margin: 0.75rem 0;
        }
        .demo-field {
          display: flex;
          flex-direction: column;
          gap: 0.35rem;
        }
        .card {
          padding: 1.5rem;
          border: 1px dashed #a3a3a3;
          border-radius: 6px;
          text-align: center;
          color: #666;
          user-select: none;
        }
        .radix-scroll-area-viewport {
          height: 120px;
        }
        /*
         * Not a ui/radix component — this package ships no standalone
         * Button primitive, matching real Radix Primitives, which also
         * leaves plain buttons to consumer styling rather than shipping
         * one. This page-level class instead reuses the exact visual
         * language every ui/radix component's own internal buttons already
         * share (compare .radix-dialog-close/.radix-alert-dialog-cancel in
         * Dialog.vue/AlertDialog.vue), so the plain <button> elements this
         * demo authors itself as slot content (the Form section's "Sign
         * up", AlertDialog's "Delete", VisuallyHidden's checkmark trigger)
         * read as part of the same design system instead of falling back
         * to the browser's raw default button chrome.
         */
        .radix-button {
          padding: 0.4rem 0.9rem;
          border: 1px solid #d9d9d9;
          border-radius: 6px;
          background-color: #fff;
          font: inherit;
          cursor: pointer;
        }
        .radix-button:hover {
          background-color: #f5f5f5;
        }
        .radix-button:focus-visible {
          outline: 2px solid #2563eb;
          outline-offset: 2px;
        }
        .radix-button-primary {
          border-color: #2563eb;
          background-color: #2563eb;
          color: #fff;
        }
        .radix-button-primary:hover {
          background-color: #1d4ed8;
        }
      </style>
    </head>
    <body>
      <h1>{{ title }}</h1>
      <p>
        A single page demonstrating every documented reference form for a
        component mounted via <code>Options.Mounts</code> (RFC 014), plus a
        full gallery below showcasing every one of the 31 components in
        <code>ui/radix</code>.
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

      <h2 class="category">Forms &amp; Inputs</h2>

      <section>
        <h3>5. Label</h3>
        <p><code>&lt;Label for="newsletter-email"&gt;Email&lt;/Label&gt;</code></p>
        <div class="demo-field">
          <Label for="newsletter-email">Email</Label>
          <input id="newsletter-email" type="email" placeholder="you@example.com" />
        </div>
      </section>

      <section>
        <h3>6. Checkbox</h3>
        <p><code>&lt;Checkbox :checked="true" :indeterminate="false" ... /&gt;</code></p>
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

      <section>
        <h3>7. RadioGroup</h3>
        <p><code>&lt;RadioGroup name="..." value="daily" :items="[...]" /&gt;</code></p>
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

      <section>
        <h3>8. Switch</h3>
        <p><code>&lt;Switch :checked="false" ... /&gt;</code></p>
        <div class="demo-field">
          <Label for="airplane-mode">Airplane mode</Label>
          <Switch id="airplane-mode" name="airplane-mode" value="on" :checked="false" :disabled="false" :required="false"></Switch>
        </div>
      </section>

      <section>
        <h3>9. Toggle</h3>
        <p><code>&lt;Toggle :pressed="true"&gt;&lt;strong&gt;B&lt;/strong&gt;&lt;/Toggle&gt;</code></p>
        <Toggle :pressed="true" :disabled="false"><strong>B</strong></Toggle>
      </section>

      <section>
        <h3>10. ToggleGroup</h3>
        <p><code>&lt;ToggleGroup type="single" :items="[...]" /&gt;</code></p>
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

      <section>
        <h3>11. Slider</h3>
        <p><code>&lt;Slider :values="[50]" :min="0" :max="100" :step="1" /&gt;</code></p>
        <div class="demo-field">
          <Slider :values="[50]" :min="0" :max="100" :step="1"></Slider>
          <Slider :values="[25, 75]" :min="0" :max="100" :step="1"></Slider>
        </div>
      </section>

      <section>
        <h3>12. PasswordToggleField</h3>
        <p><code>&lt;PasswordToggleField id="pw" name="password" ... /&gt;</code></p>
        <div class="demo-field">
          <Label for="pw">Password</Label>
          <PasswordToggleField id="pw" name="password" :required="true" :disabled="false"></PasswordToggleField>
        </div>
      </section>

      <section>
        <h3>13. OneTimePasswordField</h3>
        <p><code>&lt;OneTimePasswordField :length="[...]" name="otp" /&gt;</code></p>
        <OneTimePasswordField :length="[null, null, null, null, null, null]" name="otp"></OneTimePasswordField>
      </section>

      <section>
        <h3>14. Select</h3>
        <p><code>&lt;Select id="fruit" name="fruit" value="apple" :items="[...]" /&gt;</code></p>
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

      <section>
        <h3>15. Form</h3>
        <p><code>&lt;Form id="..." label="..." error="..."&gt;&lt;input .../&gt;&lt;/Form&gt;</code></p>
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
          <button type="submit" class="radix-button radix-button-primary">Sign up</button>
        </form>
      </section>

      <h2 class="category">Overlays &amp; Menus</h2>

      <section>
        <h3>16. AlertDialog</h3>
        <p><code>&lt;AlertDialog :open="false"&gt;...&lt;template #actions&gt;...&lt;/AlertDialog&gt;</code></p>
        <AlertDialog :open="false">
          <h3>Delete this file?</h3>
          <p>This action cannot be undone.</p>
          <template #actions>
            <button type="button" class="radix-button">Delete</button>
          </template>
        </AlertDialog>
      </section>

      <section>
        <h3>17. Popover</h3>
        <p><code>&lt;Popover id="..."&gt;...&lt;template #content&gt;...&lt;/Popover&gt;</code></p>
        <Popover id="notifications-popover">
          Notifications
          <template #content>
            <h3>Notifications</h3>
            <p>You're all caught up.</p>
          </template>
        </Popover>
      </section>

      <section>
        <h3>18. HoverCard</h3>
        <p><code>&lt;HoverCard :openDelayMs="700" :closeDelayMs="300"&gt;...&lt;/HoverCard&gt;</code></p>
        <HoverCard :openDelayMs="700" :closeDelayMs="300">
          <a href="/users/kentcdodds">@kentcdodds</a>
          <template #content>
            <strong>Kent C. Dodds</strong>
            <p>Teaching people development skills.</p>
          </template>
        </HoverCard>
      </section>

      <section>
        <h3>19. Tooltip</h3>
        <p><code>&lt;Tooltip content="Saved to your library"&gt;Save&lt;/Tooltip&gt;</code></p>
        <Tooltip content="Saved to your library" :delayMs="700">Save</Tooltip>
      </section>

      <section>
        <h3>20. DropdownMenu</h3>
        <p><code>&lt;DropdownMenu id="..." :items="[...]"&gt;File&lt;/DropdownMenu&gt;</code></p>
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

      <section>
        <h3>21. ContextMenu</h3>
        <p><code>&lt;ContextMenu :items="[...]"&gt;...&lt;/ContextMenu&gt;</code></p>
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

      <section>
        <h3>22. Menubar</h3>
        <p><code>&lt;Menubar :menus="[...]" /&gt;</code></p>
        <Menubar
          :menus="[
            {
              id: 'file-menu',
              label: 'File',
              items: [
                { type: 'item', id: 'new', label: 'New File' },
                { type: 'item', id: 'open', label: 'Open...' },
                { type: 'separator' },
                { type: 'item', id: 'close', label: 'Close', disabled: true },
              ],
            },
            {
              id: 'edit-menu',
              label: 'Edit',
              items: [
                { type: 'item', id: 'undo', label: 'Undo' },
                { type: 'item', id: 'redo', label: 'Redo' },
              ],
            },
          ]"
        ></Menubar>
      </section>

      <section>
        <h3>23. NavigationMenu</h3>
        <p><code>&lt;NavigationMenu :items="[...]" /&gt;</code></p>
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

      <section>
        <h3>24. Toast</h3>
        <p><code>&lt;Toast id="..." title="..." variant="default" /&gt;</code></p>
        <div class="demo-field">
          <Toast id="toast-saved" title="Saved" description="Your changes have been saved." :duration="5000" variant="default"></Toast>
          <Toast id="toast-error" title="Something went wrong" description="Could not save your changes." :duration="0" variant="destructive"></Toast>
        </div>
      </section>

      <h2 class="category">Layout &amp; Display</h2>

      <section>
        <h3>25. AspectRatio</h3>
        <p><code>&lt;AspectRatio :ratio="16/9"&gt;...&lt;/AspectRatio&gt;</code></p>
        <div style="max-width: 20rem">
          <AspectRatio :ratio="16/9">
            <div style="width: 100%; height: 100%; background: #e5e7eb; display: flex; align-items: center; justify-content: center; color: #666;">16:9</div>
          </AspectRatio>
        </div>
      </section>

      <section>
        <h3>26. Avatar</h3>
        <p><code>&lt;Avatar :src="..." :alt="..." :delayMs="0"&gt;&lt;template #fallback&gt;...&lt;/template&gt;&lt;/Avatar&gt;</code></p>
        <Avatar src="/no-such-avatar.jpg" alt="Kent C. Dodds" :delayMs="0">
          <template #fallback>KD</template>
        </Avatar>
      </section>

      <section>
        <h3>27. Progress</h3>
        <p><code>&lt;Progress :value="80" :max="100" :indeterminate="false" /&gt;</code></p>
        <div class="demo-field">
          <Progress :value="80" :max="100" :indeterminate="false"></Progress>
          <Progress :value="0" :max="100" :indeterminate="true"></Progress>
        </div>
      </section>

      <section>
        <h3>28. ScrollArea</h3>
        <p><code>&lt;ScrollArea&gt;...&lt;/ScrollArea&gt;</code></p>
        <div style="max-width: 20rem">
          <ScrollArea>
            <p>Radix-inspired, headless, zero-JS-baseline components for htmlc.</p>
            <p>Mount the package, then reference its components by any documented form.</p>
            <p>Every component ships a real, functional zero-JS baseline first.</p>
            <p>Progressive enhancement layers JS on top only where genuinely needed.</p>
          </ScrollArea>
        </div>
      </section>

      <section>
        <h3>29. Separator</h3>
        <p><code>&lt;Separator :orientation="'horizontal'" :decorative="false" /&gt;</code></p>
        <Separator :orientation="'horizontal'" :decorative="false"></Separator>
      </section>

      <section>
        <h3>30. Toolbar</h3>
        <p><code>&lt;Toolbar orientation="horizontal" :items="[...]" /&gt;</code></p>
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

      <section>
        <h3>31. VisuallyHidden</h3>
        <p><code>&lt;VisuallyHidden&gt;...&lt;/VisuallyHidden&gt;</code></p>
        <button type="button" class="radix-button">
          <span aria-hidden="true">&#10003;</span>
          <VisuallyHidden>Mark as complete</VisuallyHidden>
        </button>
      </section>

      <section>
        <h3>32. Collapsible</h3>
        <p><code>&lt;Collapsible :open="false"&gt;&lt;template #trigger&gt;...&lt;/template&gt;...&lt;/Collapsible&gt;</code></p>
        <Collapsible :open="false">
          <template #trigger>Show details</template>
          <p>Panel body content, composed freely by the caller.</p>
        </Collapsible>
      </section>
    </body>
  </html>
</template>
