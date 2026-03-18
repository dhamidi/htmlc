// --- h() helper ---
function h(tag, attrs, ...children) {
  const el = document.createElement(tag);
  if (attrs) {
    for (const [k, v] of Object.entries(attrs)) {
      if (k === 'style' && typeof v === 'object') {
        Object.assign(el.style, v);
      } else if (k.startsWith('on') && typeof v === 'function') {
        el.addEventListener(k.slice(2), v);
      } else if (k === 'className') {
        el.className = v;
      } else {
        el.setAttribute(k, v);
      }
    }
  }
  for (const child of children.flat(Infinity)) {
    if (child == null || child === false) continue;
    el.append(typeof child === 'string' ? child : child);
  }
  return el;
}

function svg(html) {
  const t = document.createElement('template');
  t.innerHTML = html.trim();
  return t.content.firstChild;
}

// --- SVG assets ---
const LOGO_SVG = (w, h_) =>
  `<svg viewBox="0 0 512 512" width="${w}" height="${h_}" style="border-radius:${w > 24 ? 6 : 3}px;display:block;"><rect fill="#00ADD8" x="0" y="0" width="512" height="512"/><g transform="translate(468 456)" font-family="Inter, SF Pro Display, Segoe UI, Roboto, Helvetica, Arial, sans-serif" font-size="128" font-weight="800" text-anchor="end"><text x="0" y="0"><tspan fill="#F8FAFC">html</tspan><tspan fill="#1F2937">c</tspan></text></g></svg>`;

const CROSSHAIR_SVG =
  '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="22" y1="12" x2="18" y2="12"/><line x1="6" y1="12" x2="2" y2="12"/><line x1="12" y1="6" x2="12" y2="2"/><line x1="12" y1="22" x2="12" y2="18"/></svg>';

// --- Styles ---
const STYLES = `
:host {
  position: fixed; bottom: 16px; right: 16px; z-index: 2147483647;
  pointer-events: auto;
  font-family: "JetBrains Mono", ui-monospace, monospace; font-size: 12px;
}
.panel {
  background: #0f172a; border: 1px solid #334155; border-radius: 10px;
  box-shadow: 0 8px 32px rgba(0,0,0,0.5); width: 380px; max-height: 480px;
  display: flex; flex-direction: column; overflow: hidden;
}
.titlebar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 8px 12px; background: #1e293b; border-bottom: 1px solid #334155;
  user-select: none;
}
.titlebar-left {
  display: flex; align-items: center; gap: 8px; color: #00ADD8;
  font-weight: 600; font-size: 12px; cursor: pointer;
}
.titlebar-left svg { border-radius: 4px; flex-shrink: 0; }
.titlebar-right { display: flex; align-items: center; gap: 4px; }
.filter-bar {
  padding: 6px 10px; background: #1e293b; border-bottom: 1px solid #334155;
}
.filter-input {
  width: 100%; box-sizing: border-box; background: #0f172a; border: 1px solid #334155;
  border-radius: 4px; padding: 4px 8px; color: #e2e8f0; font-family: inherit;
  font-size: 11px; outline: none;
}
.filter-input::placeholder { color: #475569; }
.filter-input:focus { border-color: #00ADD8; }
.node-filtered {
  margin-left: 14px; border-left: 1px solid #334155; padding-left: 8px;
}
.icon-btn {
  display: flex; align-items: center; justify-content: center;
  width: 24px; height: 24px; background: none; border: 1px solid transparent;
  border-radius: 4px; cursor: pointer; color: #94a3b8; padding: 0;
  font-family: inherit; transition: all 0.15s;
}
.icon-btn:hover { color: #fff; background: rgba(255,255,255,0.06); }
.icon-btn.active { color: #00ADD8; border-color: #00ADD8; background: rgba(0, 173, 216, 0.12); }
.icon-btn.close { font-size: 16px; }
.tree {
  overflow-y: auto; padding: 8px; flex: 1;
}
.tree::-webkit-scrollbar { width: 6px; }
.tree::-webkit-scrollbar-track { background: transparent; }
.tree::-webkit-scrollbar-thumb { background: #334155; border-radius: 3px; }
.node-header {
  display: flex; align-items: center; gap: 8px; padding: 4px 6px;
  border-radius: 4px; cursor: pointer; color: #94a3b8; line-height: 1.4;
  outline: none;
}
.node-header:hover { background: #1e293b; }
.node-header.focused { outline: 1px solid #00ADD8; outline-offset: -1px; }
.node-header.selected { background: rgba(0, 173, 216, 0.12); }
.tag { color: #00ADD8; font-weight: 500; white-space: nowrap; }
.file {
  color: #475569; font-size: 10px; flex: 1; overflow: hidden;
  text-overflow: ellipsis; white-space: nowrap;
}
.prop-count {
  color: #64748b; font-size: 10px; background: #1e293b;
  padding: 1px 5px; border-radius: 3px; white-space: nowrap;
}
.children {
  margin-left: 14px; border-left: 1px solid #334155; padding-left: 8px;
}
.props-panel {
  margin: 2px 0 6px 6px; padding: 8px 10px; background: #1e293b;
  border-radius: 6px; border: 1px solid #334155;
}
.props-title {
  color: #64748b; font-size: 10px; text-transform: uppercase;
  letter-spacing: 0.05em; margin-bottom: 6px;
}
.props-table {
  width: 100%; border-collapse: separate; border-spacing: 16px 0;
  margin-left: -16px;
}
.props-table td {
  padding: 5px 0; vertical-align: top; border-bottom: 1px solid #2a2d3e;
}
.props-table tr:last-child td { border-bottom: none; }
.prop-key {
  color: #7dd9ef; padding-right: 16px; white-space: nowrap;
  width: 1%; font-size: 11px;
}
.prop-val { color: #94a3b8; word-break: break-all; font-size: 11px; }
.logo-btn {
  background: none; border: none; cursor: pointer; padding: 0;
  border-radius: 6px; box-shadow: 0 4px 16px rgba(0,0,0,0.4);
  transition: transform 0.15s; display: block;
}
.logo-btn:hover { transform: scale(1.08); }
`;

// --- State Machine ---
//
// States: Collapsed, Browsing, Navigating, Picking
//
// Transitions:
//   Collapsed  --toggle/clickLogo--> Browsing
//   Browsing   --toggle/close------> Collapsed
//   Browsing   --submitFilter------> Navigating  (Enter in filter, selects first match)
//   Browsing   --clickNode---------> Navigating
//   Browsing   --startPicking------> Picking
//   Navigating --toggle/close------> Collapsed
//   Navigating --escape------------> Browsing    (back to filter)
//   Navigating --inputFilter-------> Browsing    (typing in filter clears nav focus)
//   Navigating --startPicking------> Picking
//   Picking    --escape------------> Browsing
//   Picking    --pickElement-------> Navigating  (element selected)

class Collapsed {
  get name() { return 'collapsed'; }
  get filter() { return ''; }
  get selectedEl() { return null; }
  get focusedId() { return null; }
  get picking() { return false; }
  get focusTarget() { return null; }

  toggle() { return new Browsing(); }
  clickLogo() { return new Browsing(); }
}

class Browsing {
  constructor(filter = '', selectedEl = null) {
    this.filter = filter;
    this.selectedEl = selectedEl;
  }

  get name() { return 'browsing'; }
  get focusedId() { return null; }
  get picking() { return false; }
  get focusTarget() { return 'filter'; }

  toggle() { return new Collapsed(); }
  close() { return new Collapsed(); }

  inputFilter(value) {
    return new Browsing(value, this.selectedEl);
  }

  submitFilter(firstId, elMap) {
    if (!firstId) return this;
    return new Navigating(this.filter, elMap[firstId], firstId);
  }

  clickNode(id, el) {
    const selected = this.selectedEl === el ? null : el;
    return new Navigating(this.filter, selected, id);
  }

  startPicking() {
    return new Picking(this.filter);
  }
}

class Navigating {
  constructor(filter = '', selectedEl = null, focusedId = null) {
    this.filter = filter;
    this.selectedEl = selectedEl;
    this.focusedId = focusedId;
  }

  get name() { return 'navigating'; }
  get picking() { return false; }
  get focusTarget() { return 'tree'; }

  toggle() { return new Collapsed(); }
  close() { return new Collapsed(); }

  escape() {
    return new Browsing(this.filter);
  }

  inputFilter(value) {
    return new Browsing(value);
  }

  moveFocus(id) {
    return new Navigating(this.filter, this.selectedEl, id);
  }

  selectFocused(elMap) {
    if (!this.focusedId) return this;
    const el = elMap[this.focusedId];
    const selected = this.selectedEl === el ? null : el;
    return new Navigating(this.filter, selected, this.focusedId);
  }

  clickNode(id, el) {
    const selected = this.selectedEl === el ? null : el;
    return new Navigating(this.filter, selected, id);
  }

  startPicking() {
    return new Picking(this.filter);
  }
}

class Picking {
  constructor(filter = '') {
    this.filter = filter;
  }

  get name() { return 'picking'; }
  get selectedEl() { return null; }
  get focusedId() { return null; }
  get picking() { return true; }
  get focusTarget() { return null; }

  escape() {
    return new Browsing(this.filter);
  }

  pickElement(id, el) {
    return new Navigating(this.filter, el, id);
  }

  stopPicking() {
    return new Browsing(this.filter);
  }
}

// --- Component ---
class HtmlcInspector extends HTMLElement {
  connectedCallback() {
    if (this._initialized) return;
    this._initialized = true;

    // Create a manual popover portal to enter the browser top layer so the
    // inspector paints above <dialog> elements opened with showModal().
    // The Popover API (popover="manual" + showPopover()) promotes the element
    // to the top layer without focus trapping or a backdrop.
    const portal = document.createElement('div');
    portal.setAttribute('popover', 'manual');
    portal.style.cssText = [
      'all: unset',
      'position: fixed',
      'inset: 0',
      'width: 0',
      'height: 0',
      'overflow: visible',
      'pointer-events: none',
    ].join(';');
    document.body.appendChild(portal);
    portal.showPopover(); // enters the top layer
    this._portal = portal;

    // Move this element into the portal so it inherits top-layer membership.
    // This triggers disconnectedCallback then connectedCallback again;
    // the _initialized guard above prevents re-entrant setup.
    this._movingToPortal = true;
    portal.appendChild(this);
    this._movingToPortal = false;

    this.attachShadow({ mode: 'open' });
    this._state = new Collapsed();
    this._elMap = {};
    this._nodeOrder = [];
    this._nodeParent = {};
    this._nodeChildren = {};
    this._render();
    this._setupOverlay();
    this._setupPicker();
    this._onGlobalKey = (e) => {
      if ((e.ctrlKey && e.altKey || e.ctrlKey && e.shiftKey) && (e.key === 'H' || e.key === 'h' || e.key === '\u02D9')) {
        e.preventDefault();
        this._transition(this._state.toggle());
      }
    };
    document.addEventListener('keydown', this._onGlobalKey, true);

    // showModal() makes all non-descendant elements inert at the browser level
    // (no z-index or top-layer trick can override this). The only way to remain
    // interactive while a modal dialog is open is to be a descendant of that
    // dialog. We watch for modal dialogs and reparent the inspector into them.
    this._activeModal = null;
    this._dialogObserver = new MutationObserver(() => {
      if (!this._portal) return;
      const modal = [...document.querySelectorAll('dialog')].find(d => d.open && d.matches(':modal'));
      const target = modal || this._portal;
      if (target === this.parentElement) return; // already in the right place
      this._activeModal = modal || null;
      this._movingToPortal = true;
      target.appendChild(this);
      this._movingToPortal = false;
    });
    this._dialogObserver.observe(document.documentElement, {
      subtree: true, attributes: true, attributeFilter: ['open'],
    });

    // Re-promote the portal to the top of the top layer whenever a page popover
    // opens. The top layer renders elements in promotion order; if a page popover
    // is promoted after our portal, it paints above us. By hiding and re-showing
    // the portal we move it to the end of the top layer stack.
    this._onToggle = (e) => {
      if (!this._portal) return;
      if (e.target === this._portal) return; // ignore our own portal toggle
      if (e.newState === 'open') {
        try {
          this._portal.hidePopover();
          this._portal.showPopover();
        } catch (_) {}
      }
    };
    document.addEventListener('toggle', this._onToggle, true);
  }

  disconnectedCallback() {
    if (this._movingToPortal) return;
    if (this._dialogObserver) { this._dialogObserver.disconnect(); this._dialogObserver = null; }
    if (this._onToggle) {
      document.removeEventListener('toggle', this._onToggle, true);
      this._onToggle = null;
    }
    if (this._overlay) this._overlay.remove();
    if (this._portal) this._portal.remove();
    if (this._onScroll) window.removeEventListener('scroll', this._onScroll, true);
    if (this._onGlobalKey) document.removeEventListener('keydown', this._onGlobalKey, true);
    this._teardownPicker();
  }

  _transition(newState) {
    const prev = this._state;
    this._state = newState;

    // Side effects on state exit
    if (prev.picking && !newState.picking) {
      document.body.style.cursor = '';
    }
    if (prev.name !== 'collapsed' && newState.name === 'collapsed') {
      this._clearHighlight();
    }

    // Side effects on state enter
    if (newState.picking && !prev.picking) {
      document.body.style.cursor = 'crosshair';
    }

    this._render();

    // Scroll selected page element into viewport
    if (newState.selectedEl && newState.selectedEl !== prev.selectedEl) {
      newState.selectedEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }
  }

  // --- Overlay ---

  _setupOverlay() {
    const label = h('div', { style: {
      position: 'absolute', top: '-22px', left: '-2px',
      background: '#00ADD8', color: '#0f172a', fontSize: '11px',
      fontWeight: '600', padding: '1px 6px', borderRadius: '3px 3px 0 0',
      fontFamily: "'JetBrains Mono', monospace", whiteSpace: 'nowrap',
    }});
    this._overlay = h('div', { style: {
      position: 'fixed', pointerEvents: 'none', zIndex: '2147483646',
      border: '2px solid #00ADD8', borderRadius: '4px',
      background: 'rgba(0, 173, 216, 0.08)', display: 'none',
    }}, label);
    this._overlayLabel = label;
    this._portal.appendChild(this._overlay);
    this._highlightedEl = null;
    this._onScroll = () => {
      if (this._highlightedEl) this._positionOverlay(this._highlightedEl);
    };
    window.addEventListener('scroll', this._onScroll, true);
  }

  _positionOverlay(el) {
    const r = el.getBoundingClientRect();
    Object.assign(this._overlay.style, {
      display: 'block', top: r.top + 'px', left: r.left + 'px',
      width: r.width + 'px', height: r.height + 'px',
    });
  }

  _highlightEl(el) {
    if (!el || !this._overlay) return;
    this._highlightedEl = el;
    this._positionOverlay(el);
    this._overlayLabel.textContent = '<' + (el.dataset.htmlcComponent || '?') + '>';
  }

  _clearHighlight() {
    this._highlightedEl = null;
    if (this._overlay) this._overlay.style.display = 'none';
  }

  // --- Picker ---

  _setupPicker() {
    this._onPickerMove = (e) => {
      if (!this._state.picking) return;
      const el = this._findComponent(e.target);
      if (el) this._highlightEl(el);
      else this._clearHighlight();
    };
    this._onPickerClick = (e) => {
      if (!this._state.picking) return;
      e.preventDefault();
      e.stopPropagation();
      const el = this._findComponent(e.target);
      if (el) {
        const id = this._idForEl(el);
        this._transition(this._state.pickElement(id, el));
        this._highlightEl(el);
      }
    };
    this._onPickerKeydown = (e) => {
      if (!this._state.picking) return;
      if (e.key === 'Escape') {
        this._clearHighlight();
        this._transition(this._state.escape());
      }
    };
    document.addEventListener('mousemove', this._onPickerMove, true);
    document.addEventListener('click', this._onPickerClick, true);
    document.addEventListener('keydown', this._onPickerKeydown, true);
  }

  _teardownPicker() {
    document.removeEventListener('mousemove', this._onPickerMove, true);
    document.removeEventListener('click', this._onPickerClick, true);
    document.removeEventListener('keydown', this._onPickerKeydown, true);
  }

  _findComponent(target) {
    let el = target;
    while (el && el !== document.body) {
      if (el.dataset && el.dataset.htmlcComponent) return el;
      el = el.parentElement;
    }
    return null;
  }

  _idForEl(el) {
    for (const [id, mapped] of Object.entries(this._elMap)) {
      if (mapped === el) return id;
    }
    return el.dataset.htmlcComponent + '-picked';
  }

  // --- Tree ---

  _buildTree() {
    const els = document.querySelectorAll('[data-htmlc-component]');
    const nodes = [];
    const stack = [];
    els.forEach(el => {
      const node = {
        name: el.dataset.htmlcComponent,
        file: el.dataset.htmlcFile,
        props: el.dataset.htmlcProps,
        el, children: [], depth: 0,
      };
      while (stack.length > 0 && !stack[stack.length - 1].el.contains(el)) stack.pop();
      if (stack.length > 0) {
        node.depth = stack[stack.length - 1].depth + 1;
        stack[stack.length - 1].children.push(node);
      } else {
        nodes.push(node);
      }
      stack.push(node);
    });
    return nodes;
  }

  _nodeMatches(node, filter) {
    if (node.name.toLowerCase().includes(filter)) return true;
    return node.children.some(c => this._nodeMatches(c, filter));
  }

  _renderNode(node, idx) {
    const filter = this._state.filter.toLowerCase();
    const selfMatches = !filter || node.name.toLowerCase().includes(filter);
    const subtreeMatches = !filter || this._nodeMatches(node, filter);

    if (!subtreeMatches) return null;

    const id = node.name + '-' + idx;

    // Recurse into children first (needed for both matched and skipped nodes)
    const childRendered = node.children.map((c, i) => {
      const childId = c.name + '-' + idx + '-' + i;
      this._nodeParent[childId] = id;
      return this._renderNode(c, idx + '-' + i);
    }).filter(Boolean);

    const childrenEl = childRendered.length > 0 &&
      h('div', { className: 'children' }, ...childRendered);

    // Node doesn't match but descendants do — skip this node, keep children indented
    if (!selfMatches) {
      if (childRendered.length === 0) return null;
      return h('div', { className: 'node-filtered' }, ...childRendered);
    }

    // Node matches — register it for keyboard nav and render fully
    let propsObj = {};
    try { propsObj = JSON.parse(node.props || '{}'); } catch(e) {}
    const propKeys = Object.keys(propsObj);
    const isSelected = this._state.selectedEl === node.el;
    const isFocused = this._state.focusedId === id;
    this._elMap[id] = node.el;
    this._nodeOrder.push(id);
    this._nodeChildren[id] = childRendered.length > 0
      ? node.children.map((c, i) => c.name + '-' + idx + '-' + i).filter(cid => this._elMap[cid])
      : [];

    let cls = 'node-header';
    if (isSelected) cls += ' selected';
    if (isFocused) cls += ' focused';

    const header = h('div', {
      className: cls,
      'data-id': id,
      onmouseenter: () => this._highlightEl(node.el),
      onmouseleave: () => { if (this._state.focusedId !== id) this._clearHighlight(); },
      onclick: () => {
        this._transition(this._state.clickNode(id, node.el));
        this._highlightEl(node.el);
      },
    },
      h('span', { className: 'tag' }, '<' + node.name + '>'),
      h('span', { className: 'file' }, node.file || ''),
      propKeys.length > 0 && h('span', { className: 'prop-count' }, propKeys.length + ' props'),
    );

    const propsPanel = isSelected && propKeys.length > 0 &&
      h('div', { className: 'props-panel' },
        h('div', { className: 'props-title' }, 'Props'),
        h('table', { className: 'props-table' },
          h('tbody', null,
            ...Object.entries(propsObj).map(([k, v]) => {
              const val = typeof v === 'string' ? v : JSON.stringify(v, null, 2);
              const display = val.length > 120 ? val.slice(0, 120) + '\u2026' : val;
              return h('tr', null,
                h('td', { className: 'prop-key' }, k),
                h('td', { className: 'prop-val' }, display),
              );
            }),
          ),
        ),
      );

    return h('div', { className: 'node' }, header, propsPanel, childrenEl);
  }

  // --- Keyboard (Navigating state) ---

  _handleTreeKeydown(e) {
    if (this._state.name !== 'navigating') return;
    if (e.ctrlKey || e.metaKey || e.altKey) return;

    const order = this._nodeOrder;
    if (!order.length) return;

    const idx = this._state.focusedId ? order.indexOf(this._state.focusedId) : -1;

    switch (e.key) {
      case 'ArrowDown':
      case 'j': {
        e.preventDefault();
        const nextId = order[idx < order.length - 1 ? idx + 1 : 0];
        this._transition(this._state.moveFocus(nextId));
        const el = this._elMap[nextId];
        if (el) this._highlightEl(el);
        break;
      }
      case 'ArrowUp':
      case 'k': {
        e.preventDefault();
        const prevId = order[idx > 0 ? idx - 1 : order.length - 1];
        this._transition(this._state.moveFocus(prevId));
        const el = this._elMap[prevId];
        if (el) this._highlightEl(el);
        break;
      }
      case 'ArrowRight':
      case 'l': {
        e.preventDefault();
        const children = this._nodeChildren[this._state.focusedId];
        if (children && children.length > 0) {
          this._transition(this._state.moveFocus(children[0]));
          const el = this._elMap[children[0]];
          if (el) this._highlightEl(el);
        }
        break;
      }
      case 'ArrowLeft':
      case 'h': {
        e.preventDefault();
        const parent = this._nodeParent[this._state.focusedId];
        if (parent) {
          this._transition(this._state.moveFocus(parent));
          const el = this._elMap[parent];
          if (el) this._highlightEl(el);
        }
        break;
      }
      case 'Enter':
      case ' ':
        e.preventDefault();
        this._transition(this._state.selectFocused(this._elMap));
        break;
      case 'Escape':
        e.preventDefault();
        this._clearHighlight();
        this._transition(this._state.escape());
        break;
    }
  }

  // --- Render ---

  _render() {
    this.shadowRoot.replaceChildren();
    this._elMap = {};
    this._nodeOrder = [];
    this._nodeParent = {};
    this._nodeChildren = {};

    const style = h('style', null, STYLES);
    const state = this._state;

    // --- Collapsed ---
    if (state.name === 'collapsed') {
      const btn = h('button', {
        className: 'logo-btn',
        title: 'Open htmlc inspector',
        onclick: () => this._transition(state.clickLogo()),
      });
      btn.innerHTML = LOGO_SVG(36, 36);
      this.shadowRoot.append(style, btn);
      return;
    }

    // --- Open states: Browsing, Navigating, Picking ---
    const tree = this._buildTree();
    const treeNodes = tree.map((n, i) => this._renderNode(n, i)).filter(Boolean);

    const pickBtn = h('button', {
      className: 'icon-btn' + (state.picking ? ' active' : ''),
      title: 'Select element on page',
      onclick: (e) => {
        e.stopPropagation();
        if (state.picking) {
          this._clearHighlight();
          this._transition(state.stopPicking());
        } else {
          this._transition(state.startPicking());
        }
      },
    });
    pickBtn.innerHTML = CROSSHAIR_SVG;

    const closeBtn = h('button', {
      className: 'icon-btn close',
      title: 'Close inspector',
      onclick: () => this._transition(state.close()),
    }, '\u00D7');

    const titleLeft = h('span', { className: 'titlebar-left' });
    titleLeft.innerHTML = LOGO_SVG(20, 20);
    titleLeft.append(' inspector');

    const filterInput = h('input', {
      className: 'filter-input',
      type: 'text',
      placeholder: 'Filter components\u2026',
      value: state.filter,
      oninput: (e) => this._transition(state.inputFilter(e.target.value)),
      onkeydown: (e) => {
        e.stopPropagation();
        if (e.key === 'Enter') {
          e.preventDefault();
          this._transition(state.submitFilter(this._nodeOrder[0], this._elMap));
          const firstEl = this._elMap[this._nodeOrder[0]];
          if (firstEl) this._highlightEl(firstEl);
        }
        if (e.key === 'Escape') {
          e.preventDefault();
          this._transition(state.close());
        }
      },
    });

    const treeEl = h('div', {
      className: 'tree',
      tabindex: '0',
      onkeydown: (e) => this._handleTreeKeydown(e),
    }, ...treeNodes);

    const panel = h('div', { className: 'panel' },
      h('div', { className: 'titlebar' },
        titleLeft,
        h('span', { className: 'titlebar-right' }, pickBtn, closeBtn),
      ),
      h('div', { className: 'filter-bar' }, filterInput),
      treeEl,
    );

    this.shadowRoot.append(style, panel);

    // Scroll focused or selected node into view
    const target = this.shadowRoot.querySelector('.node-header.focused') ||
                   this.shadowRoot.querySelector('.node-header.selected');
    if (target) target.scrollIntoView({ block: 'nearest' });

    // Focus follows state
    if (state.focusTarget === 'filter') {
      filterInput.focus();
      filterInput.selectionStart = filterInput.selectionEnd = filterInput.value.length;
    } else if (state.focusTarget === 'tree') {
      treeEl.focus();
    }
  }
}

if (!customElements.get('htmlc-inspector')) {
  customElements.define('htmlc-inspector', HtmlcInspector);
}

document.body.appendChild(document.createElement('htmlc-inspector'));
