<template>
  <AdminLayout :siteTitle="siteTitle">
    <div class="form-shell">
      <div class="form-header">
        <h1 class="form-title">{{ pageTitle }}</h1>
        <a href="/admin/" class="back-link">&larr; Dashboard</a>
      </div>
      <form method="POST" :action="action" class="post-form" id="post-form">
        <div class="field">
          <label for="title" class="field-label">Title</label>
          <input id="title" type="text" name="title" :value="post.Title" required class="field-input" placeholder="Post title" />
        </div>
        <div class="field">
          <label for="slug" class="field-label">Slug</label>
          <input id="slug" type="text" name="slug" :value="post.Slug" class="field-input" placeholder="url-friendly-slug" />
        </div>
        <div class="field">
          <label for="tags" class="field-label">Tags (comma-separated)</label>
          <input id="tags" type="text" name="tags" :value="post.TagsCSV" class="field-input" placeholder="go, tutorial, web" />
        </div>
        <div class="field">
          <label for="body" class="field-label">Body (Markdown)</label>
          <textarea id="body" name="body" rows="18" class="field-textarea" placeholder="Write your post here…">{{ post.Body }}</textarea>
        </div>
        <div class="form-footer">
          <button type="submit" name="publish" value="0" class="submit-btn">Save Draft</button>
          <button type="submit" name="publish" value="1" class="submit-btn submit-btn-publish">Publish</button>
          <a href="/admin/" class="cancel-link">Cancel</a>
        </div>
      </form>
      <form method="POST" action="/admin/posts/preview" target="_blank" class="preview-form">
        <input type="hidden" name="title" :value="post.Title" />
        <input type="hidden" name="body" :value="post.Body" />
        <input type="hidden" name="tags" :value="post.TagsCSV" />
        <button type="submit" class="preview-btn">Preview in new tab</button>
      </form>
    </div>
  </AdminLayout>
</template>

<style>
.form-shell {
  max-width: 720px;
}

.form-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 1.75rem;
  gap: 1rem;
}

.form-title {
  font-family: Georgia, serif;
  font-size: 1.3rem;
}

.back-link {
  font-size: 0.8rem;
  color: #888;
  text-decoration: none;
  white-space: nowrap;
}

.back-link:hover {
  color: #b5451b;
}

.post-form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.field-label {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: #666;
}

.field-input,
.field-textarea {
  background: #fff;
  border: 1px solid #ccc;
  border-radius: 3px;
  color: #1a1a1a;
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.9rem;
  line-height: 1.6;
  padding: 0.6rem 0.8rem;
  width: 100%;
  outline: none;
  resize: vertical;
}

.field-input:focus,
.field-textarea:focus {
  border-color: #1a1a1a;
}

.form-footer {
  display: flex;
  align-items: center;
  gap: 1.5rem;
  margin-top: 0.5rem;
}

.submit-btn {
  background: #1a1a1a;
  border: none;
  border-radius: 3px;
  color: #ffd43b;
  cursor: pointer;
  font-family: inherit;
  font-size: 0.85rem;
  font-weight: bold;
  letter-spacing: 0.04em;
  padding: 0.55rem 1.5rem;
  text-transform: uppercase;
}

.submit-btn:hover {
  background: #333;
}

.submit-btn-publish {
  background: #b5451b;
  color: #fff;
}

.submit-btn-publish:hover {
  background: #d4541f;
}

.cancel-link {
  font-size: 0.85rem;
  color: #888;
}

.cancel-link:hover {
  color: #b5451b;
}

.preview-form {
  margin-top: 1rem;
}

.preview-btn {
  background: none;
  border: 1px solid #ccc;
  border-radius: 3px;
  color: #666;
  cursor: pointer;
  font-family: inherit;
  font-size: 0.8rem;
  padding: 0.4rem 1rem;
}

.preview-btn:hover {
  border-color: #888;
  color: #1a1a1a;
}
</style>
