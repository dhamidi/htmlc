<template>
  <Layout :siteTitle="siteTitle" pageTitle="Page Not Found" ogType="website" description="">
    <div class="not-found">
      <p class="not-found-code">404</p>
      <h1 class="not-found-title">Page not found</h1>
      <p class="not-found-body">The page you're looking for doesn't exist or has been moved.</p>
      <a href="/" class="not-found-link">&larr; Back to home</a>
    </div>
  </Layout>
</template>

<style scoped>
.not-found {
  text-align: center;
  padding: 4rem 0;
}

.not-found-code {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 5rem;
  font-weight: bold;
  color: #e0ddd8;
  line-height: 1;
  margin-bottom: 1rem;
}

.not-found-title {
  font-family: Georgia, serif;
  font-size: 2rem;
  margin-bottom: 1rem;
}

.not-found-body {
  color: #888;
  margin-bottom: 2rem;
}

.not-found-link {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.85rem;
  color: #b5451b;
}
</style>
