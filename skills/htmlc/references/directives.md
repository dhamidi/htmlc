# htmlc Directive Reference

## Table of Contents

1. [Text interpolation](#text-interpolation)
2. [v-if / v-else-if / v-else](#v-if--v-else-if--v-else)
3. [v-for](#v-for)
4. [v-show](#v-show)
5. [v-bind](#v-bind)
6. [v-text and v-html](#v-text-and-v-html)
7. [v-switch / v-case / v-default](#v-switch--v-case--v-default)
8. [v-pre](#v-pre)
9. [v-slot / #name](#v-slot--name)
10. [Stripped directives](#stripped-directives)

---

## Text interpolation

```html
<p>Hello, {{ name }}!</p>
<p>{{ a }} + {{ b }} = {{ a + b }}</p>
```

- HTML-escaped
- Multiple interpolations per text node are supported
- Evaluated against the current render scope

---

## v-if / v-else-if / v-else

```html
<p v-if="count > 0">{{ count }} items</p>
<p v-else-if="count === 0">Empty</p>
<p v-else>Unknown</p>
```

- `v-else-if` and `v-else` must immediately follow a sibling with `v-if` or `v-else-if`
- Can be used on `<template>` to group multiple elements without a wrapper

---

## v-for

```html
<!-- slice -->
<li v-for="item in items">{{ item.name }}</li>

<!-- with index -->
<li v-for="(item, index) in items">{{ index }}: {{ item.name }}</li>

<!-- map -->
<li v-for="(value, key) in obj">{{ key }}: {{ value }}</li>

<!-- integer range (1..n inclusive) -->
<li v-for="i in 5">{{ i }}</li>
```

- Iteration variable is scoped to the loop body
- Map iteration order is **not guaranteed** (Go `reflect.MapKeys()`)
- Integer range `v-for="i in 5"` produces values 1, 2, 3, 4, 5

---

## v-show

```html
<div v-show="isVisible">Content</div>
```

- Element is always rendered in the HTML
- Adds `style="display:none"` when the expression is falsy
- Unlike `v-if`, the element is never removed from the output

---

## v-bind

### Single attribute

```html
<a :href="url">Link</a>
<img :src="imageSrc" :alt="imageAlt" />
```

### Spread from map or struct

```html
<div v-bind="attrs">...</div>
```

`attrs` must be `map[string]any` or a Go struct (fields mapped via JSON tags).

### :class

Object syntax — keys are class names, values are booleans:
```html
<div :class="{ active: isActive, disabled: isDisabled }">...</div>
```

Array syntax — each element is a class name string:
```html
<div :class="[baseClass, extraClass]">...</div>
```

Can be combined with a static `class` attribute.

### :style

Object syntax with camelCase keys converted to kebab-case CSS properties:
```html
<div :style="{ backgroundColor: color, fontSize: size + 'px' }">...</div>
```

---

## v-text and v-html

```html
<!-- HTML-escaped; replaces all children -->
<span v-text="message"></span>

<!-- NOT escaped; use only with trusted content -->
<div v-html="rawHtml"></div>
```

`v-text` is equivalent to `{{ expr }}` but replaces the entire element's children.

---

## v-switch / v-case / v-default

Implements Vue RFC #482 (not yet stable in Vue.js). Only valid on `<template>` elements.

```html
<template v-switch="status">
  <div v-case="'active'">Active</div>
  <div v-case="'inactive'">Inactive</div>
  <div v-default>Unknown</div>
</template>
```

- `v-switch` evaluates the expression once
- Each `v-case` is compared with `===`
- `v-default` renders when no case matches
- Only the first matching case renders

---

## v-pre

```html
<div v-pre>
  {{ this will not be evaluated }}
  <span v-if="ignored">also ignored</span>
</div>
```

Skips all interpolation and directive processing for the element and its entire subtree. Useful for displaying raw template syntax as documentation.

---

## v-slot / #name

### Default slot

In the child component (`Card.vue`):
```html
<template>
  <div class="card">
    <slot />
  </div>
</template>
```

In the caller:
```html
<Card>
  <p>This goes into the default slot</p>
</Card>
```

### Named slots

In the child (`Layout.vue`):
```html
<template>
  <header><slot name="header" /></header>
  <main><slot /></main>
</template>
```

In the caller:
```html
<Layout>
  <template #header><h1>Title</h1></template>
  <p>Main content</p>
</Layout>
```

### Scoped slots

In the child (`List.vue`):
```html
<template>
  <ul>
    <li v-for="item in items">
      <slot :item="item" />
    </li>
  </ul>
</template>
```

In the caller:
```html
<List :items="items">
  <template #default="{ item }">
    <strong>{{ item.name }}</strong>
  </template>
</List>
```

Slot content is always evaluated in the **caller's** scope. Slot props are merged into scope only for that slot's content.

---

## Stripped directives

These directives are accepted by the parser but produce no output — they are client-side only:

| Directive | Vue.js purpose |
|---|---|
| `v-model` | Two-way data binding |
| `v-on` / `@event` | Event handlers |
| `v-cloak` | Hide un-compiled template |
| `v-memo` | Memoize subtree |
| `v-once` | Render once, skip future updates |

Passing these directives does not cause an error. They are silently removed from the rendered HTML.
