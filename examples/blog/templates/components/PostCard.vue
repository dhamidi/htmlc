<template>
  <article class="post-card">
    <header class="card-header">
      <h2 class="card-title">
        <a :href="'/posts/' + post.Slug" class="card-title-link">{{ post.Title }}</a>
      </h2>
      <p class="card-meta">
        <span class="card-date">{{ post.PublishedAt }}</span>
        <span class="card-sep">&middot;</span>
        <span class="card-reading-time">{{ post.ReadingTime }} min read</span>
      </p>
    </header>
    <div class="card-excerpt" v-html="post.ExcerptHTML"></div>
    <footer class="card-footer">
      <div class="card-tags">
        <TagBadge v-for="tag in post.Tags" :tag="tag" />
      </div>
      <a :href="'/posts/' + post.Slug" class="card-read-more">Read more &rarr;</a>
      <div class="card-actions">
        <slot name="actions"></slot>
      </div>
    </footer>
  </article>
</template>

<style scoped>
.post-card {
  padding: 1.5rem 0;
  border-bottom: 1px solid #e0ddd8;
}

.post-card:last-child {
  border-bottom: none;
}

.card-title {
  font-family: Georgia, serif;
  font-size: 1.3rem;
  line-height: 1.3;
  margin-bottom: 0.35rem;
}

.card-title-link {
  color: #1a1a1a;
  text-decoration: none;
}

.card-title-link:hover {
  color: #b5451b;
}

.card-meta {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.73rem;
  color: #888;
  margin-bottom: 0.75rem;
}

.card-sep {
  margin: 0 0.4rem;
}

.card-excerpt {
  font-size: 1rem;
  line-height: 1.7;
  color: #444;
  margin-bottom: 0.9rem;
}

.card-footer {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}

.card-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  flex: 1;
}

.card-read-more {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.78rem;
  color: #b5451b;
  white-space: nowrap;
}

.card-read-more:hover {
  text-decoration: underline;
}

.card-actions {
  display: flex;
  gap: 0.5rem;
}
</style>
