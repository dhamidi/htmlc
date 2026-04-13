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
  font-size: 1.6rem;
  margin-bottom: 2rem;
  border-bottom: 2px solid #1a1a1a;
  padding-bottom: 0.5rem;
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
  border-top: 1px solid #e0ddd8;
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.8rem;
}

.page-link {
  color: #b5451b;
  text-decoration: none;
  padding: 0.3rem 0.6rem;
  border: 1px solid currentColor;
  border-radius: 3px;
}

.page-disabled {
  color: #ccc;
  border-color: #e0ddd8;
}
</style>
