<template>
  <Layout :siteTitle="siteTitle">
    <h1 class="page-title">Posts</h1>
    <div v-if="posts.length === 0" class="empty">
      <p>No posts yet.</p>
    </div>
    <ul v-else class="post-list">
      <li v-for="post in posts" class="post-item">
        <a :href="'/posts/' + post.ID" class="post-title">{{ post.Title }}</a>
        <span class="post-meta">{{ post.PublishedAt }}</span>
      </li>
    </ul>
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

.post-list {
  list-style: none;
}

.post-item {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.75rem 0;
  border-bottom: 1px solid #e0ddd8;
}

.post-item:last-child {
  border-bottom: none;
}

.post-title {
  font-family: Georgia, serif;
  font-size: 1.1rem;
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
