export interface MenuItem {
    label: string;
    category?: string;
    onClick: () => void;
}

export class ContextMenu {
    private el: HTMLDivElement;
    private boundHide: (e: MouseEvent) => void;
    private boundEscape: (e: KeyboardEvent) => void;

    constructor(container: HTMLElement) {
        this.el = document.createElement('div');
        Object.assign(this.el.style, {
            position: 'absolute',
            display: 'none',
            backgroundColor: '#16213e',
            border: '1px solid #0f3460',
            borderRadius: '6px',
            boxShadow: '0 4px 16px rgba(0,0,0,0.5)',
            padding: '4px 0',
            zIndex: '1000',
            minWidth: '160px',
            fontFamily: 'sans-serif',
            fontSize: '13px',
        });
        container.style.position = 'relative';
        container.appendChild(this.el);

        this.boundHide = (e: MouseEvent) => {
            if (!this.el.contains(e.target as HTMLElement)) this.hide();
        };
        this.boundEscape = (e: KeyboardEvent) => {
            if (e.key === 'Escape') this.hide();
        };
    }

    show(x: number, y: number, items: MenuItem[]): void {
        this.el.innerHTML = '';

        // Group by category
        const groups = new Map<string, MenuItem[]>();
        for (const item of items) {
            const cat = item.category || 'other';
            if (!groups.has(cat)) groups.set(cat, []);
            groups.get(cat)!.push(item);
        }

        for (const [category, catItems] of groups) {
            // Category header
            const header = document.createElement('div');
            Object.assign(header.style, {
                padding: '4px 12px 2px',
                color: '#666',
                fontSize: '10px',
                textTransform: 'uppercase',
                letterSpacing: '0.5px',
            });
            header.textContent = category;
            this.el.appendChild(header);

            for (const item of catItems) {
                const row = document.createElement('div');
                Object.assign(row.style, {
                    padding: '6px 12px',
                    color: '#ccc',
                    cursor: 'pointer',
                });
                row.textContent = item.label;
                row.addEventListener('mouseenter', () => { row.style.backgroundColor = '#1a1a4e'; });
                row.addEventListener('mouseleave', () => { row.style.backgroundColor = 'transparent'; });
                row.addEventListener('click', () => { item.onClick(); this.hide(); });
                this.el.appendChild(row);
            }
        }

        this.el.style.left = `${x}px`;
        this.el.style.top = `${y}px`;
        this.el.style.display = 'block';

        // Defer listeners to avoid catching the originating click
        setTimeout(() => {
            document.addEventListener('mousedown', this.boundHide);
            document.addEventListener('keydown', this.boundEscape);
        }, 0);
    }

    hide(): void {
        this.el.style.display = 'none';
        document.removeEventListener('mousedown', this.boundHide);
        document.removeEventListener('keydown', this.boundEscape);
    }
}
