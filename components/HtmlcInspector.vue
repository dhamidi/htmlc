<template>
  <script>
    class HtmlcInspector extends HTMLElement {
      connectedCallback() {
        this.attachShadow({ mode: 'open' });
        this._expanded = true;
        this._selectedEl = null;
        this._overlay = null;
        this._render();
        this._setupOverlay();
      }

      disconnectedCallback() {
        if (this._overlay) this._overlay.remove();
      }

      _setupOverlay() {
        this._overlay = document.createElement('div');
        Object.assign(this._overlay.style, {
          position: 'fixed', pointerEvents: 'none', zIndex: '99998',
          border: '2px solid #00ADD8', borderRadius: '4px',
          background: 'rgba(0, 173, 216, 0.08)', display: 'none',
        });
        const label = document.createElement('div');
        Object.assign(label.style, {
          position: 'absolute', top: '-22px', left: '-2px',
          background: '#00ADD8', color: '#0f172a', fontSize: '11px',
          fontWeight: '600', padding: '1px 6px', borderRadius: '3px 3px 0 0',
          fontFamily: "'JetBrains Mono', monospace', monospace",
          whiteSpace: 'nowrap',
        });
        this._overlay.appendChild(label);
        this._overlayLabel = label;
        document.body.appendChild(this._overlay);
      }

      _highlightEl(el) {
        if (!el || !this._overlay) return;
        const r = el.getBoundingClientRect();
        Object.assign(this._overlay.style, {
          display: 'block', top: r.top + 'px', left: r.left + 'px',
          width: r.width + 'px', height: r.height + 'px',
        });
        this._overlayLabel.textContent = '<' + (el.dataset.htmlcComponent || '?') + '>';
      }

      _clearHighlight() {
        if (this._overlay) this._overlay.style.display = 'none';
      }

      _buildTree() {
        const els = document.querySelectorAll('[data-htmlc-component]');
        const nodes = [];
        const stack = [];
        els.forEach(el => {
          const node = {
            name: el.dataset.htmlcComponent,
            file: el.dataset.htmlcFile,
            props: el.dataset.htmlcProps,
            el: el,
            children: [],
            depth: 0,
          };
          while (stack.length > 0 && !stack[stack.length - 1].el.contains(el)) {
            stack.pop();
          }
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

      _renderNode(node) {
        let propsObj = {};
        try { propsObj = JSON.parse(node.props || '{}'); } catch(e) {}
        const propKeys = Object.keys(propsObj);
        const isSelected = this._selectedEl === node.el;

        let html = '<div class="node">';
        html += '<div class="node-header' + (isSelected ? ' selected' : '') + '" data-idx="' + node.name + '">';
        html += '<span class="tag">&lt;' + node.name + '&gt;</span>';
        html += '<span class="file">' + (node.file || '') + '</span>';
        if (propKeys.length > 0) {
          html += '<span class="prop-count">' + propKeys.length + ' props</span>';
        }
        html += '</div>';

        if (isSelected && propKeys.length > 0) {
          html += '<div class="props-panel">';
          html += '<div class="props-title">Props</div>';
          html += '<table class="props-table">';
          for (const [k, v] of Object.entries(propsObj)) {
            const val = typeof v === 'string' ? v : JSON.stringify(v);
            const display = val.length > 80 ? val.slice(0, 80) + '...' : val;
            html += '<tr><td class="prop-key">' + k + '</td>';
            html += '<td class="prop-val">' + display.replace(/</g, '&lt;').replace(/>/g, '&gt;') + '</td></tr>';
          }
          html += '</table></div>';
        }

        if (node.children.length > 0) {
          html += '<div class="children">';
          node.children.forEach(c => { html += this._renderNode(c); });
          html += '</div>';
        }
        html += '</div>';
        return html;
      }

      _render() {
        const tree = this._buildTree();
        let treeHtml = '';
        tree.forEach(n => { treeHtml += this._renderNode(n); });

        this.shadowRoot.innerHTML = `
          <style>
            :host {
              position: fixed;
              bottom: 16px;
              right: 16px;
              z-index: 99999;
              font-family: 'JetBrains Mono', ui-monospace, monospace;
              font-size: 12px;
            }
            .panel {
              background: #0f172a;
              border: 1px solid #334155;
              border-radius: 10px;
              box-shadow: 0 8px 32px rgba(0,0,0,0.5);
              width: 380px;
              max-height: 480px;
              display: flex;
              flex-direction: column;
              overflow: hidden;
            }
            .panel.collapsed { max-height: none; }
            .titlebar {
              display: flex;
              align-items: center;
              justify-content: space-between;
              padding: 8px 12px;
              background: #1e293b;
              border-bottom: 1px solid #334155;
              cursor: pointer;
              user-select: none;
            }
            .titlebar-left {
              display: flex;
              align-items: center;
              gap: 6px;
              color: #00ADD8;
              font-weight: 600;
              font-size: 12px;
            }
            .toggle-btn {
              color: #94a3b8;
              font-size: 14px;
              background: none;
              border: none;
              cursor: pointer;
              padding: 0 4px;
            }
            .tree {
              overflow-y: auto;
              padding: 8px;
              flex: 1;
            }
            .node { margin-left: 0; }
            .children { margin-left: 14px; border-left: 1px solid #334155; padding-left: 8px; }
            .node-header {
              display: flex;
              align-items: center;
              gap: 8px;
              padding: 4px 6px;
              border-radius: 4px;
              cursor: pointer;
              color: #94a3b8;
              line-height: 1.4;
            }
            .node-header:hover { background: #1e293b; }
            .node-header.selected { background: rgba(0, 173, 216, 0.12); }
            .tag { color: #00ADD8; font-weight: 500; white-space: nowrap; }
            .file { color: #475569; font-size: 10px; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
            .prop-count {
              color: #64748b;
              font-size: 10px;
              background: #1e293b;
              padding: 1px 5px;
              border-radius: 3px;
              white-space: nowrap;
            }
            .props-panel {
              margin: 2px 0 6px 6px;
              padding: 8px;
              background: #1e293b;
              border-radius: 6px;
              border: 1px solid #334155;
            }
            .props-title {
              color: #64748b;
              font-size: 10px;
              text-transform: uppercase;
              letter-spacing: 0.05em;
              margin-bottom: 6px;
            }
            .props-table { width: 100%; border-collapse: collapse; }
            .props-table td {
              padding: 2px 0;
              vertical-align: top;
            }
            .prop-key {
              color: #8b5cf6;
              padding-right: 8px;
              white-space: nowrap;
              width: 1%;
            }
            .prop-val {
              color: #94a3b8;
              word-break: break-all;
              font-size: 11px;
            }
          </style>
          <div class="panel ${this._expanded ? '' : 'collapsed'}">
            <div class="titlebar" id="titlebar">
              <span class="titlebar-left">htmlc inspector</span>
              <button class="toggle-btn" id="toggle">${this._expanded ? '\u25BC' : '\u25B6'}</button>
            </div>
            ${this._expanded ? '<div class="tree" id="tree">' + treeHtml + '</div>' : ''}
          </div>
        `;

        this.shadowRoot.getElementById('titlebar').addEventListener('click', () => {
          this._expanded = !this._expanded;
          this._clearHighlight();
          this._render();
        });

        if (this._expanded) {
          this.shadowRoot.querySelectorAll('.node-header').forEach(header => {
            header.addEventListener('mouseenter', () => {
              const name = header.dataset.idx;
              const el = document.querySelector('[data-htmlc-component="' + name + '"]');
              if (el) this._highlightEl(el);
            });
            header.addEventListener('mouseleave', () => this._clearHighlight());
            header.addEventListener('click', (e) => {
              const name = header.dataset.idx;
              const el = document.querySelector('[data-htmlc-component="' + name + '"]');
              this._selectedEl = this._selectedEl === el ? null : el;
              this._render();
            });
          });
        }
      }
    }

    if (!customElements.get('htmlc-inspector')) {
      customElements.define('htmlc-inspector', HtmlcInspector);
    }
  </script>
  <htmlc-inspector></htmlc-inspector>
</template>
