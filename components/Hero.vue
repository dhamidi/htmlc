<template>
  <section class="hero">
    <div class="hero-logo">
      <img src="/images/logo.webp" alt="htmlc" width="96" height="96"
           style="border-radius:16px;" />
    </div>
    <div class="hero-badge">Go &bull; Zero JS &bull; Vue SFC syntax</div>
    <h1 class="hero-title">
      Server-side component<br />rendering for Go
    </h1>
    <p class="hero-sub">
      Write reusable HTML components in <code>.vue</code> files.
      Render them server-side in Go — no Node.js, no JavaScript runtime,
      no virtual DOM.
    </p>
    <div class="hero-actions">
      <a href="/docs/index.html" class="btn btn-primary">Get started</a>
      <a href="/docs/cli.html" class="btn btn-secondary">CLI reference</a>
    </div>
    <div class="hero-install">
      <code>go install github.com/dhamidi/htmlc/cmd/htmlc@latest</code>
    </div>
  </section>
</template>

<style scoped>
.hero {
  padding: 6rem 0 4rem;
  text-align: center;
  max-width: 760px;
  margin: 0 auto;
}

.hero-badge {
  display: inline-block;
  background: rgba(0, 173, 216, 0.12);
  border: 1px solid rgba(0, 173, 216, 0.35);
  color: #7dd9ef;
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  padding: 0.3em 0.9em;
  border-radius: 100px;
  margin-bottom: 1.5rem;
}

.hero-title {
  font-size: clamp(2.2rem, 5vw, 3.6rem);
  font-weight: 800;
  line-height: 1.1;
  margin-bottom: 1.25rem;
  color: #f0f2ff;
  letter-spacing: -0.04em;
}

.hero-sub {
  font-size: 1.1rem;
  color: #8b8fa8;
  margin-bottom: 2.5rem;
  max-width: 560px;
  margin-left: auto;
  margin-right: auto;
}

.hero-actions {
  display: flex;
  gap: 1rem;
  justify-content: center;
  margin-bottom: 2rem;
  flex-wrap: wrap;
}

.btn {
  display: inline-flex;
  align-items: center;
  padding: 0.65rem 1.5rem;
  border-radius: 8px;
  font-weight: 600;
  font-size: 0.9rem;
  transition: all 0.15s;
  text-decoration: none;
}

.btn-primary {
  background: #00ADD8;
  color: #fff;
}
.btn-primary:hover { background: #0099bf; text-decoration: none; }

.btn-secondary {
  background: rgba(255,255,255,0.06);
  border: 1px solid rgba(255,255,255,0.12);
  color: #e2e4f0;
}
.btn-secondary:hover { background: rgba(255,255,255,0.1); text-decoration: none; }

.hero-logo {
  margin-bottom: 1.75rem;
}

.hero-install {
  display: inline-block;
  background: #161822;
  border: 1px solid #2a2d3e;
  border-radius: 8px;
  padding: 0.6rem 1.2rem;
  font-size: 0.85rem;
  color: #5be49b;
}
</style>
