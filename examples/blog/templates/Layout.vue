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
  background: #fff;
  color: #000;
  font-family: Georgia, "Times New Roman", serif;
  line-height: 1.6;
}

.site-header {
  border-bottom: 3px solid #000;
  padding: 1.5rem 2rem;
  font-family: "Arial Black", Arial, sans-serif;
  font-weight: 900;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.site-header-home {
  color: inherit;
  text-decoration: none;
}

.site-header-home:hover {
  text-decoration: underline;
}

.site-main {
  max-width: 800px;
  margin: 0 auto;
  padding: 3rem 2rem;
}

.site-footer {
  border-top: 3px solid #000;
  padding: 1.5rem 2rem;
  font-size: 0.875rem;
}
</style>
