import { writable } from 'svelte/store';

export interface PresenceState {
  [userId: string]: {
    status: 'online' | 'offline';
    last_seen: number;
  };
}

const initialState: PresenceState = {};

export const presenceStore = writable<PresenceState>(initialState);

export function updateUserStatus(userId: string, status: 'online' | 'offline', last_seen: number) {
  presenceStore.update(state => {
    return {
      ...state,
      [userId]: {
        status,
        last_seen,
      },
    };
  });
}
