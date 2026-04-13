<template>
  <a :href="'/tags/' + tag" class="tag-badge">{{ tag }}</a>
</template>

<style scoped>
.tag-badge {
  display: inline-block;
  background: #f0ede8;
  color: #555;
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.7rem;
  padding: 0.15rem 0.5rem;
  border-radius: 2px;
  text-decoration: none;
  white-space: nowrap;
}

.tag-badge:hover {
  background: #1a1a1a;
  color: #ffd43b;
}
</style>
