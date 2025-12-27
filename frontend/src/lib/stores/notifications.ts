import { writable } from 'svelte/store';
import type { Notification } from '$lib/api'; // Assuming Notification type is exported from api.ts

interface NotificationState {
  notifications: Notification[];
  unreadCount: number;
  isLoading: boolean;
  error: string | null;
}

const initialState: NotificationState = {
  notifications: [],
  unreadCount: 0,
  isLoading: false,
  error: null,
};

export const notifications = writable<NotificationState>(initialState);

// Function to add a new notification (from WebSocket)
export const addNotification = (newNotification: Notification) => {
  notifications.update((state) => {
    // Add new notification to the beginning of the list
    const updatedNotifications = [newNotification, ...state.notifications];
    // Increment unread count if the new notification is not read
    const newUnreadCount = state.unreadCount + (newNotification.read ? 0 : 1);
    return { ...state, notifications: updatedNotifications, unreadCount: newUnreadCount };
  });
};

// Function to mark a notification as read
export const markNotificationAsRead = (notificationId: string) => {
  notifications.update((state) => {
    const updatedNotifications = state.notifications.map((n) =>
      n.id === notificationId ? { ...n, read: true } : n
    );
    const newUnreadCount = updatedNotifications.filter((n) => !n.read).length;
    return { ...state, notifications: updatedNotifications, unreadCount: newUnreadCount };
  });
};

// Function to set initial notifications (e.g., from API fetch)
export const setNotifications = (fetchedNotifications: Notification[], totalUnread: number) => {
  notifications.update((state) => ({
    ...state,
    notifications: fetchedNotifications,
    unreadCount: totalUnread,
    isLoading: false,
    error: null,
  }));
};

// Helper for infinite scroll
export const appendNotifications = (newNotifications: Notification[]) => {
  notifications.update((state) => {
    // Prevent duplicates
    const existingIds = new Set(state.notifications.map(n => n.id));
    const uniqueNew = newNotifications.filter(n => !existingIds.has(n.id));
    return {
      ...state,
      notifications: [...state.notifications, ...uniqueNew],
    };
  });
}

export const setUnreadCount = (count: number) => {
  notifications.update((state) => ({ ...state, unreadCount: count }));
};

export const setLoading = (loading: boolean) => {
  notifications.update((state) => ({ ...state, isLoading: loading }));
};

export const setError = (error: string | null) => {
  notifications.update((state) => ({ ...state, error }));
};