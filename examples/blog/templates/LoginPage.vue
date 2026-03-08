<template>
  <html>
    <head>
      <meta charset="utf-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
      <title>Admin Login – {{ siteTitle }}</title>
    </head>
    <body>
      <div class="login-wrap">
        <div class="login-box">
          <h1 class="login-title">{{ siteTitle }}</h1>
          <p class="login-sub">Admin Login</p>
          <div v-if="error" class="login-error">{{ error }}</div>
          <form method="POST" action="/admin/login" class="login-form">
            <label class="field">
              <span class="field-label">Username</span>
              <input type="text" name="username" autocomplete="username" required class="field-input" />
            </label>
            <label class="field">
              <span class="field-label">Password</span>
              <input type="password" name="password" autocomplete="current-password" required class="field-input" />
            </label>
            <button type="submit" class="login-btn">Log in</button>
          </form>
          <p class="login-back"><a href="/">← Back to blog</a></p>
        </div>
      </div>
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
  background: #1a1a1a;
  color: #f8f7f4;
  font-family: "SF Mono", "Fira Code", monospace;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
}

a {
  color: #ffd43b;
}

.login-wrap {
  width: 100%;
  max-width: 380px;
  padding: 1rem;
}

.login-box {
  background: #252525;
  border: 1px solid #333;
  border-radius: 4px;
  padding: 2.5rem 2rem;
}

.login-title {
  font-family: Georgia, serif;
  font-size: 1.4rem;
  color: #ffd43b;
  margin-bottom: 0.25rem;
}

.login-sub {
  font-size: 0.8rem;
  color: #888;
  margin-bottom: 1.75rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.login-error {
  background: #5c1a1a;
  border: 1px solid #8b2020;
  color: #ffaaaa;
  font-size: 0.85rem;
  padding: 0.6rem 0.9rem;
  border-radius: 3px;
  margin-bottom: 1.25rem;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.field-label {
  font-size: 0.75rem;
  color: #888;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.field-input {
  background: #1a1a1a;
  border: 1px solid #444;
  border-radius: 3px;
  color: #f8f7f4;
  font-family: inherit;
  font-size: 0.95rem;
  padding: 0.5rem 0.75rem;
  width: 100%;
  outline: none;
}

.field-input:focus {
  border-color: #ffd43b;
}

.login-btn {
  background: #ffd43b;
  border: none;
  border-radius: 3px;
  color: #1a1a1a;
  cursor: pointer;
  font-family: inherit;
  font-size: 0.9rem;
  font-weight: bold;
  margin-top: 0.5rem;
  padding: 0.6rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.login-btn:hover {
  background: #ffe566;
}

.login-back {
  font-size: 0.8rem;
  color: #666;
  margin-top: 1.5rem;
  text-align: center;
}
</style>
