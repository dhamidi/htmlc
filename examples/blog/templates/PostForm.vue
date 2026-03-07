<template>
  <form method="POST" :action="action" class="post-form">
    <div class="post-form-group">
      <label for="title" class="post-form-label">Title</label>
      <input id="title" name="title" type="text" :value="post.Title" placeholder="Post title" class="post-form-input post-form-input--title" />
    </div>
    <div class="post-form-group">
      <label for="body" class="post-form-label">Body</label>
      <textarea id="body" name="body" placeholder="Write your post here…" class="post-form-textarea">{{ post.Body }}</textarea>
    </div>
    <button type="submit" class="post-form-submit">{{ submitLabel }}</button>
  </form>
</template>

<style>
.post-form {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  background: #fff;
  border: 2px solid #1a1a1a;
  padding: 2rem;
  box-shadow: 4px 4px 0 #1a1a1a;
}

.post-form-label {
  display: block;
  font-family: inherit;
  font-weight: 900;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-size: 0.75rem;
  margin-bottom: 0.4rem;
  color: #666;
}

.post-form-input,
.post-form-textarea {
  display: block;
  width: 100%;
  border: 2px solid #ddd;
  padding: 0.7rem 0.85rem;
  font-size: 1rem;
  font-family: inherit;
  background: #fafafa;
  color: #1a1a1a;
  border-radius: 0;
  outline: none;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.post-form-input:focus,
.post-form-textarea:focus {
  border-color: #1a1a1a;
  box-shadow: 3px 3px 0 #e0e0e0;
  background: #fff;
}

.post-form-input--title {
  font-size: 1.3rem;
  font-weight: 700;
}

.post-form-textarea {
  min-height: 18rem;
  resize: vertical;
  line-height: 1.7;
}

.post-form-submit {
  align-self: flex-start;
  background: #1a1a1a;
  color: #fff;
  border: 2px solid #1a1a1a;
  padding: 0.65rem 1.8rem;
  font-family: inherit;
  font-size: 0.85rem;
  font-weight: 900;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  cursor: pointer;
  transition: all 0.15s ease;
}

.post-form-submit:hover {
  background: #c85000;
  border-color: #c85000;
}
</style>
