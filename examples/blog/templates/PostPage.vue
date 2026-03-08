<template>
  <Layout :siteTitle="siteTitle">
    <article class="post">
      <header class="post-header">
        <h1 class="post-title">{{ post.Title }}</h1>
        <p class="post-meta">
          Published {{ post.PublishedAt }}
          &middot;
          {{ post.Impressions }} view<span v-if="post.Impressions !== 1">s</span>
        </p>
      </header>
      <div class="post-body" v-html="post.Body"></div>
      <footer class="post-footer">
        <a href="/" class="back-link">&larr; All posts</a>
      </footer>
    </article>
  </Layout>
</template>

<style>
.post {
  max-width: 100%;
}

.post-header {
  margin-bottom: 2rem;
  padding-bottom: 1.5rem;
  border-bottom: 2px solid #1a1a1a;
}

.post-title {
  font-family: Georgia, serif;
  font-size: 2rem;
  line-height: 1.25;
  margin-bottom: 0.75rem;
}

.post-meta {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.78rem;
  color: #888;
}

.post-body {
  font-size: 1.05rem;
  line-height: 1.8;
}

.post-body p {
  margin-bottom: 1.25rem;
}

.post-body h2,
.post-body h3 {
  font-family: Georgia, serif;
  margin: 2rem 0 0.75rem;
}

.post-footer {
  margin-top: 3rem;
  padding-top: 1.5rem;
  border-top: 1px solid #e0ddd8;
}

.back-link {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.85rem;
  color: #888;
}

.back-link:hover {
  color: #b5451b;
}
</style>
