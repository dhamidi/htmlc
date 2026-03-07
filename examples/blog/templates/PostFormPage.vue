<template>
  <Layout>
    <template #header>{{ pageTitle }}</template>
    <div class="editor-shell">
      <nav class="editor-nav">
        <a href="/">← Back</a>
      </nav>
      <PostForm :action="action" :post="post" :submitLabel="submitLabel" />
    </div>
  </Layout>
</template>

<style>
.editor-shell {
  max-width: 720px;
  margin: 0 auto;
}

.editor-nav {
  margin-bottom: 1.5rem;
}

.editor-nav a {
  color: #1a1a1a;
  font-family: inherit;
  font-weight: 700;
  font-size: 0.85rem;
  text-decoration: none;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.editor-nav a:hover {
  color: #c85000;
}
</style>
