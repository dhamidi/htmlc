<template>
  <div class="post-card">
    <h2 v-text="title"></h2>
    <p v-text="body"></p>
    <a :href="'/posts/' + id + '/edit'">Edit</a>
  </div>
</template>

<style>
.post-card {
  border: 3px solid #000;
  box-shadow: 5px 5px 0 #000;
  padding: 1.25rem;
  background: #fff;
  margin-bottom: 1.5rem;
}

.post-card:hover {
  box-shadow: 3px 3px 0 #000;
  transform: translate(2px, 2px);
}

.post-card h2 {
  font-size: 1.4rem;
  font-weight: 900;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  margin-bottom: 0.5rem;
}

.post-card p {
  margin-bottom: 1rem;
  line-height: 1.5;
}

.post-card a {
  display: inline-block;
  background: #000;
  color: #fff;
  padding: 0.3rem 0.75rem;
  font-weight: 700;
  text-decoration: none;
  border: 2px solid #000;
}

.post-card a:hover {
  background: #fff;
  color: #000;
}
</style>
