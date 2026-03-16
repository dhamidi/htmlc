# CLAUDE.md

## Branch Policy

- **`main`** — source code and code-adjacent documentation (README.md, go doc comments). All code changes and related documentation updates go here.
- **`docs`** — documentation website content, managed separately. Read access is allowed when working on `main`, but do not commit documentation website changes to `main`.
- only on the branch you are working on

## How to write a proposal

See [./docs/proposals/CLAUDE.md]

Proposals belong on the main branch.

## Documenting features

- Feature documentation (tutorials, how-to guides, explanations) lives on the `docs` branch, not `main`.
- For each new user-facing feature, write a **tutorial**: a short, worked example that shows a reader how to use the feature end-to-end.
- Keep tutorials focused and concise — one feature, one goal, enough code to run.
- Use `docs/tutorial-template-integration.md` as the canonical style reference.
