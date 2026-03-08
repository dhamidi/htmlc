<template>
  <html>
    <head>
      <meta charset="utf-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
      <title>Admin – {{ siteTitle }}</title>
    </head>
    <body>
      <header class="admin-header">
        <a href="/admin/" class="admin-logo">{{ siteTitle }} &rsaquo; Admin</a>
        <nav class="admin-nav">
          <a href="/admin/" class="admin-nav-link">Dashboard</a>
          <a href="/admin/drafts" class="admin-nav-link">Drafts</a>
          <a href="/admin/posts/new" class="admin-nav-link">+ New Post</a>
          <form method="POST" action="/admin/logout" class="logout-form">
            <button type="submit" class="logout-btn">Log out</button>
          </form>
        </nav>
      </header>
      <main class="admin-main">
        <slot>No content.</slot>
      </main>
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
  background: #f0eeea;
  color: #1a1a1a;
  font-family: "SF Mono", "Fira Code", "Fira Mono", monospace;
  line-height: 1.6;
  min-height: 100vh;
}

a {
  color: #b5451b;
  text-decoration: none;
}

a:hover {
  text-decoration: underline;
}

.admin-header {
  background: #1a1a1a;
  color: #f8f7f4;
  padding: 0.75rem 2rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.admin-logo {
  color: #ffd43b;
  font-weight: bold;
  font-size: 0.9rem;
  text-decoration: none;
}

.admin-logo:hover {
  text-decoration: none;
  color: #fff;
}

.admin-nav {
  display: flex;
  align-items: center;
  gap: 1.5rem;
}

.admin-nav-link {
  color: #aaa;
  font-size: 0.8rem;
  text-decoration: none;
}

.admin-nav-link:hover {
  color: #fff;
  text-decoration: none;
}

.logout-form {
  margin: 0;
}

.logout-btn {
  background: none;
  border: 1px solid #444;
  color: #aaa;
  cursor: pointer;
  font-family: inherit;
  font-size: 0.8rem;
  padding: 0.2rem 0.6rem;
  border-radius: 2px;
}

.logout-btn:hover {
  border-color: #aaa;
  color: #fff;
}

.admin-main {
  max-width: 900px;
  margin: 0 auto;
  padding: 2.5rem 2rem 5rem;
}
</style>
