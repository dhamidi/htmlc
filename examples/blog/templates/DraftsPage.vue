<template>
  <AdminLayout :siteTitle="siteTitle">
    <h1 class="section-title">Drafts</h1>
    <div v-if="posts.length === 0" class="empty">
      <p>No drafts. <a href="/admin/posts/new">Write something</a>.</p>
    </div>
    <ul v-else class="draft-list">
      <li v-for="post in posts" class="draft-item">
        <div class="draft-info">
          <span class="draft-title">{{ post.Title }}</span>
          <span class="draft-date">Created {{ post.CreatedAt }}</span>
        </div>
        <div class="draft-actions">
          <a :href="'/admin/posts/' + post.ID + '/edit'" class="action-link">Edit</a>
          <form method="POST" :action="'/admin/posts/' + post.ID + '/publish'" class="inline-form">
            <button type="submit" class="action-btn action-btn-primary">Publish</button>
          </form>
          <form method="POST" :action="'/admin/posts/' + post.ID + '/delete'" class="inline-form">
            <button type="submit" class="action-btn action-btn-danger">Delete</button>
          </form>
        </div>
      </li>
    </ul>
  </AdminLayout>
</template>

<style>
.section-title {
  font-size: 1.3rem;
  margin-bottom: 1.75rem;
  font-family: Georgia, serif;
}

.empty {
  color: #888;
}

.draft-list {
  list-style: none;
}

.draft-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.9rem 0;
  border-bottom: 1px solid #ddd;
}

.draft-item:last-child {
  border-bottom: none;
}

.draft-info {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.draft-title {
  font-size: 1rem;
  color: #1a1a1a;
}

.draft-date {
  font-size: 0.75rem;
  color: #888;
}

.draft-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

.action-link {
  font-size: 0.8rem;
  color: #555;
  text-decoration: none;
}

.action-link:hover {
  color: #b5451b;
}

.inline-form {
  display: inline;
  margin: 0;
}

.action-btn {
  background: none;
  border: 1px solid #ccc;
  border-radius: 2px;
  color: #555;
  cursor: pointer;
  font-family: inherit;
  font-size: 0.78rem;
  padding: 0.15rem 0.5rem;
}

.action-btn:hover {
  border-color: #888;
  color: #1a1a1a;
}

.action-btn-primary {
  border-color: #28a745;
  color: #28a745;
}

.action-btn-primary:hover {
  background: #28a745;
  color: #fff;
}

.action-btn-danger {
  border-color: #dc3545;
  color: #dc3545;
}

.action-btn-danger:hover {
  background: #dc3545;
  color: #fff;
}
</style>
