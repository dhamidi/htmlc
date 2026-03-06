<template>
  <form method="POST" :action="action">
    <div>
      <label for="title">Title</label>
      <input id="title" name="title" type="text" :value="post.Title" />
    </div>
    <div>
      <label for="body">Body</label>
      <textarea id="body" name="body">{{ post.Body }}</textarea>
    </div>
    <button type="submit">{{ submitLabel }}</button>
  </form>
</template>
