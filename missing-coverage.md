# Directive Coverage Notes

Date: 2026-03-09
Go: go1.26.1 linux/arm64
Commit: e464d3f5607562fa88baeb5bd2a946fb0c7e785d

## Coverage Table

| Directive              | Test exists | Pass |
|------------------------|-------------|------|
| `v-text`               | YES — TestRender_VText, TestRender_VTextHTMLEscaped, TestRender_VTextReplacesChildren | PASS |
| `v-html`               | YES — TestRender_VHtml, TestRender_VHtmlNotEscaped, TestRender_VHtmlReplacesChildren | PASS |
| `v-show`               | YES — TestRender_VShowFalse, TestRender_VShowTrue, TestRender_VShowMergesExistingStyle, TestRender_VShowScopeExpression | PASS |
| `v-if`                 | YES — TestRender_VIfTrue, TestRender_VIfFalse, TestRender_VIfElseChain, TestRender_VElseRendersWhenAllFalsy, TestRender_VIfTemplateWrapper, TestRender_VIfScopeExpression, TestRender_VIfOnlyFirstTruthyBranchRenders | PASS |
| `v-else-if`            | YES — TestRender_VIfElseChain, TestRender_VElseRendersWhenAllFalsy | PASS |
| `v-else`               | YES — TestRender_VIfElseChain, TestRender_VElseRendersWhenAllFalsy | PASS |
| `v-for` (array)        | YES — TestRender_VForSimpleArray, TestRender_VForWithIndex, TestRender_VForEmptyArray, TestRender_VForTemplateWrapper | PASS |
| `v-for` (map)          | YES — TestRender_VForObject | PASS |
| `v-for` (range)        | YES — TestRender_VForInteger | PASS |
| `v-bind :class` (obj)  | YES — TestRender_VBindClassObjectTruthy, TestRender_VBindClassObjectScope, TestRender_VBindStaticAndDynamicClassMerge | PASS |
| `v-bind :class` (arr)  | YES — TestRender_VBindClassArrayTrue, TestRender_VBindClassArrayFalse | PASS |
| `v-bind :style`        | YES — TestRender_VBindStyleObject | PASS |
| `v-bind` boolean attrs | YES — TestRender_VBindDisabledFalse, TestRender_VBindDisabledTrue, TestRender_VBindChecked | PASS |
| `v-pre`                | YES — TestRender_VPreLiteral, TestRender_VPreStripsDirective, TestRender_VPreSkipsDescendants | PASS |
| `v-slot` default       | YES — TestRender_ComponentLayoutSlot, TestRender_ComponentSlot, TestRender_SlotFallbackWhenMissing | PASS |
| `v-slot` named         | YES — TestRender_NamedSlots, TestRender_NamedSlotsHashSyntax, TestRender_NamedSlotOverridesFallback, TestRender_NamedScopedSlot | PASS |
| `v-slot` scoped        | YES — TestRender_ScopedSlotDestructured, TestRender_ScopedSlotSingleVar, TestRender_ScopedSlotInsideVFor, TestRender_ScopedSlotParentScopeAccessible, TestRender_ScopedSlotPropOverridesParentVar | PASS |

## Missing Coverage

**No directives are missing test coverage.**

All 17 directive variants in the support matrix have at least one passing test.
