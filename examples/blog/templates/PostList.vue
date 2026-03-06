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
