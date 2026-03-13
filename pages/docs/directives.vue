<template>
  <Layout pageTitle="Directives — htmlc.sh" description="Full reference for all htmlc template directives: v-if, v-for, v-bind, v-show, v-html, v-text, v-switch, v-slot." :siteTitle="siteTitle">

    <div class="docs-layout">
      <aside class="docs-sidebar">
        <SidebarSection label="Conditionals">
          <a href="#v-if" class="sidebar-link">v-if / v-else-if / v-else</a>
          <a href="#v-show" class="sidebar-link">v-show</a>
          <a href="#v-switch" class="sidebar-link">v-switch / v-case</a>
        </SidebarSection>
        <SidebarSection label="Lists">
          <a href="#v-for" class="sidebar-link">v-for</a>
        </SidebarSection>
        <SidebarSection label="Binding">
          <a href="#v-bind" class="sidebar-link">v-bind / :attr</a>
          <a href="#v-html" class="sidebar-link">v-html</a>
          <a href="#v-text" class="sidebar-link">v-text</a>
          <a href="#v-pre" class="sidebar-link">v-pre</a>
        </SidebarSection>
        <SidebarSection label="Components">
          <a href="#v-slot" class="sidebar-link">v-slot / #slot</a>
          <a href="#dynamic-component" class="sidebar-link">component :is</a>
        </SidebarSection>
        <SidebarSection label="Not supported">
          <a href="#not-supported" class="sidebar-link">Stripped directives</a>
        </SidebarSection>
      </aside>

      <div class="docs-content">
        <h1>Directives</h1>
        <p class="lead">Full reference for all template directives supported by htmlc.</p>

        <h2 id="v-if">v-if / v-else-if / v-else</h2>
        <p>Renders the element only when the expression is truthy. Whitespace-only text nodes between branches are ignored.</p>
        <pre><code>&lt;p v-if="role === 'admin'"&gt;Admin panel&lt;/p&gt;
&lt;p v-else-if="role === 'editor'"&gt;Editor view&lt;/p&gt;
&lt;p v-else&gt;Read-only view&lt;/p&gt;</code></pre>

        <p>Works on <code>&lt;template&gt;</code> elements too (renders children only, no wrapper element):</p>
        <pre><code>&lt;template v-if="items.length &gt; 0"&gt;
  &lt;ul&gt;
    &lt;li v-for="item in items"&gt;{{ "{{" }} item }}&lt;/li&gt;
  &lt;/ul&gt;
&lt;/template&gt;
&lt;template v-else&gt;
  &lt;p&gt;No items.&lt;/p&gt;
&lt;/template&gt;</code></pre>

        <h2 id="v-show">v-show</h2>
        <p>Adds <code>style="display:none"</code> when the expression is falsy. The element is always rendered (unlike <code>v-if</code>). Merges with any existing <code>style</code> attribute.</p>
        <pre><code>&lt;div v-show="isVisible"&gt;Visible when isVisible is truthy&lt;/div&gt;</code></pre>

        <h2 id="v-switch">v-switch / v-case / v-default</h2>
        <p>Switch/case conditional (implements Vue RFC #482). Must be on a <code>&lt;template&gt;</code> element. Renders the first matching <code>v-case</code> branch.</p>
        <pre><code>&lt;template v-switch="status"&gt;
  &lt;div v-case="'active'"&gt;Active&lt;/div&gt;
  &lt;div v-case="'pending'"&gt;Pending approval&lt;/div&gt;
  &lt;div v-default&gt;Unknown status&lt;/div&gt;
&lt;/template&gt;</code></pre>

        <h2 id="v-for">v-for</h2>
        <p>Repeats the element for each item in the iterable. Supports arrays, maps, and objects.</p>
        <pre><code>&lt;!-- Array --&gt;
&lt;li v-for="item in items"&gt;{{ "{{" }} item }}&lt;/li&gt;

&lt;!-- With index --&gt;
&lt;li v-for="(item, index) in items"&gt;{{ "{{" }} index }}: {{ "{{" }} item }}&lt;/li&gt;

&lt;!-- Object/map --&gt;
&lt;li v-for="(value, key) in obj"&gt;{{ "{{" }} key }}: {{ "{{" }} value }}&lt;/li&gt;

&lt;!-- Range (integer) --&gt;
&lt;li v-for="i in 5"&gt;{{ "{{" }} i }}&lt;/li&gt;</code></pre>

        <Callout>
          <p><strong>Note:</strong> Map iteration order follows Go's <code>reflect.MapKeys()</code> — not insertion order. Sort your maps before passing them if order matters.</p>
        </Callout>

        <h2 id="v-bind">v-bind / :attr</h2>
        <p>Dynamically binds an HTML attribute to an expression. The shorthand is <code>:</code>.</p>
        <pre><code>&lt;!-- Long form --&gt;
&lt;a v-bind:href="url"&gt;Link&lt;/a&gt;

&lt;!-- Shorthand --&gt;
&lt;a :href="url"&gt;Link&lt;/a&gt;
&lt;img :src="imageUrl" :alt="imageAlt" /&gt;

&lt;!-- Boolean attributes: rendered only when truthy --&gt;
&lt;button :disabled="isLoading"&gt;Submit&lt;/button&gt;

&lt;!-- Class binding --&gt;
&lt;div :class="isActive ? 'active' : ''"&gt;...&lt;/div&gt;</code></pre>

        <p>When passing props to a component, <code>:propName</code> evaluates the expression:</p>
        <pre><code>&lt;Card :title="post.title" :author="post.author" /&gt;</code></pre>

        <h3>:class — object and array syntax</h3>
        <pre><code>&lt;!-- Object: keys with truthy values are included --&gt;
&lt;div :class="{ active: isActive, disabled: !isEnabled }"&gt;...&lt;/div&gt;

&lt;!-- Array: non-empty string elements are included --&gt;
&lt;div :class="['btn', isPrimary ? 'primary' : '']"&gt;...&lt;/div&gt;

&lt;!-- Static class and :class are merged --&gt;
&lt;div class="card" :class="{ featured: post.featured }"&gt;...&lt;/div&gt;</code></pre>

        <h3>:style — object syntax</h3>
        <pre><code>&lt;!-- camelCase keys are converted to kebab-case in output --&gt;
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
        <pre><code>&lt;button :disabled="isLoading"&gt;Submit&lt;/button&gt;
&lt;!-- renders as &lt;button&gt; when isLoading is false --&gt;
&lt;!-- renders as &lt;button disabled&gt; when isLoading is true --&gt;</code></pre>

        <h3>v-bind="obj" — attribute spreading</h3>
        <p>
          When <code>v-bind</code> is used without an attribute name its value must
          evaluate to a <code>map[string]any</code>. Each entry is spread as an HTML
          attribute. <code>class</code> and <code>style</code> keys follow the same
          merge rules. Boolean attribute semantics apply per key.
        </p>
        <pre><code>&lt;!-- Spread HTMX attributes --&gt;
&lt;button v-bind="htmxAttrs"&gt;Delete&lt;/button&gt;

&lt;!-- Spread props into a child component --&gt;
&lt;Card v-bind="cardProps" :title="override" /&gt;</code></pre>
        <p>
          On child components, explicit <code>:prop</code> bindings take precedence
          over keys in the spread map.
        </p>

        <h2 id="v-html">v-html</h2>
        <p>Sets the element's inner HTML to the expression value. The value is <strong>not</strong> HTML-escaped. Only use with trusted content.</p>
        <pre><code>&lt;div v-html="renderedMarkdown"&gt;&lt;/div&gt;</code></pre>

        <Callout>
          <p><strong>Warning:</strong> Never use <code>v-html</code> with user-supplied data — it can introduce XSS vulnerabilities.</p>
        </Callout>

        <h2 id="v-text">v-text</h2>
        <p>Sets the element's text content to the expression value. HTML-escaped. Replaces all child nodes.</p>
        <pre><code>&lt;span v-text="message"&gt;&lt;/span&gt;
&lt;!-- equivalent to --&gt;
&lt;span&gt;{{ "{{" }} message }}&lt;/span&gt;</code></pre>

        <h2 id="v-pre">v-pre</h2>
        <p>
          Skips all interpolation and directive processing for the element and all
          its descendants. Mustache syntax (<code>{{ "{{" }} }}</code>) is emitted literally.
          The <code>v-pre</code> attribute itself is stripped from the output.
        </p>
        <pre><code>&lt;!-- This renders literally: {{ "{{" }} raw }} --&gt;
&lt;code v-pre&gt;{{ "{{" }} raw }}&lt;/code&gt;</code></pre>

        <p>
          Use <code>v-pre</code> to show template syntax as documentation or source
          examples without the engine treating it as an expression.
        </p>

        <h2 id="v-slot">v-slot / #slot</h2>
        <p>Passes content into a named slot of a child component.</p>
        <pre><code>&lt;!-- In Layout.vue --&gt;
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
        <pre><code>&lt;!-- In List.vue --&gt;
&lt;ul&gt;
  &lt;li v-for="item in items"&gt;
    &lt;slot :item="item"&gt;{{ "{{" }} item }}&lt;/slot&gt;
  &lt;/li&gt;
&lt;/ul&gt;

&lt;!-- Usage --&gt;
&lt;List :items="posts"&gt;
  &lt;template #default="{ item }"&gt;
    &lt;a :href="item.url"&gt;{{ "{{" }} item.title }}&lt;/a&gt;
  &lt;/template&gt;
&lt;/List&gt;</code></pre>

        <h2 id="dynamic-component">&lt;component :is&gt;</h2>
        <p>
          Renders a component whose name is determined at runtime. The <code>:is</code>
          expression must evaluate to a non-empty string naming a registered component
          or a standard HTML element.
        </p>
        <pre><code>&lt;!-- Resolve from a variable --&gt;
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
          <li><code>:is</code> is required; omitting it is a render error.</li>
        </ul>

        <h2 id="not-supported">Stripped directives</h2>
        <p>These directives are parsed but produce no output — they are client-side only and have no meaning in a server-side renderer:</p>
        <ul>
          <li><code>v-model</code> — two-way binding</li>
          <li><code>@event</code> / <code>v-on:event</code> — event listeners</li>
          <li><code>v-once</code> — one-time render optimisation hint</li>
          <li><code>v-memo</code> — memoisation hint</li>
          <li><code>v-cloak</code> — FOUC prevention</li>
        </ul>
      </div>
    </div>

  </Layout>
</template>

<style>
  p { margin: 1rem 0; }
  ul, ol { padding-left: 1.5rem; margin: 1rem 0; }
  li { margin: 0.25rem 0; }

  .docs-layout { display: grid; grid-template-columns: 220px 1fr; gap: 0; max-width: 1200px; margin: 0 auto; }
  @media (max-width: 800px) { .docs-layout { grid-template-columns: 1fr; } .docs-sidebar { display: none; } }
  .docs-sidebar { border-right: 1px solid var(--border); padding: 2rem 1.5rem; position: sticky; top: var(--nav-height); height: calc(100vh - var(--nav-height)); overflow-y: auto; }
  .docs-content { padding: 3rem 3rem 5rem; max-width: 800px; }
  .docs-content h1 { font-size: 2.2rem; margin-bottom: 0.75rem; color: #f0f2ff; }
  .docs-content h2 { font-size: 1.4rem; margin: 2.5rem 0 0.75rem; padding-top: 2.5rem; border-top: 1px solid var(--border); }
  .docs-content h2:first-of-type { border-top: none; padding-top: 0; }
  .docs-content h3 { font-size: 1.1rem; margin: 2rem 0 0.5rem; color: #e2e4f0; }
  .lead { font-size: 1.1rem; color: var(--muted); margin-bottom: 2rem; }
</style>
