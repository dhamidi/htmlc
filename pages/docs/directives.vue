<template>
  <DocsPage
    pageTitle="Directives — htmlc.sh"
    description="Full reference for all htmlc template directives: v-if, v-for, v-bind, v-show, v-html, v-text, v-switch, v-slot."
    :siteTitle="siteTitle"
    :navItems="[
      {label: 'Conditionals'},
      {href: '#v-if', label: 'v-if / v-else-if / v-else'},
      {href: '#v-show', label: 'v-show'},
      {href: '#v-switch', label: 'v-switch / v-case'},
      {label: 'Lists'},
      {href: '#v-for', label: 'v-for'},
      {label: 'Binding'},
      {href: '#v-bind', label: 'v-bind / :attr'},
      {href: '#v-html', label: 'v-html'},
      {href: '#v-text', label: 'v-text'},
      {href: '#v-pre', label: 'v-pre'},
      {label: 'Components'},
      {href: '#v-slot', label: 'v-slot / #slot'},
      {href: '#dynamic-component', label: 'component :is'},
      {label: 'Not supported'},
      {href: '#not-supported', label: 'Stripped directives'}
    ]"
  >
    <h1>Directives</h1>
    <p class="lead">Full reference for all template directives supported by htmlc.</p>

    <h2 id="v-if">v-if / v-else-if / v-else</h2>
    <p>Renders the element only when the expression is truthy. Whitespace-only text nodes between branches are ignored.</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;p v-if="role === 'admin'"&gt;Admin panel&lt;/p&gt;
&lt;p v-else-if="role === 'editor'"&gt;Editor view&lt;/p&gt;
&lt;p v-else&gt;Read-only view&lt;/p&gt;</code></pre>

    <p>Works on <code>&lt;template&gt;</code> elements too (renders children only, no wrapper element):</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;template v-if="items.length &gt; 0"&gt;
  &lt;ul&gt;
    &lt;li v-for="item in items"&gt;{{ item }}&lt;/li&gt;
  &lt;/ul&gt;
&lt;/template&gt;
&lt;template v-else&gt;
  &lt;p&gt;No items.&lt;/p&gt;
&lt;/template&gt;</code></pre>

    <h2 id="v-show">v-show</h2>
    <p>Adds <code>style="display:none"</code> when the expression is falsy. The element is always rendered (unlike <code>v-if</code>). Merges with any existing <code>style</code> attribute.</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;div v-show="isVisible"&gt;Visible when isVisible is truthy&lt;/div&gt;</code></pre>

    <h2 id="v-switch">v-switch / v-case / v-default</h2>
    <p>Switch/case conditional (implements Vue RFC #482). Must be on a <code>&lt;template&gt;</code> element. Renders the first matching <code>v-case</code> branch.</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;template v-switch="status"&gt;
  &lt;div v-case="'active'"&gt;Active&lt;/div&gt;
  &lt;div v-case="'pending'"&gt;Pending approval&lt;/div&gt;
  &lt;div v-default&gt;Unknown status&lt;/div&gt;
&lt;/template&gt;</code></pre>

    <h2 id="v-for">v-for</h2>
    <p>Repeats the element for each item in the iterable. Supports arrays, maps, and objects.</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- Array --&gt;
&lt;li v-for="item in items"&gt;{{ item }}&lt;/li&gt;

&lt;!-- With index --&gt;
&lt;li v-for="(item, index) in items"&gt;{{ index }}: {{ item }}&lt;/li&gt;

&lt;!-- Object/map --&gt;
&lt;li v-for="(value, key) in obj"&gt;{{ key }}: {{ value }}&lt;/li&gt;

&lt;!-- Range (integer) --&gt;
&lt;li v-for="i in 5"&gt;{{ i }}&lt;/li&gt;</code></pre>

    <Callout>
      <p><strong>Note:</strong> Map iteration order follows Go's <code>reflect.MapKeys()</code> — not insertion order. Sort your maps before passing them if order matters.</p>
    </Callout>

    <h2 id="v-bind">v-bind / :attr</h2>
    <p>Dynamically binds an HTML attribute to an expression. The shorthand is <code>:</code>.</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- Long form --&gt;
&lt;a v-bind:href="url"&gt;Link&lt;/a&gt;

&lt;!-- Shorthand --&gt;
&lt;a :href="url"&gt;Link&lt;/a&gt;
&lt;img :src="imageUrl" :alt="imageAlt" /&gt;

&lt;!-- Boolean attributes: rendered only when truthy --&gt;
&lt;button :disabled="isLoading"&gt;Submit&lt;/button&gt;

&lt;!-- Class binding --&gt;
&lt;div :class="isActive ? 'active' : ''"&gt;...&lt;/div&gt;</code></pre>

    <p>When passing props to a component, <code>:propName</code> evaluates the expression:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;Card :title="post.title" :author="post.author" /&gt;</code></pre>

    <h3>:class — object and array syntax</h3>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- Object: keys with truthy values are included --&gt;
&lt;div :class="{ active: isActive, disabled: !isEnabled }"&gt;...&lt;/div&gt;

&lt;!-- Array: non-empty string elements are included --&gt;
&lt;div :class="['btn', isPrimary ? 'primary' : '']"&gt;...&lt;/div&gt;

&lt;!-- Static class and :class are merged --&gt;
&lt;div class="card" :class="{ featured: post.featured }"&gt;...&lt;/div&gt;</code></pre>

    <h3>:style — object syntax</h3>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- camelCase keys are converted to kebab-case in output --&gt;
&lt;p :style="{ fontSize: '14px', backgroundColor: theme.bg }"&gt;...&lt;/p&gt;</code></pre>

    <h3>Boolean attributes</h3>
    <p>
      When a bound attribute name is a recognised boolean attribute
      (<code>disabled</code>, <code>checked</code>, <code>selected</code>,
      <code>readonly</code>, <code>required</code>, <code>multiple</code>,
      <code>autofocus</code>, <code>open</code>), it is <strong>omitted
      entirely</strong> when the value is falsy, and rendered without a value
      when truthy.
    </p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;button :disabled="isLoading"&gt;Submit&lt;/button&gt;
&lt;!-- renders as &lt;button&gt; when isLoading is false --&gt;
&lt;!-- renders as &lt;button disabled&gt; when isLoading is true --&gt;</code></pre>

    <h3>v-bind="obj" — attribute spreading</h3>
    <p>
      When <code>v-bind</code> is used without an attribute name its value must
      evaluate to a <code>map[string]any</code>. Each entry is spread as an HTML
      attribute. <code>class</code> and <code>style</code> keys follow the same
      merge rules. Boolean attribute semantics apply per key.
    </p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- Spread HTMX attributes --&gt;
&lt;button v-bind="htmxAttrs"&gt;Delete&lt;/button&gt;

&lt;!-- Spread props into a child component --&gt;
&lt;Card v-bind="cardProps" :title="override" /&gt;</code></pre>
    <p>
      On child components, explicit <code>:prop</code> bindings take precedence
      over keys in the spread map.
    </p>

    <h2 id="v-html">v-html</h2>
    <p>Sets the element's inner HTML to the expression value. The value is <strong>not</strong> HTML-escaped. Only use with trusted content.</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;div v-html="renderedMarkdown"&gt;&lt;/div&gt;</code></pre>

    <Callout>
      <p><strong>Warning:</strong> Never use <code>v-html</code> with user-supplied data — it can introduce XSS vulnerabilities.</p>
    </Callout>

    <h2 id="v-text">v-text</h2>
    <p>Sets the element's text content to the expression value. HTML-escaped. Replaces all child nodes.</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;span v-text="message"&gt;&lt;/span&gt;
&lt;!-- equivalent to --&gt;
&lt;span&gt;{{ message }}&lt;/span&gt;</code></pre>

    <h2 id="v-pre">v-pre</h2>
    <p>
      Skips all interpolation and directive processing for the element and all
      its descendants. Mustache syntax (<code v-pre>{{ }}</code>) is emitted literally.
      The <code>v-pre</code> attribute itself is stripped from the output.
    </p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- This renders literally: {{ raw }} --&gt;
&lt;code v-pre&gt;{{ raw }}&lt;/code&gt;</code></pre>

    <p>
      Use <code>v-pre</code> to show template syntax as documentation or source
      examples without the engine treating it as an expression.
    </p>

    <h2 id="v-slot">v-slot / #slot</h2>
    <p>Passes content into a named slot of a child component.</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- In Layout.vue --&gt;
&lt;header&gt;&lt;slot name="header" /&gt;&lt;/header&gt;
&lt;main&gt;&lt;slot /&gt;&lt;/main&gt;

&lt;!-- Usage --&gt;
&lt;Layout&gt;
  &lt;template #header&gt;
    &lt;nav&gt;&lt;a href="/"&gt;Home&lt;/a&gt;&lt;/nav&gt;
  &lt;/template&gt;
  &lt;p&gt;Page content goes into the default slot.&lt;/p&gt;
&lt;/Layout&gt;</code></pre>

    <p>Scoped slots (slot props):</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- In List.vue --&gt;
&lt;ul&gt;
  &lt;li v-for="item in items"&gt;
    &lt;slot :item="item"&gt;{{ item }}&lt;/slot&gt;
  &lt;/li&gt;
&lt;/ul&gt;

&lt;!-- Usage --&gt;
&lt;List :items="posts"&gt;
  &lt;template #default="{ item }"&gt;
    &lt;a :href="item.url"&gt;{{ item.title }}&lt;/a&gt;
  &lt;/template&gt;
&lt;/List&gt;</code></pre>

    <h2 id="dynamic-component">&lt;component :is&gt;</h2>
    <p>
      Renders a component whose name is determined at runtime. The <code>:is</code>
      expression must evaluate to a non-empty string naming a registered component
      or a standard HTML element.
    </p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- Resolve from a variable --&gt;
&lt;component :is="activeView" /&gt;

&lt;!-- Inline string literal --&gt;
&lt;component :is="'Card'" :title="pageTitle"&gt;
  &lt;p&gt;slot content&lt;/p&gt;
&lt;/component&gt;

&lt;!-- Switch between components in a loop --&gt;
&lt;div v-for="item in items"&gt;
  &lt;component :is="item.type" :data="item" /&gt;
&lt;/div&gt;</code></pre>
    <ul>
      <li>All attributes other than <code>:is</code> are forwarded as props.</li>
      <li>Default and named slots work exactly as with static component tags.</li>
      <li>
        If the resolved name is a standard HTML element (e.g. <code>"div"</code>),
        it is rendered as a plain tag, not looked up in the component registry.
      </li>
      <li><code v-pre>:is</code> is required; omitting it is a render error.</li>
    </ul>

    <h2 id="not-supported">Stripped directives</h2>
    <p>These directives are parsed but produce no output — they are client-side only and have no meaning in a server-side renderer:</p>
    <ul>
      <li><code v-pre>v-model</code> — two-way binding</li>
      <li><code v-pre>@event</code> / <code>v-on:event</code> — event listeners</li>
      <li><code v-pre>v-once</code> — one-time render optimisation hint</li>
      <li><code v-pre>v-memo</code> — memoisation hint</li>
      <li><code v-pre>v-cloak</code> — FOUC prevention</li>
    </ul>
  </DocsPage>
</template>
