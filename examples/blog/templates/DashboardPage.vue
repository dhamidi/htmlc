<template>
  <AdminLayout :siteTitle="siteTitle">
    <h1 class="section-title">Dashboard</h1>
    <div v-if="posts.length === 0" class="empty">
      <p>No posts yet. <a href="/admin/posts/new">Create one</a>.</p>
    </div>
    <table v-else class="posts-table">
      <thead>
        <tr>
          <th>Title</th>
          <th>Slug</th>
          <th>Status</th>
          <th>Views</th>
          <th>Created</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="post in posts">
          <td class="col-title">
            <a v-if="post.Published" :href="'/posts/' + post.Slug">{{ post.Title }}</a>
            <span v-else class="draft-title">{{ post.Title }}</span>
          </td>
          <td class="col-slug">
            <code class="post-slug">{{ post.Slug }}</code>
          </td>
          <td class="col-status">
            <span v-if="post.Published" class="badge badge-published">published</span>
            <span v-else class="badge badge-draft">draft</span>
          </td>
          <td class="col-views">{{ post.Impressions }}</td>
          <td class="col-date">{{ post.CreatedAt }}</td>
          <td class="col-actions">
            <a :href="'/admin/posts/' + post.ID + '/edit'" class="action-link">Edit</a>
            <form v-if="post.Published" method="POST" :action="'/admin/posts/' + post.ID + '/unpublish'" class="inline-form">
              <button type="submit" class="action-btn">Unpublish</button>
            </form>
            <form v-else method="POST" :action="'/admin/posts/' + post.ID + '/publish'" class="inline-form">
              <button type="submit" class="action-btn action-btn-primary">Publish</button>
            </form>
            <form method="POST" :action="'/admin/posts/' + post.ID + '/delete'" class="inline-form">
              <button type="submit" class="action-btn action-btn-danger">Delete</button>
            </form>
          </td>
        </tr>
      </tbody>
    </table>
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

.posts-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.posts-table th {
  text-align: left;
  padding: 0.5rem 0.75rem;
  border-bottom: 2px solid #1a1a1a;
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: #555;
}

.posts-table td {
  padding: 0.65rem 0.75rem;
  border-bottom: 1px solid #ddd;
  vertical-align: middle;
}

.posts-table tr:last-child td {
  border-bottom: none;
}

.col-title {
  width: 40%;
}

.draft-title {
  color: #888;
}

.badge {
  display: inline-block;
  font-size: 0.7rem;
  padding: 0.15rem 0.5rem;
  border-radius: 2px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: bold;
}

.badge-published {
  background: #d4edda;
  color: #155724;
}

.badge-draft {
  background: #e2e3e5;
  color: #495057;
}

.col-views,
.col-date {
  white-space: nowrap;
  color: #888;
}

.col-actions {
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
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
