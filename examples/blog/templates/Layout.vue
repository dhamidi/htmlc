<template>
  <html>
    <head>
      <meta charset="utf-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
      <title>My Blog</title>
    </head>
    <body>
      <header class="site-header">
        <a href="/" class="site-header-home"><slot name="header">Untitled</slot></a>
        <nav class="site-nav">
          <a href="/posts/new" class="site-nav-link">+ New Post</a>
        </nav>
      </header>
      <main class="site-main">
        <slot>No content.</slot>
      </main>
      <footer class="site-footer">
        <slot name="footer">© 2024 My Blog</slot>
      </footer>
    </body>
  </html>
</template>

<style>
*, *::before, *::after {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  background: #f8f7f4;
  color: #1a1a1a;
  font-family: "SF Mono", "Fira Code", "Fira Mono", "Roboto Mono", "Courier New", monospace;
  line-height: 1.6;
  min-height: 100vh;
}

a {
  transition: all 0.15s ease;
}

.site-header {
  background: #1a1a1a;
  color: #fff;
  padding: 1.25rem 2rem;
  font-family: inherit;
  font-weight: 900;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.site-header-home {
  color: #fff;
  text-decoration: none;
  font-size: 1.1rem;
}

.site-header-home:hover {
  color: #ffd43b;
}

.site-nav-link {
  color: #1a1a1a;
  background: #ffd43b;
  text-decoration: none;
  font-size: 0.8rem;
  font-weight: 900;
  padding: 0.4rem 0.9rem;
  letter-spacing: 0.04em;
}

.site-nav-link:hover {
  background: #fff;
}

.site-main {
  max-width: 800px;
  margin: 0 auto;
  padding: 2.5rem 2rem 4rem;
}

.site-footer {
  background: #1a1a1a;
  color: #999;
  padding: 1.25rem 2rem;
  font-size: 0.8rem;
  letter-spacing: 0.03em;
}
</style>
