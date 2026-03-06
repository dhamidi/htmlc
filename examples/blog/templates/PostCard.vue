<template>
  <div class="post-card">
    <h2 v-text="title"></h2>
    <p v-text="body"></p>
    <a :href="'/posts/' + id + '/edit'">Edit</a>
  </div>
</template>
