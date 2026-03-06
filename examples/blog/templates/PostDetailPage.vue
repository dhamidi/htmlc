<template>
  <Layout>
    <template #header>{{ post.Title }}</template>
    <div class="post-detail">
      <nav class="post-detail-nav">
        <a href="/">← Back</a>
        <a :href="'/posts/' + post.ID + '/edit'">Edit</a>
      </nav>
      <article class="post-body" v-text="post.Body"></article>
    </div>
  </Layout>
</template>

<style>
.post-detail {
  max-width: 680px;
}

.post-detail-nav {
  display: flex;
  gap: 1rem;
  margin-bottom: 2rem;
}

.post-detail-nav a {
  color: #000;
  font-weight: 700;
  text-decoration: none;
  border-bottom: 2px solid #000;
}

.post-detail-nav a:hover {
  background: #000;
  color: #fff;
}

.post-body {
  font-size: 1.125rem;
  line-height: 1.8;
}

.post-body p {
  margin-bottom: 1.5rem;
}
</style>
