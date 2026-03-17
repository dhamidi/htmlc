# CLAUDE.md

## Branch Policy

- **`main`** — source code and code-adjacent documentation (README.md, go doc comments). All code changes and related documentation updates go here.
- **`docs`** — documentation website content, managed separately. Read access is allowed when working on `main`, but do not commit documentation website changes to `main`.
- only on the branch you are working on

## Test conventions

Prefer `testing/fstest.MapFS` over `os.WriteFile`/`t.TempDir()` for test input fixtures.

**Accepted exceptions** — tests that must use the real OS filesystem:
- `cmd/htmlc/build_command_test.go: TestRunBuild_Dev_RebuildsOnChange` — exercises the full dev-server rebuild loop end-to-end, including real mtime-based `dirHash` and `os.Chtimes`; an `fs.FS` abstraction cannot substitute here.
- `cmd/htmlc/external_directive_test.go: TestBuildExternalDirective` — launches real OS processes (external directive scripts); requires real filesystem paths.

## How to write a proposal

See [./docs/proposals/CLAUDE.md]

Proposals belong on the main branch.

## Documenting features

- Feature documentation (tutorials, how-to guides, explanations) lives on the `docs` branch, not `main`.
- For each new user-facing feature, write a **tutorial**: a short, worked example that shows a reader how to use the feature end-to-end.
- Keep tutorials focused and concise — one feature, one goal, enough code to run.
- Use `docs/tutorial-template-integration.md` as the canonical style reference.

## README.md conventions

- Do **not** use numbered prefixes on `##` section headings (e.g., `## 1. Overview` is wrong; `## Overview` is correct).
- Numbered prefixes break GitHub anchor links: `## 13. Testing` generates the anchor `#13-testing`, but the Table of Contents links to `#testing`.
- The Table of Contents ordered list (`1.`, `2.`, …) provides visual numbering — headings themselves must remain unnumbered.
