import type { Node, NodeType } from '../core/types.js';

export class ConfigModal {
    private overlay: HTMLDivElement;
    private modal: HTMLDivElement;
    private boundEscape: (e: KeyboardEvent) => void;

    constructor(container: HTMLElement) {
        this.overlay = document.createElement('div');
        Object.assign(this.overlay.style, {
            position: 'absolute',
            top: '0', left: '0', right: '0', bottom: '0',
            backgroundColor: 'rgba(0,0,0,0.6)',
            display: 'none',
            justifyContent: 'center',
            alignItems: 'center',
            zIndex: '2000',
        });

        this.modal = document.createElement('div');
        Object.assign(this.modal.style, {
            backgroundColor: '#16213e',
            border: '1px solid #0f3460',
            borderRadius: '8px',
            padding: '20px',
            minWidth: '280px',
            boxShadow: '0 8px 32px rgba(0,0,0,0.6)',
            fontFamily: 'sans-serif',
            color: '#ccc',
        });

        this.overlay.appendChild(this.modal);
        container.style.position = 'relative';
        container.appendChild(this.overlay);

        this.overlay.addEventListener('click', (e) => {
            if (e.target === this.overlay) this.hide();
        });

        this.boundEscape = (e: KeyboardEvent) => {
            if (e.key === 'Escape') this.hide();
        };
    }

    show(node: Node, _nodeType: NodeType, onSave: (config: Record<string, string>) => void): void {
        this.modal.innerHTML = '';

        // Title
        const title = document.createElement('div');
        Object.assign(title.style, { fontSize: '16px', fontWeight: 'bold', marginBottom: '16px', color: '#fff' });
        title.textContent = `Configure: ${node.label}`;
        this.modal.appendChild(title);

        const config = { ...(node.config || {}) };
        const inputs: Record<string, HTMLInputElement> = {};

        // Config fields - for delay nodes show duration
        const fields: { key: string; label: string; type: string; placeholder: string }[] = [];

        // Auto-detect configurable fields or use known ones
        if (config.duration !== undefined || _nodeType.category === 'delay') {
            fields.push({ key: 'duration', label: 'Duration (ms)', type: 'number', placeholder: '1000' });
        }

        // Also show any existing config keys
        for (const key of Object.keys(config)) {
            if (!fields.find(f => f.key === key)) {
                fields.push({ key, label: key, type: 'text', placeholder: '' });
            }
        }

        if (fields.length === 0) {
            const msg = document.createElement('div');
            msg.style.color = '#666';
            msg.textContent = 'No configurable properties.';
            this.modal.appendChild(msg);
        }

        for (const field of fields) {
            const row = document.createElement('div');
            row.style.marginBottom = '12px';

            const label = document.createElement('label');
            Object.assign(label.style, { display: 'block', fontSize: '12px', marginBottom: '4px', color: '#999' });
            label.textContent = field.label;
            row.appendChild(label);

            const input = document.createElement('input');
            Object.assign(input.style, {
                width: '100%',
                padding: '6px 8px',
                backgroundColor: '#1a1a2e',
                border: '1px solid #0f3460',
                borderRadius: '4px',
                color: '#fff',
                fontSize: '14px',
                boxSizing: 'border-box',
            });
            input.type = field.type;
            input.value = config[field.key] || '';
            input.placeholder = field.placeholder;
            row.appendChild(input);

            inputs[field.key] = input;
            this.modal.appendChild(row);
        }

        // Buttons
        const buttons = document.createElement('div');
        Object.assign(buttons.style, { display: 'flex', justifyContent: 'flex-end', gap: '8px', marginTop: '16px' });

        const cancelBtn = document.createElement('button');
        Object.assign(cancelBtn.style, {
            padding: '6px 16px', backgroundColor: 'transparent', border: '1px solid #0f3460',
            borderRadius: '4px', color: '#999', cursor: 'pointer', fontSize: '13px',
        });
        cancelBtn.textContent = 'Cancel';
        cancelBtn.addEventListener('click', () => this.hide());
        buttons.appendChild(cancelBtn);

        const saveBtn = document.createElement('button');
        Object.assign(saveBtn.style, {
            padding: '6px 16px', backgroundColor: '#e94560', border: 'none',
            borderRadius: '4px', color: '#fff', cursor: 'pointer', fontSize: '13px',
        });
        saveBtn.textContent = 'Save';
        saveBtn.addEventListener('click', () => {
            const result: Record<string, string> = {};
            for (const [key, input] of Object.entries(inputs)) {
                if (input.value) result[key] = input.value;
            }
            onSave(result);
            this.hide();
        });
        buttons.appendChild(saveBtn);

        this.modal.appendChild(buttons);

        this.overlay.style.display = 'flex';
        document.addEventListener('keydown', this.boundEscape);

        // Focus first input
        const firstInput = Object.values(inputs)[0];
        if (firstInput) setTimeout(() => firstInput.focus(), 0);
    }

    hide(): void {
        this.overlay.style.display = 'none';
        document.removeEventListener('keydown', this.boundEscape);
    }
}
