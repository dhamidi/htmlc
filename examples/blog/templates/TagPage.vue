<template>
  <Layout :siteTitle="siteTitle" :pageTitle="'Tag: ' + tag" ogType="website" description="">
    <h1 class="page-title">Posts tagged &ldquo;{{ tag }}&rdquo;</h1>
    <div v-if="posts.length === 0" class="empty">
      <p>No posts with this tag.</p>
    </div>
    <div v-else class="post-list">
      <PostCard v-for="post in posts" :post="post" />
    </div>
    <nav v-if="pagination.PrevURL !== '' || pagination.NextURL !== ''" class="pagination">
      <a v-if="pagination.PrevURL !== ''" :href="pagination.PrevURL" class="page-link">&larr; Previous</a>
      <span v-else class="page-link page-disabled">&larr; Previous</span>
      <a v-if="pagination.NextURL !== ''" :href="pagination.NextURL" class="page-link">Next &rarr;</a>
      <span v-else class="page-link page-disabled">Next &rarr;</span>
    </nav>
  </Layout>
</template>

<style>
.page-title {
  font-family: Georgia, serif;
  font-size: 1.8rem;
  font-weight: 700;
  margin-bottom: 2rem;
  border-bottom: 3px solid #1a1a1a;
  padding-bottom: 0.5rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.empty {
  color: #888;
  font-style: italic;
}

.pagination {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-top: 2.5rem;
  padding-top: 1.5rem;
  border-top: 3px solid #1a1a1a;
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.8rem;
}

.page-link {
  color: #1a1a1a;
  text-decoration: none;
  padding: 0.35rem 0.8rem;
  border: 2px solid #1a1a1a;
  font-weight: 700;
  box-shadow: 3px 3px 0 #1a1a1a;
  transition: transform 0.1s ease, box-shadow 0.1s ease, background 0.1s ease;
}

.page-link:hover {
  text-decoration: none;
  background: #ffd43b;
  transform: translate(-2px, -2px);
  box-shadow: 5px 5px 0 #1a1a1a;
}

.page-disabled {
  color: #bbb;
  border-color: #ccc;
  box-shadow: 3px 3px 0 #ccc;
  cursor: default;
}

.page-disabled:hover {
  background: none;
  color: #bbb;
  transform: none;
  box-shadow: 3px 3px 0 #ccc;
}
</style>
