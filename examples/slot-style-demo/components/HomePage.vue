<template>
  <Layout>
    <template #header>
      <h1 class="page-title">{{ title }}</h1>
      <nav class="page-nav">
        <a v-for="link in links" :href="link.url">{{ link.label }}</a>
      </nav>
    </template>
    <Card v-for="post in posts">
      <h2 class="post-title">{{ post.title }}</h2>
      <p class="post-body">{{ post.body }}</p>
    </Card>
  </Layout>
</template>
<style scoped>
.page-title { font-size: 2rem; margin: 0; }
.page-nav   { display: flex; gap: 1rem; }
.post-title { font-size: 1.25rem; margin: 0 0 .5rem; }
.post-body  { color: #555; margin: 0; }
</style>
