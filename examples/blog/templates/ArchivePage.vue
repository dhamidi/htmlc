<template>
  <Layout :siteTitle="siteTitle">
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
  font-size: 1.6rem;
  margin-bottom: 2rem;
  border-bottom: 2px solid #1a1a1a;
  padding-bottom: 0.5rem;
}

.empty {
  color: #888;
  font-style: italic;
}

.archive-group {
  margin-bottom: 2.5rem;
}

.group-label {
  font-family: Georgia, serif;
  font-size: 1.15rem;
  color: #888;
  font-weight: normal;
  margin-bottom: 0.75rem;
  padding-bottom: 0.4rem;
  border-bottom: 1px solid #e0ddd8;
}

.post-list {
  list-style: none;
}

.post-item {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.6rem 0;
  border-bottom: 1px solid #f0ede8;
}

.post-item:last-child {
  border-bottom: none;
}

.post-title {
  font-family: Georgia, serif;
  font-size: 1rem;
  color: #1a1a1a;
  text-decoration: none;
  flex: 1;
}

.post-title:hover {
  color: #b5451b;
}

.post-meta {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.75rem;
  color: #888;
  white-space: nowrap;
}
</style>
