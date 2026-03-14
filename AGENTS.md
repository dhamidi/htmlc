# Agent Instructions — htmlc Docs

## Branch Context

This is the `docs` branch of [github.com/dhamidi/htmlc](https://github.com/dhamidi/htmlc).

It is automatically deployed to Cloudflare Pages at https://htmlc.sh

When documenting htmlc, always refer to the latest commit of the `main` branch.
You are free to add a worktree of the main branch into `/tmp` to inspect the code.

## Build & Validation

Always run the build before committing:

```bash
mise run build
```

Expected output: `Build complete: N pages, 0 errors.`

### Common Error: Self-Closing Custom Component Tags

`htmlc build -strict` rejects self-closing custom component tags.

**WRONG:**
```html
<InstallCommand />
<MyComponent />
```

**CORRECT:**
```html
<InstallCommand></InstallCommand>
<MyComponent></MyComponent>
```

Standard HTML void elements (`<br />`, `<img />`, `<hr />`) are allowed as self-closing.

## Commit Requirements

- Run `mise run build` and confirm 0 errors before committing
- Create exactly one commit per work item
- Do not push to remote
