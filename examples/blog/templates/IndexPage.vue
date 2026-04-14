<template>
  <Layout :siteTitle="siteTitle" pageTitle="Posts" description="Latest posts" ogType="website">
    <h1 class="page-title">Posts</h1>
    <div v-if="posts.length === 0" class="empty">
      <p>No posts yet.</p>
    </div>
    <div v-else class="post-list">
      <PostCard v-for="post in posts" :post="post" />
    </div>
    <Pagination :pagination="pagination" />
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

.post-list {
  display: flex;
  flex-direction: column;
  gap: 0;
}
</style>
