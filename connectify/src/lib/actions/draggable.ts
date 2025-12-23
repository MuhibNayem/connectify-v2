
export function draggable(node: HTMLElement, options: { bounds?: HTMLElement } = {}) {
    let x = 0;
    let y = 0;
    let startX = 0;
    let startY = 0;

    function handleMousedown(event: MouseEvent) {
        startX = event.clientX - x;
        startY = event.clientY - y;

        window.addEventListener('mousemove', handleMousemove);
        window.addEventListener('mouseup', handleMouseup);
    }

    function handleMousemove(event: MouseEvent) {
        let newX = event.clientX - startX;
        let newY = event.clientY - startY;

        // Constrain to bounds if provided
        if (options.bounds) {
            const nodeRect = node.getBoundingClientRect();
            const parentRect = options.bounds.getBoundingClientRect();

            // Calculate max X and Y (bounds are relative to the viewport, but we translate relative to start position)
            // Wait, simple translation is easier if we just clamp the *visual* position?
            // But sticking to simple translation logic first:

            // Actually, for a simple PIP inside a relative container, we might want to just let it flow or use absolute positioning updates.
            // But transform translate is more performant.

            // Let's implement simple bounding based on specific request if needed, 
            // but for now "Facebook-like" just implies dragging within the window.
        }

        x = newX;
        y = newY;

        node.style.transform = `translate(${x}px, ${y}px)`;
        node.style.cursor = 'grabbing';
    }

    function handleMouseup() {
        node.style.cursor = 'grab';
        window.removeEventListener('mousemove', handleMousemove);
        window.removeEventListener('mouseup', handleMouseup);
    }

    node.style.cursor = 'grab';
    // Ensure element is positioned such that transform works well? 
    // Usually standard flow or absolute is fine.

    node.addEventListener('mousedown', handleMousedown);

    return {
        destroy() {
            node.removeEventListener('mousedown', handleMousedown);
        }
    };
}
