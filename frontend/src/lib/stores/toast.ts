import { writable } from 'svelte/store';

export interface Toast {
  id: number;
  message: string;
  type: 'info' | 'success' | 'error';
}

const toasts = writable<Toast[]>([]);

export const toastStore = {
  subscribe: toasts.subscribe,
  add: (message: string, type: 'info' | 'success' | 'error' = 'info') => {
    const id = Date.now();
    toasts.update((all) => [...all, { id, message, type }]);
    setTimeout(() => toastStore.remove(id), 3000); // Auto-remove after 3 seconds
  },
  remove: (id: number) => {
    toasts.update((all) => all.filter((t) => t.id !== id));
  },
};
