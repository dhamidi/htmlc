<template>
  <Layout :siteTitle="siteTitle" :pageTitle="post.Title" :description="post.ExcerptText" ogType="article">
    <article class="post">
      <header class="post-header">
        <h1 class="post-title">{{ post.Title }}</h1>
        <p class="post-meta">
          <template v-switch="post.Published">
            <span v-case="true">Published {{ post.PublishedAt }}</span>
            <span v-default>Draft</span>
          </template>
          &middot;
          {{ post.ReadingTime }} min read
          &middot;
          {{ post.Impressions }} view<span v-if="post.Impressions !== 1">s</span>
        </p>
        <div class="post-tags" v-if="post.Tags.length > 0">
          <TagBadge v-for="tag in post.Tags" :tag="tag" />
        </div>
      </header>
      <div class="post-body" v-html="post.BodyHTML"></div>
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
  padding: 1.5rem;
  padding-bottom: 1.5rem;
  border: 3px solid #1a1a1a;
  box-shadow: 5px 5px 0 #1a1a1a;
  background: #fff;
}

.post-title {
  font-family: Georgia, serif;
  font-size: 2rem;
  font-weight: 700;
  line-height: 1.25;
  margin-bottom: 0.75rem;
}

.post-meta {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.78rem;
  color: #888;
  font-weight: 700;
  margin-bottom: 0.5rem;
}

.post-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  margin-top: 0.75rem;
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
  font-weight: 700;
  margin: 2rem 0 0.75rem;
  border-bottom: 2px solid #1a1a1a;
  padding-bottom: 0.25rem;
}

.post-footer {
  margin-top: 3rem;
  padding-top: 1.5rem;
  border-top: 3px solid #1a1a1a;
}

.back-link {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.85rem;
  font-weight: 700;
  color: #1a1a1a;
  text-decoration: none;
  border: 2px solid #1a1a1a;
  padding: 0.3rem 0.7rem;
  box-shadow: 3px 3px 0 #1a1a1a;
  transition: transform 0.1s ease, box-shadow 0.1s ease, background 0.1s ease;
  display: inline-block;
}

.back-link:hover {
  text-decoration: none;
  background: #ffd43b;
  transform: translate(-2px, -2px);
  box-shadow: 5px 5px 0 #1a1a1a;
}
</style>
