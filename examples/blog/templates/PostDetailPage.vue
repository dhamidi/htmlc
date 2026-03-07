<template>
  <Layout>
    <template #header>{{ post.Title }}</template>
    <div class="post-detail">
      <nav class="post-detail-nav">
        <a href="/" class="post-detail-back">← All Posts</a>
        <a :href="'/posts/' + post.ID + '/edit'" class="post-detail-edit">Edit</a>
      </nav>
      <article class="post-body">
        <h1 class="post-body-title" v-text="post.Title"></h1>
        <div class="post-body-text" v-text="post.Body"></div>
      </article>
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
  align-items: center;
  margin-bottom: 2rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid #e0e0e0;
}

.post-detail-back {
  color: #1a1a1a;
  font-family: inherit;
  font-weight: 700;
  font-size: 0.85rem;
  text-decoration: none;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.post-detail-back:hover {
  color: #c85000;
}

.post-detail-edit {
  color: #888;
  font-family: inherit;
  font-size: 0.8rem;
  font-weight: 700;
  text-decoration: none;
}

.post-detail-edit:hover {
  color: #1a1a1a;
}

.post-body-title {
  font-family: inherit;
  font-size: 2rem;
  font-weight: 900;
  line-height: 1.2;
  margin-bottom: 1.5rem;
}

.post-body-text {
  font-size: 1.1rem;
  line-height: 1.85;
  color: #333;
  white-space: pre-line;
}
</style>
