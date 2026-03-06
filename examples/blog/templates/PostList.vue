<template>
  <div class="post-list">
    <p v-if="posts.length === 0">No posts yet.</p>
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
  border-bottom: 2px solid #000;
}

.post-list li:last-child {
  border-bottom: none;
}

.post-list p {
  font-size: 1.2rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 2rem 0;
  border: 3px solid #000;
  text-align: center;
}
</style>
