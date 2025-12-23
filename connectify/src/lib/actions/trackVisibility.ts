export function trackVisibility(node: HTMLElement, options: { onVisible: () => void }) {
    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                options.onVisible();
            }
        });
    });

    observer.observe(node);

    return {
        destroy() {
            observer.unobserve(node);
        }
    };
}