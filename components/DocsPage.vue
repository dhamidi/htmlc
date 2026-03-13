<template>
  <Layout :fullWidth="true" :pageTitle="pageTitle" :description="description" :siteTitle="siteTitle">

    <div class="docs-layout">
      <aside class="docs-sidebar">
        <template v-if="navItems">
          <template v-for="item in navItems">
            <div v-if="!item.href" class="sidebar-label">{{ item.label }}</div>
            <a v-if="item.href" :href="item.href" class="sidebar-link">{{ item.label }}</a>
          </template>
        </template>
        <slot name="sidebar"></slot>
      </aside>

      <details class="mobile-nav">
        <summary>On this page</summary>
        <template v-if="navItems">
          <template v-for="item in navItems">
            <div v-if="!item.href" class="sidebar-label">{{ item.label }}</div>
            <a v-if="item.href" :href="item.href" class="sidebar-link">{{ item.label }}</a>
          </template>
        </template>
      </details>

      <div class="docs-content">
        <slot></slot>
      </div>
    </div>

  </Layout>
</template>

<script>
export default {
  props: ['pageTitle', 'description', 'siteTitle', 'navItems']
}
</script>

<style>
  p { margin: 1rem 0; }
  ul, ol { padding-left: 1.5rem; margin: 1rem 0; }
  li { margin: 0.25rem 0; }

  .docs-layout {
    display: grid;
    grid-template-columns: 220px 1fr;
    gap: 0;
    max-width: 1200px;
    margin: 0 auto;
    min-height: calc(100vh - var(--nav-height));
  }

  .docs-sidebar {
    border-right: 1px solid var(--border);
    padding: 2rem 1.5rem;
    position: sticky;
    top: var(--nav-height);
    height: calc(100vh - var(--nav-height));
    overflow-y: auto;
  }

  .docs-content {
    padding: 3rem 3rem 5rem;
    min-width: 0;
  }

  .docs-content h1 { font-size: 2.2rem; margin-bottom: 0.75rem; color: #f0f2ff; }
  .docs-content h2 { font-size: 1.4rem; margin: 2.5rem 0 0.75rem; padding-top: 2.5rem; border-top: 1px solid var(--border); color: #e2e4f0; }
  .docs-content h2:first-of-type { border-top: none; padding-top: 0; }
  .docs-content h3 { font-size: 1.1rem; margin: 2rem 0 0.5rem; color: #e2e4f0; }
  .docs-content h4 { margin-top: 1.25rem; margin-bottom: 0.3rem; font-size: 0.95rem; color: #e2e4f0; }

  .sidebar-label {
    font-size: 0.7rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--muted);
    padding: 0.75rem 0.5rem 0.25rem;
  }

  .sidebar-link {
    display: block;
    padding: 0.35rem 0.5rem;
    font-size: 0.875rem;
    color: var(--muted);
    border-radius: 4px;
    text-decoration: none;
    transition: color 0.15s, background 0.15s;
  }

  .sidebar-link:hover {
    color: var(--text);
    background: rgba(255,255,255,0.06);
    text-decoration: none;
  }

  .mobile-nav { display: none; }
  .mobile-nav summary { list-style: none; cursor: pointer; font-size: 0.875rem; font-weight: 600; color: var(--muted); padding: 0.75rem 1rem; background: var(--bg2); border: 1px solid var(--border); border-radius: 8px; margin: 1rem 0; user-select: none; transition: color 0.15s; }
  .mobile-nav summary::-webkit-details-marker { display: none; }
  .mobile-nav[open] summary { color: var(--text); border-bottom-left-radius: 0; border-bottom-right-radius: 0; border-bottom-color: transparent; }
  .mobile-nav[open] { background: var(--bg2); border: 1px solid var(--border); border-radius: 8px; margin: 1rem 0; overflow: hidden; }
  .mobile-nav[open] summary { margin: 0; border: none; border-bottom: 1px solid var(--border); border-radius: 0; }
  .mobile-nav .sidebar-label { font-size: 0.7rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.1em; color: var(--muted); padding: 0.75rem 1rem 0.25rem; }
  .mobile-nav .sidebar-link { display: block; padding: 0.35rem 1rem; font-size: 0.875rem; color: var(--muted); text-decoration: none; transition: color 0.15s, background 0.15s; }
  .mobile-nav .sidebar-link:hover { color: var(--text); background: rgba(255,255,255,0.06); }

  .lead { font-size: 1.1rem; color: var(--muted); margin-bottom: 2rem; }

  @media (max-width: 800px) {
    .docs-layout {
      grid-template-columns: 1fr;
    }
    .docs-sidebar {
      display: none;
    }
    .mobile-nav {
      display: block;
    }
    .docs-content {
      padding: 1.5rem 1rem 3rem;
    }
  }
</style>
