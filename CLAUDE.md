# htmlc Docs Site — Claude Code Guide

## Project Structure

```
/workspace
├── components/       # Vue SFC components (.vue)
├── pages/            # Site pages (.vue and subdirectories)
│   └── docs/         # Documentation pages
├── public/           # Static assets
├── CLAUDE.md         # This file
└── AGENTS.md         # Agent pipeline instructions
```

## Build Instructions

Run the build before committing:

```bash
mise run build
```

A successful build outputs: `Build complete: N pages, 0 errors.`

## Validation Rules

### No Self-Closing Custom Component Tags

`htmlc build -strict` rejects self-closing custom component tags. Always use explicit open/close form.

**WRONG:**

```html
<InstallCommand />
<MyComponent />
<Buttonlink />
```

**CORRECT:**

```html
<InstallCommand></InstallCommand>
<MyComponent></MyComponent>
<ButtonLink></ButtonLink>
```

Note: Standard HTML void elements (`<br />`, `<img />`, `<input />`) are fine.

## Workflow

1. Make changes to `.vue` files in `components/` or `pages/`
2. Run `mise run build` to validate
3. Fix any errors reported (0 errors required before committing)
4. Commit changes
