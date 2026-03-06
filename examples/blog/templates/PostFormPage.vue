<template>
  <Layout>
    <template #header>{{ pageTitle }}</template>
    <div class="editor-shell">
      <PostForm :action="action" :post="post" :submitLabel="submitLabel" />
    </div>
  </Layout>
</template>

<style>
.editor-shell {
  max-width: 720px;
  margin: 0 auto;
}
</style>
