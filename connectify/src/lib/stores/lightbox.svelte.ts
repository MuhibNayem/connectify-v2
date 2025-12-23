export type LightboxMedia = {
    url: string;
    type: 'image' | 'video';
    name?: string;
};

function createLightboxStore() {
    let isOpen = $state(false);
    let media = $state<LightboxMedia[]>([]);
    let currentIndex = $state(0);

    return {
        get isOpen() {
            return isOpen;
        },
        get media() {
            return media;
        },
        get currentIndex() {
            return currentIndex;
        },
        get currentItem() {
            return media[currentIndex];
        },
        open(items: LightboxMedia[], index = 0) {
            media = items;
            currentIndex = index;
            isOpen = true;
        },
        close() {
            isOpen = false;
            media = [];
            currentIndex = 0;
        },
        next() {
            if (currentIndex < media.length - 1) {
                currentIndex++;
            }
        },
        prev() {
            if (currentIndex > 0) {
                currentIndex--;
            }
        },
        setIndex(index: number) {
            if (index >= 0 && index < media.length) {
                currentIndex = index;
            }
        }
    };
}

export const lightbox = createLightboxStore();
