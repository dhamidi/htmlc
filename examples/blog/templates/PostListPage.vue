<template>
  <Layout>
    <template #header>My Blog</template>
    <PostList :posts="posts">
      <template #item="{ post, index }">
        <PostCard :id="post.ID" :title="post.Title" :body="post.Body" />
      </template>
    </PostList>
  </Layout>
</template>
