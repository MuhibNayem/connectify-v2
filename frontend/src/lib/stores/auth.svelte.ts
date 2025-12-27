import { browser } from '$app/environment';

// Define the shape of the user and auth state
import type { User } from '$lib/types';

interface AuthState {
    user: User | null;
    accessToken: string | null;
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api';

// Create the state with Svelte 5 Runes
const authState = $state<AuthState>({
    user: null,
    accessToken: null,
});

function getSessionItem(key: string) {
    if (!browser) return null;
    try {
        return sessionStorage.getItem(key);
    } catch (error) {
        console.error('Session storage read failed:', error);
        return null;
    }
}

function setSessionItem(key: string, value: string) {
    if (!browser) return;
    try {
        sessionStorage.setItem(key, value);
    } catch (error) {
        console.error('Session storage write failed:', error);
    }
}

function removeSessionItem(key: string) {
    if (!browser) return;
    try {
        sessionStorage.removeItem(key);
    } catch (error) {
        console.error('Session storage remove failed:', error);
    }
}

// Function to initialize state from sessionStorage
function initializeState() {
    if (!browser) return;

    const storedUser = getSessionItem('currentUser');

    if (storedUser) {
        try {
            authState.user = JSON.parse(storedUser);
        } catch (e) {
            console.error('Failed to parse stored auth state:', e);
            clearState(); // Clear corrupted data
        }
    }
}

// Function to persist state to sessionStorage
function persistState() {
    if (!browser) return;
    if (authState.user) {
        setSessionItem('currentUser', JSON.stringify(authState.user));
    } else {
        removeSessionItem('currentUser');
    }
}

// Function to clear the auth state
function clearState() {
    authState.user = null;
    authState.accessToken = null;
    persistState();
}

let refreshPromise: Promise<boolean> | null = null;

// Main exportable auth store object
export const auth = {
    // Expose state reactively
    get state() {
        return authState;
    },

    // Initialize the store
    initialize: initializeState,

    // Login method
    login: async (credentials: { email: string; password: string }) => {
        const response = await fetch(`${API_BASE_URL}/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            body: JSON.stringify(credentials),
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Login failed');
        }

        const data = await response.json();
        authState.user = data.user;
        authState.accessToken = data.access_token;
        persistState();
        return data;
    },

    // Register method
    register: async (userData: any) => {
        const response = await fetch(`${API_BASE_URL}/auth/register`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            body: JSON.stringify(userData),
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Registration failed');
        }

        const data = await response.json();
        authState.user = data.user;
        authState.accessToken = data.access_token;
        persistState();
        return data;
    },

    // Logout method
    logout: async () => {
        if (authState.accessToken) {
            try {
                await fetch(`${API_BASE_URL}/auth/logout`, {
                    method: 'POST',
                    headers: { Authorization: `Bearer ${authState.accessToken}` },
                    credentials: 'include',
                });
            } catch (error) {
                console.error('Logout API call failed, clearing state regardless.', error);
            }
        }
        clearState();
    },

    // Token refresh method
    refresh: async (): Promise<boolean> => {
        if (refreshPromise) {
            return refreshPromise;
        }

        refreshPromise = (async () => {
            try {
                const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
                    method: 'POST',
                    credentials: 'include',
                });

                if (!response.ok) {
                    throw new Error('Failed to refresh token');
                }

                const data = await response.json();
                authState.user = data.user;
                authState.accessToken = data.access_token;
                persistState();
                return true;
            } catch (error) {
                console.error('Token refresh failed:', error);
                clearState();
                return false;
            } finally {
                refreshPromise = null;
            }
        })();

        return refreshPromise;
    },

    // Update user method
    updateUser: (userData: Partial<User>) => {
        if (authState.user) {
            authState.user = { ...authState.user, ...userData };
            persistState();
        }
    }
};

// Initialize on module load
initializeState();
if (browser) {
    auth.refresh().catch(() => {
        // swallow errors and keep user logged out if refresh fails
    });
}
