<template>
  <form method="POST" :action="action">
    <div class="form-group">
      <label for="title">Title</label>
      <input id="title" name="title" type="text" :value="post.Title" placeholder="Post title" class="title-input" />
    </div>
    <div class="form-group">
      <label for="body">Body</label>
      <textarea id="body" name="body" placeholder="Write your post here…">{{ post.Body }}</textarea>
    </div>
    <button type="submit">{{ submitLabel }}</button>
  </form>
</template>

<style>
form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

label {
  display: block;
  font-weight: 900;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-size: 0.85rem;
  margin-bottom: 0.35rem;
}

input[type="text"],
textarea {
  width: 100%;
  border: 3px solid #000;
  padding: 0.6rem 0.75rem;
  font-size: 1rem;
  font-family: inherit;
  background: #fff;
  color: #000;
  border-radius: 0;
  outline: none;
  box-shadow: inset 2px 2px 0 #eee;
  transition: box-shadow 0.1s;
}

input[type="text"]:focus,
textarea:focus {
  box-shadow: 5px 5px 0 #000;
  outline: none;
}

.title-input {
  font-size: 1.25rem;
  font-weight: 700;
}

textarea {
  min-height: 16rem;
  resize: vertical;
}

button[type="submit"] {
  align-self: flex-start;
  background: #000;
  color: #fff;
  border: 3px solid #000;
  padding: 0.6rem 1.5rem;
  font-size: 1rem;
  font-weight: 900;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  cursor: pointer;
  box-shadow: 4px 4px 0 #000;
}

button[type="submit"]:hover {
  background: #fff;
  color: #000;
  box-shadow: 2px 2px 0 #000;
  transform: translate(2px, 2px);
}
</style>
