<template>
  <div class="post-list">
    <div v-if="posts.length === 0" class="post-list-empty">
      <p>No posts yet.</p>
      <a href="/posts/new">Write your first post</a>
    </div>
    <ul v-else>
      <li v-for="(post, index) in posts">
        <slot name="item" :post="post" :index="index">{{ post.Title }}</slot>
      </li>
    </ul>
  </div>
</template>

<style>
.post-list ul {
  list-style: none;
  padding: 0;
  margin: 0;
}

.post-list li {
  padding: 0;
}

.post-list-empty {
  text-align: center;
  padding: 3rem 2rem;
  border: 2px dashed #ccc;
  background: #fff;
}

.post-list-empty p {
  font-size: 1.1rem;
  color: #888;
  margin-bottom: 1rem;
}

.post-list-empty a {
  display: inline-block;
  background: #1a1a1a;
  color: #fff;
  padding: 0.5rem 1.2rem;
  font-family: inherit;
  font-size: 0.85rem;
  font-weight: 900;
  text-transform: uppercase;
  text-decoration: none;
  letter-spacing: 0.04em;
}

.post-list-empty a:hover {
  background: #c85000;
}
</style>
