<template>
  <article class="post-card">
    <div class="post-card-content">
      <h2 class="post-card-title"><a :href="'/posts/' + id" v-text="title"></a></h2>
      <p class="post-card-body" v-text="body"></p>
    </div>
    <div class="post-card-footer">
      <a :href="'/posts/' + id" class="post-card-read">Read more</a>
      <a :href="'/posts/' + id + '/edit'" class="post-card-edit">Edit</a>
    </div>
  </article>
</template>

<style>
.post-card {
  border: 2px solid #1a1a1a;
  background: #fff;
  margin-bottom: 1.5rem;
  transition: box-shadow 0.15s ease, transform 0.15s ease;
  box-shadow: 4px 4px 0 #1a1a1a;
  display: flex;
  flex-direction: column;
}

.post-card:hover {
  box-shadow: 6px 6px 0 #1a1a1a;
  transform: translate(-2px, -2px);
}

.post-card-content {
  padding: 1.5rem 1.5rem 1rem;
  flex: 1;
}

.post-card-title {
  font-size: 1.35rem;
  font-weight: 900;
  font-family: inherit;
  margin-bottom: 0.6rem;
  line-height: 1.3;
}

.post-card-title a {
  color: #1a1a1a;
  text-decoration: none;
}

.post-card-title a:hover {
  color: #c85000;
}

.post-card-body {
  color: #444;
  line-height: 1.65;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
  font-size: 0.95rem;
}

.post-card-footer {
  border-top: 1px solid #e5e5e5;
  padding: 0.75rem 1.5rem;
  display: flex;
  gap: 1rem;
  align-items: center;
}

.post-card-read {
  font-family: inherit;
  font-size: 0.8rem;
  font-weight: 900;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #1a1a1a;
  text-decoration: none;
  padding: 0.3rem 0.7rem;
  background: #ffd43b;
}

.post-card-read:hover {
  background: #1a1a1a;
  color: #fff;
}

.post-card-edit {
  font-family: inherit;
  font-size: 0.8rem;
  font-weight: 700;
  color: #888;
  text-decoration: none;
  letter-spacing: 0.03em;
}

.post-card-edit:hover {
  color: #1a1a1a;
}
</style>
