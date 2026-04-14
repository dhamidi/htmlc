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
  padding: 1.5rem;
  border: 2px solid #1a1a1a;
  box-shadow: 4px 4px 0 #1a1a1a;
  margin-bottom: 1.5rem;
  background: #f8f7f4;
  transition: transform 0.1s ease, box-shadow 0.1s ease;
}

.post-card:hover {
  transform: translate(-2px, -2px);
  box-shadow: 6px 6px 0 #1a1a1a;
}

.card-title {
  font-family: Georgia, serif;
  font-size: 1.3rem;
  font-weight: 700;
  line-height: 1.3;
  margin-bottom: 0.35rem;
}

.card-title-link {
  color: #1a1a1a;
  text-decoration: none;
}

.card-title-link:hover {
  color: #1a1a1a;
  text-decoration: underline;
  text-decoration-color: #ffd43b;
  text-decoration-thickness: 3px;
}

.card-meta {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.73rem;
  color: #888;
  margin-bottom: 0.75rem;
  font-weight: 700;
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
  padding-top: 0.75rem;
  border-top: 2px solid #1a1a1a;
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
  font-weight: 700;
  color: #1a1a1a;
  white-space: nowrap;
  text-decoration: none;
  border: 2px solid #1a1a1a;
  padding: 0.2rem 0.6rem;
  box-shadow: 2px 2px 0 #1a1a1a;
  transition: transform 0.1s ease, box-shadow 0.1s ease, background 0.1s ease;
}

.card-read-more:hover {
  text-decoration: none;
  background: #ffd43b;
  transform: translate(-2px, -2px);
  box-shadow: 4px 4px 0 #1a1a1a;
}

.card-actions {
  display: flex;
  gap: 0.5rem;
}
</style>
