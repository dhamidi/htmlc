<template>
  <Layout>
    <template #header>{{ pageTitle }}</template>
    <PostForm :action="action" :post="post" :submitLabel="submitLabel" />
  </Layout>
</template>
