<template>
  <Layout :siteTitle="siteTitle" :pageTitle="post.Title" ogType="article" description="">
    <div class="preview-banner">Preview — not published</div>
    <article class="post">
      <header class="post-header">
        <h1 class="post-title">{{ post.Title }}</h1>
        <p class="post-meta">
          {{ post.PublishedAt }}
          &middot;
          {{ post.ReadingTime }} min read
        </p>
        <div class="post-tags">
          <TagBadge v-for="tag in post.Tags" :tag="tag" />
        </div>
      </header>
      <div class="post-body" v-html="post.BodyHTML"></div>
      <footer class="post-footer">
        <a href="/admin/" class="back-link">&larr; Back to dashboard</a>
      </footer>
    </article>
  </Layout>
</template>

<style>
.preview-banner {
  background: #ffd43b;
  color: #1a1a1a;
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.8rem;
  font-weight: bold;
  text-align: center;
  padding: 0.5rem 1rem;
  margin-bottom: 2rem;
  border-radius: 3px;
}

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
  margin-bottom: 0.5rem;
}

.post-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  margin-top: 0.5rem;
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
