<template>
  <section class="hero">
    <div class="hero-badge">Go &bull; Zero JS &bull; Vue SFC syntax</div>
    <div class="hero-logo">
      <LogoIcon :size="96" style="border-radius:16px;"></LogoIcon>
    </div>
    <h1 class="hero-title">
      Server-side component<br />rendering for Go
    </h1>
    <p class="hero-sub">
      Write reusable HTML components in <code>.vue</code> files.
      Render them server-side in Go — no Node.js, no JavaScript runtime,
      no virtual DOM.
    </p>
    <div class="hero-actions">
      <ButtonLink href="/docs/tutorial.html" variant="primary">Get started</ButtonLink>
      <ButtonLink href="/docs/index.html" variant="secondary">Documentation</ButtonLink>
    </div>
    <div class="hero-install">
      <InstallCommand></InstallCommand>
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
  background: var(--accent-alpha12);
  border: 1px solid var(--accent-alpha35);
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
  color: var(--text);
  letter-spacing: -0.04em;
}

.hero-sub {
  font-size: 1.1rem;
  color: var(--muted);
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

.hero-logo {
  margin-bottom: 1.75rem;
}

/* Prevent the global :not(pre) > code border from showing inside the install block */
.hero-install :not(pre) > code {
  border: none;
  background: none;
  padding: 0;
  border-radius: 0;
}
</style>
