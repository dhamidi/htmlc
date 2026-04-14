<template>
  <Layout :siteTitle="siteTitle" pageTitle="Archive" ogType="website" description="">
    <h1 class="page-title">Archive</h1>
    <div v-if="groups.length === 0" class="empty">
      <p>No posts yet.</p>
    </div>
    <div v-else class="archive">
      <section v-for="group in groups" class="archive-group">
        <h2 class="group-label">{{ group.Label }}</h2>
        <ul class="post-list">
          <li v-for="post in group.Posts" class="post-item">
            <a :href="'/posts/' + post.Slug" class="post-title">{{ post.Title }}</a>
            <span class="post-meta">{{ post.PublishedAt }}</span>
          </li>
        </ul>
      </section>
    </div>
  </Layout>
</template>

<style>
.page-title {
  font-family: Georgia, serif;
  font-size: 1.8rem;
  font-weight: 700;
  margin-bottom: 2rem;
  border-bottom: 3px solid #1a1a1a;
  padding-bottom: 0.5rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.empty {
  color: #888;
  font-style: italic;
}

.archive-group {
  margin-bottom: 2.5rem;
}

.group-label {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.9rem;
  font-weight: 700;
  color: #1a1a1a;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  margin-bottom: 0.75rem;
  padding: 0.3rem 0.6rem;
  background: #ffd43b;
  border: 2px solid #1a1a1a;
  display: inline-block;
}

.post-list {
  list-style: none;
  border: 2px solid #1a1a1a;
}

.post-item {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.6rem 0.75rem;
  border-bottom: 2px solid #1a1a1a;
  transition: background 0.1s ease;
}

.post-item:last-child {
  border-bottom: none;
}

.post-item:hover {
  background: #ffd43b;
}

.post-title {
  font-family: Georgia, serif;
  font-size: 1rem;
  font-weight: 700;
  color: #1a1a1a;
  text-decoration: none;
  flex: 1;
}

.post-title:hover {
  text-decoration: underline;
  color: #1a1a1a;
}

.post-meta {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.75rem;
  font-weight: 700;
  color: #555;
  white-space: nowrap;
}
</style>
