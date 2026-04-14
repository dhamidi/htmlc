<template>
  <Layout :siteTitle="siteTitle" pageTitle="About" ogType="website" description="">
    <h1 class="page-title">About</h1>
    <div class="about-content" v-html="content"></div>
  </Layout>
</template>

<style>
.page-title {
  font-family: Georgia, serif;
  font-size: 1.8rem;
  font-weight: 700;
  margin-bottom: 2rem;
  border-bottom: 3px solid #1a1a1a;
  padding-bottom: 0.5rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.about-content {
  font-size: 1.05rem;
  line-height: 1.8;
}

.about-content p {
  margin-bottom: 1.25rem;
}

.about-content h2,
.about-content h3 {
  font-family: Georgia, serif;
  font-weight: 700;
  margin: 2rem 0 0.75rem;
  border-bottom: 2px solid #1a1a1a;
  padding-bottom: 0.25rem;
}
</style>
