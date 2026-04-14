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
  padding: 4rem 1rem;
}

.not-found-code {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 8rem;
  font-weight: 700;
  color: #ffd43b;
  line-height: 1;
  margin-bottom: 0.5rem;
  text-shadow: 6px 6px 0 #1a1a1a;
  display: block;
}

.not-found-title {
  font-family: Georgia, serif;
  font-size: 2.2rem;
  font-weight: 700;
  margin-bottom: 1rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border: 3px solid #1a1a1a;
  display: inline-block;
  padding: 0.4rem 1.2rem;
  box-shadow: 5px 5px 0 #1a1a1a;
  background: #fff;
}

.not-found-body {
  color: #555;
  margin: 1.5rem auto 2rem;
  font-size: 1.1rem;
  max-width: 400px;
}

.not-found-link {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.85rem;
  font-weight: 700;
  color: #1a1a1a;
  text-decoration: none;
  border: 2px solid #1a1a1a;
  padding: 0.4rem 1rem;
  box-shadow: 4px 4px 0 #1a1a1a;
  display: inline-block;
  transition: transform 0.1s ease, box-shadow 0.1s ease, background 0.1s ease;
}

.not-found-link:hover {
  text-decoration: none;
  background: #ffd43b;
  transform: translate(-2px, -2px);
  box-shadow: 6px 6px 0 #1a1a1a;
}
</style>
