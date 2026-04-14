<template>
  <a :href="'/tags/' + tag" class="tag-badge">{{ tag }}</a>
</template>

<style scoped>
.tag-badge {
  display: inline-block;
  background: #ffd43b;
  color: #1a1a1a;
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.7rem;
  font-weight: 700;
  padding: 0.15rem 0.5rem;
  border: 2px solid #1a1a1a;
  text-decoration: none;
  white-space: nowrap;
  box-shadow: 2px 2px 0 #1a1a1a;
  transition: transform 0.1s ease, box-shadow 0.1s ease, background 0.1s ease;
}

.tag-badge:hover {
  text-decoration: none;
  background: #1a1a1a;
  color: #ffd43b;
  transform: translate(-1px, -1px);
  box-shadow: 3px 3px 0 #555;
}
</style>
