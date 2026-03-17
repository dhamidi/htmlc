import { readFileSync } from 'fs';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

const __dirname = dirname(fileURLToPath(import.meta.url));

// Extract the <script> block from HtmlcInspector.vue and execute it to register
// the custom element. The guard inside the IIFE (customElements.get check) means
// subsequent calls are no-ops, so we can safely call this once at module load.
function loadInspector() {
  const src = readFileSync(join(__dirname, 'HtmlcInspector.vue'), 'utf8');
  const m = src.match(/<script>([\s\S]*?)<\/script>/);
  if (!m) throw new Error('No <script> block found in HtmlcInspector.vue');
  // eslint-disable-next-line no-new-func
  new Function(m[1])();
}

loadInspector();

describe('HtmlcInspector', () => {
  let card;
  let inspector;

  beforeEach(() => {
    // Set up a component node so the tree is non-empty (step 2).
    card = document.createElement('div');
    card.dataset.htmlcComponent = 'Card';
    card.dataset.htmlcProps = '{}';
    document.body.appendChild(card);

    // Create and append the inspector (step 3).
    inspector = document.createElement('htmlc-inspector');
    document.body.appendChild(inspector);
  });

  afterEach(() => {
    document.body.innerHTML = '';
    vi.restoreAllMocks();
  });

  it('click on node-header does not leak to light DOM (step 6)', async () => {
    // Wait one microtask for connectedCallback to finish (step 3).
    await Promise.resolve();

    // Attach a click listener on the host element (step 4).
    let hostClickCount = 0;
    inspector.addEventListener('click', () => { hostClickCount++; });

    // Dispatch a composed click on the .node-header inside the shadow root (step 5).
    const header = inspector.shadowRoot.querySelector('.node-header');
    expect(header).not.toBeNull();
    header.dispatchEvent(new MouseEvent('click', { bubbles: true, composed: true }));

    // The click must not reach the host element (step 6).
    expect(hostClickCount).toBe(0);
  });

  it('_render() is called exactly once per click interaction (step 7)', async () => {
    // Spy on the prototype before any re-render so we capture calls triggered
    // by the click handler but not the initial connectedCallback render.
    const HtmlcInspectorClass = customElements.get('htmlc-inspector');
    const renderSpy = vi.spyOn(HtmlcInspectorClass.prototype, '_render');

    // Wait one microtask for connectedCallback (step 3).
    await Promise.resolve();

    // The connectedCallback has already called _render() once. Clear so we
    // only count renders triggered by the click.
    renderSpy.mockClear();

    // Dispatch a composed click on a .node-header (step 5).
    const header = inspector.shadowRoot.querySelector('.node-header');
    expect(header).not.toBeNull();
    header.dispatchEvent(new MouseEvent('click', { bubbles: true, composed: true }));

    // _render() must have been called exactly once — the selection toggle —
    // not continuously (step 7).
    expect(renderSpy).toHaveBeenCalledTimes(1);
  });

  it('_render() re-entrant guard prevents a second call from running', async () => {
    await Promise.resolve();

    const HtmlcInspectorClass = customElements.get('htmlc-inspector');

    // Count how many times the body of _render actually executes past the guard.
    let bodyEntryCount = 0;
    const original = HtmlcInspectorClass.prototype._render;
    vi.spyOn(HtmlcInspectorClass.prototype, '_render').mockImplementation(function () {
      // Simulate re-entrant call: call _render() again from inside itself.
      if (bodyEntryCount === 0) {
        bodyEntryCount++;
        // Try to call _render() again while _rendering is true.
        original.call(this); // first real call
        // After original call returns, _rendering is false again; calling it
        // once more should go through normally (testing guard during, not after).
      } else {
        original.call(this);
      }
    });

    // Use a simpler direct re-entrancy test: set _rendering manually and verify
    // _render() exits early.
    inspector._rendering = true;
    const shadowBefore = inspector.shadowRoot.innerHTML;
    inspector._render(); // should return immediately
    const shadowAfter = inspector.shadowRoot.innerHTML;

    // Shadow DOM must not have changed because _render returned early.
    expect(shadowAfter).toBe(shadowBefore);

    // Reset for cleanup.
    inspector._rendering = false;
  });
});
