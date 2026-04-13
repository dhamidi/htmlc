<template>
  <Layout :siteTitle="siteTitle">
    <div class="not-found">
      <h1 class="not-found-title">404</h1>
      <p class="not-found-message">Page not found.</p>
      <a href="/" class="not-found-link">&larr; Back to home</a>
    </div>
  </Layout>
</template>

<style>
.not-found {
  text-align: center;
  padding: 4rem 0;
}

.not-found-title {
  font-family: Georgia, serif;
  font-size: 5rem;
  color: #1a1a1a;
  line-height: 1;
  margin-bottom: 1rem;
}

.not-found-message {
  font-size: 1.2rem;
  color: #666;
  margin-bottom: 2rem;
}

.not-found-link {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.9rem;
  color: #b5451b;
}
</style>
