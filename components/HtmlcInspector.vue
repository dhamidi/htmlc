<template>
  <htmlc-inspector></htmlc-inspector>
</template>

<script>(function () {
  if (customElements.get('htmlc-inspector')) return;

  class HtmlcInspector extends HTMLElement {
    connectedCallback() {
      this._rendering = false;
      this._open = true;
      this._selected = null;
      this._render();

      document.addEventListener('keydown', (e) => {
        if (e.altKey && e.shiftKey && e.key === 'D') {
          this._open = !this._open;
          this._render();
        }
      });
    }

    _render() {
      if (this._rendering) return;
      this._rendering = true;
      try {
        if (!this.shadowRoot) {
          this.attachShadow({ mode: 'open' });
        }
        const shadow = this.shadowRoot;
        const nodes = Array.from(document.querySelectorAll('[data-htmlc-component]'));

        shadow.innerHTML = `
          <style>
            :host { all: initial; }
            aside {
              position: fixed;
              top: 0;
              right: 0;
              width: 320px;
              max-height: 100vh;
              overflow-y: auto;
              background: #1e1e2e;
              color: #cdd6f4;
              font-family: monospace;
              font-size: 13px;
              z-index: 2147483647;
              box-shadow: -2px 0 8px rgba(0,0,0,0.4);
              display: ${this._open ? 'block' : 'none'};
            }
            #titlebar {
              padding: 8px 12px;
              background: #313244;
              cursor: pointer;
              user-select: none;
              display: flex;
              justify-content: space-between;
              align-items: center;
            }
            #titlebar:hover { background: #45475a; }
            .node-header {
              padding: 4px 12px;
              cursor: pointer;
              white-space: nowrap;
              overflow: hidden;
              text-overflow: ellipsis;
            }
            .node-header:hover { background: #313244; }
            .node-header.selected { background: #45475a; }
            .node-props {
              padding: 4px 12px 8px 24px;
              white-space: pre-wrap;
              word-break: break-all;
              font-size: 11px;
              color: #a6e3a1;
            }
          </style>
          <aside>
            <div id="titlebar">
              <span>htmlc inspector</span>
              <span>${this._open ? '\u2715' : '\u2630'}</span>
            </div>
            <div id="tree"></div>
          </aside>`;

        shadow.getElementById('titlebar').addEventListener('click', (e) => {
          e.stopPropagation();
          this._open = !this._open;
          this._render();
        });

        const tree = shadow.getElementById('tree');
        nodes.forEach((node, idx) => {
          const name = node.dataset.htmlcComponent;
          const file = node.dataset.htmlcFile || '';
          const propsRaw = node.dataset.htmlcProps || '{}';
          const isSelected = this._selected === idx;

          const header = document.createElement('div');
          header.className = 'node-header' + (isSelected ? ' selected' : '');
          header.dataset.idx = String(idx);
          header.textContent = name + (file ? ' (' + file + ')' : '');
          tree.appendChild(header);

          if (isSelected) {
            const propsEl = document.createElement('pre');
            propsEl.className = 'node-props';
            try {
              propsEl.textContent = JSON.stringify(JSON.parse(propsRaw), null, 2);
            } catch (_) {
              propsEl.textContent = propsRaw;
            }
            tree.appendChild(propsEl);
          }

          header.addEventListener('click', (e) => {
            e.stopPropagation();
            const name = header.dataset.idx;
            const i = Number(name);
            this._selected = this._selected === i ? null : i;
            this._render();
          });
        });
      } finally {
        this._rendering = false;
      }
    }
  }

  customElements.define('htmlc-inspector', HtmlcInspector);
})();</script>
