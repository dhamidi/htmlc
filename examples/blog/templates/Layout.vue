<template>
  <html>
    <head>
      <meta charset="utf-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
      <title>{{ siteTitle }}</title>
    </head>
    <body>
      <header class="site-header">
        <a href="/" class="site-logo">{{ siteTitle }}</a>
        <nav class="site-nav">
          <a href="/feed.atom" class="nav-link">Feed</a>
        </nav>
      </header>
      <main class="site-main">
        <slot>No content.</slot>
      </main>
      <footer class="site-footer">
        &copy; {{ siteTitle }}
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
  font-family: Georgia, "Times New Roman", serif;
  line-height: 1.7;
  min-height: 100vh;
}

a {
  color: #b5451b;
  text-decoration: none;
}

a:hover {
  text-decoration: underline;
}

.site-header {
  background: #1a1a1a;
  color: #f8f7f4;
  padding: 1rem 2rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.site-logo {
  color: #f8f7f4;
  font-size: 1.2rem;
  font-weight: bold;
  text-decoration: none;
  font-family: Georgia, serif;
}

.site-logo:hover {
  color: #ffd43b;
  text-decoration: none;
}

.site-nav {
  display: flex;
  gap: 1.5rem;
}

.nav-link {
  color: #aaa;
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.8rem;
  text-decoration: none;
}

.nav-link:hover {
  color: #ffd43b;
  text-decoration: none;
}

.site-main {
  max-width: 760px;
  margin: 0 auto;
  padding: 3rem 2rem 5rem;
}

.site-footer {
  background: #1a1a1a;
  color: #666;
  padding: 1rem 2rem;
  font-size: 0.8rem;
  font-family: "SF Mono", "Fira Code", monospace;
  text-align: center;
}
</style>
