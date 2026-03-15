# RFC Authoring Guide

This guide codifies how to write a high-quality RFC for this project, derived from the patterns established in [RFC 001: Component Namespaces](001-component-namespaces.md).

---

## RFC Structure

Every RFC must contain the following sections, in order:

```
# RFC NNN: <Title>

- **Status**: Draft | Accepted | Rejected | Superseded
- **Date**: YYYY-MM-DD
- **Author**: <name or TBD>

---

## 1. Motivation
## 2. Goals
## 3. Non-Goals
## 4. Proposed Design
## 5. Syntax Summary        (if user-visible syntax is introduced)
## 6. Examples
## 7. Implementation Sketch
## 8. Backward Compatibility
## 9. Alternatives Considered
## 10. Open Questions
```

Sections may be omitted only when genuinely not applicable (e.g., no new syntax → omit §5). Do not reorder sections.

---

## Section-by-Section Guidance

### §1 Motivation

Open with a concrete statement of the current problem in one or two sentences. Then:

- Show the failure mode with a real directory or code example.
- Explain why the failure is **silent or hard to detect** — this is what makes it worth fixing.
- Address the most obvious alternative fix and explain why it does not apply to this project's constraints.

The goal is to make a reader who has never seen the codebase understand exactly what goes wrong and why it matters.

### §2 Goals

Numbered list. Each goal should be:
- **Actionable**: something that can be evaluated against the implementation.
- **Concise**: one line per goal.
- **Specific**: avoid vague goals like "improve performance". Prefer "proximity-based resolution: the nearest component wins".

Aim for 3–6 goals.

### §3 Non-Goals

Numbered list of things this RFC explicitly does **not** do. Each entry should:
- Name the thing that is out of scope.
- Give a brief reason why it is deferred or excluded.

Non-goals prevent scope creep during review and implementation.

### §4 Proposed Design

This is the main technical section. Use numbered subsections (4.1, 4.2, …).

Each subsection should follow this pattern:

1. **Current state** — what the code does today (quote field names or function signatures from the actual source).
2. **Proposed extension** — the delta. Use Go pseudocode or data-structure sketches to make the change concrete.
3. **Evaluation** — for design choices with multiple options, enumerate options using ✅/⚠️/❌ bullets and state a verdict.

Keep pseudocode clearly labelled `// pseudo-code — not implementation` to avoid confusion with actual implementation.

### §5 Syntax Summary

A Markdown table with two columns: **Syntax** and **Meaning**. Cover every new or changed surface in one place. This section is the reference a template author will consult after the RFC is accepted.

### §6 Examples

Provide 3–5 numbered examples. Each example should:
- Include a directory tree (use plain `tree`-style ASCII art).
- Include a resolution table (Template / Tag / Resolves to) or an HTML snippet, depending on which is clearer.
- Cover edge cases: flat projects (backward compatibility), cross-namespace references, walk-up fallback, dynamic references.

### §7 Implementation Sketch

High-level Go-level changes only. Do **not** write full implementations. Group changes by file (`engine.go`, `renderer.go`, etc.). Use brief numbered lists. Explicitly note when a change is a one-liner vs. a new method.

End with any notes about platform or OS considerations (e.g., `path` vs `filepath` for `fs.FS`).

### §8 Backward Compatibility

For each public API surface, state explicitly whether it changes. Use subheadings per surface. The default assumption should always be: **no breaking changes**. If a break is unavoidable, explain the migration path.

### §9 Alternatives Considered

List every alternative that was seriously considered during design. For each:
- State what it proposes.
- Give the reason it was rejected.

Do not include alternatives that were dismissed immediately — only ones that required real thought.

### §10 Open Questions

Numbered list of decisions that are deferred or unresolved. For each:
- State the question clearly.
- If there is a tentative recommendation, state it and why.
- Label questions that **must** be resolved before implementation (`blocking`) vs. those that can be addressed during or after (`non-blocking`).

---

## Style Rules

- **Headers**: use `##` for top-level sections, `###` for subsections, `####` for sub-subsections. Do not skip levels.
- **Code blocks**: always include a language identifier (`go`, `html`, `text`).
- **Tables**: use Markdown pipe tables. Align columns for readability.
- **Pseudocode**: label with `// pseudo-code — not implementation` comment inside the block.
- **Emphasis**: use `**bold**` for field names and new terms on first use; use `` `backticks` `` for code identifiers.
- **Verdict lines**: when evaluating options, use a bold **Verdict**: line to summarise the decision.
- **✅ / ⚠️ / ❌ bullets**: use for option evaluation grids. ✅ = advantage, ⚠️ = caveat, ❌ = disadvantage.
- **Horizontal rules** (`---`): separate the preamble from the body, and separate top-level sections when a visual break aids readability.
- **Tone**: technical and precise. Avoid marketing language. Avoid hedging ("might", "perhaps") unless expressing genuine uncertainty.
- **Length**: prefer thorough over brief. An RFC that a reviewer can evaluate without reading the source code is a good RFC.

---

## Numbering Convention

RFCs are numbered sequentially starting at `001`. File names follow the pattern:

```
NNN-short-slug.md
```

where `NNN` is zero-padded to three digits and `short-slug` is a lowercase hyphenated summary of the title.

---

## Pre-Review Checklist

Before submitting an RFC for review, verify:

- [ ] Status is set to `Draft`.
- [ ] Date is set to the authoring date (not a future target date).
- [ ] All sections §1–§10 are present or explicitly omitted with a note.
- [ ] Every new public API surface is listed in §8 with a compatibility statement.
- [ ] Every design option in §4 has a **Verdict**.
- [ ] §10 distinguishes blocking from non-blocking open questions.
- [ ] §6 includes a backward-compatibility example (flat project, no change in behaviour).
- [ ] Code blocks have language identifiers.
- [ ] Pseudocode is labelled as pseudocode.
- [ ] The file is named `NNN-short-slug.md` with the correct sequential number.

---

## Post-implementation documentation

Once an RFC is implemented, write a tutorial on the `docs` branch to show users how to use the feature. Use `docs/tutorial-template-integration.md` as the style reference.
