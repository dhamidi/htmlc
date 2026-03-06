<template>
  <html>
    <head>
      <meta charset="utf-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
    </head>
    <body>
      <header>
        <slot name="header">Untitled</slot>
      </header>
      <main>
        <slot></slot>
      </main>
      <footer>
        <slot name="footer">© 2024 My Blog</slot>
      </footer>
    </body>
  </html>
</template>
