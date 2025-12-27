import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import { addNotification } from './stores/notifications';
import type { Notification } from './api';
import { updateUserStatus } from './stores/presence';
import { voiceCallService } from './stores/voice-call.svelte';
import { auth } from '$lib/stores/auth.svelte';
import type { ReactionEvent, ReadReceiptEvent, MessageEditedEvent, MessageCreatedEvent } from '$lib/types';

const WS_AUTH_PROTOCOL = 'connectify.auth';

export interface WebSocketEvent {
	type: string;
	data: any; // Can be any payload
}

export const websocketMessages = writable<WebSocketEvent | null>(null);
let ws: WebSocket | null = null;
let reconnectInterval: number | null = null;

const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8081/ws';

export function connectWebSocket() {
	if (!browser) return;

	if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
		return;
	}

	const token = getActiveAccessToken();
	if (!token) {
		console.log('No access token found, WebSocket not connecting.');
		return;
	}

	console.log('Attempting to connect WebSocket...');
	const url = WS_URL;
	const protocols = [WS_AUTH_PROTOCOL, token];
	ws = new WebSocket(url, protocols);

	ws.onopen = () => {
		console.log('WebSocket connected.');
		if (reconnectInterval) {
			clearInterval(reconnectInterval);
			reconnectInterval = null;
		}
	};

	ws.onmessage = (event) => {
		try {
			const parsedEvent: WebSocketEvent = JSON.parse(event.data);
			console.log('WebSocket event received:', parsedEvent);

			switch (parsedEvent.type) {
				case 'NOTIFICATION_CREATED':
					addNotification(parsedEvent.data as Notification);
					break;
				case 'MESSAGE_REACTION_UPDATE':
					// Ensure the data is correctly typed as ReactionEvent
					websocketMessages.set({
						type: parsedEvent.type,
						data: parsedEvent.data as ReactionEvent
					});
					break;
				case 'MESSAGE_READ_UPDATE':
					// Ensure the data is correctly typed as ReadReceiptEvent
					websocketMessages.set({
						type: parsedEvent.type,
						data: parsedEvent.data as ReadReceiptEvent
					});
					break;
				case 'MESSAGE_EDITED_UPDATE':
					// Ensure the data is correctly typed as MessageEditedEvent
					websocketMessages.set({
						type: parsedEvent.type,
						data: parsedEvent.data as MessageEditedEvent
					});
					break;
				case 'MESSAGE_CREATED':
					websocketMessages.set({
						type: parsedEvent.type,
						data: parsedEvent.data as MessageCreatedEvent
					});
					break;
				case 'presence_update':
					const { user_id, status, last_seen } = parsedEvent.data;
					updateUserStatus(user_id, status, last_seen);
					break;
				case 'VOICE_CALL_SIGNAL':
					// Ensure parsedEvent.data is passed correctly
					voiceCallService.handleIncomingSignal(parsedEvent.data);
					break;
				// For other events, we update the generic store for other components to use
				default:
					websocketMessages.set(parsedEvent);
					break;
			}
		} catch (e) {
			console.error('Failed to parse WebSocket message:', e, event.data);
		}
	};

	ws.onclose = (event) => {
		console.log('WebSocket disconnected:', event.code, event.reason);
		if (!reconnectInterval) {
			reconnectInterval = window.setInterval(() => {
				connectWebSocket();
			}, 3000);
		}
	};

	ws.onerror = (error) => {
		console.error('WebSocket error:', error);
		ws?.close();
	};
}

export function disconnectWebSocket() {
	if (ws) {
		if (reconnectInterval) {
			clearInterval(reconnectInterval);
			reconnectInterval = null;
		}
		ws.close();
		ws = null;
		console.log('WebSocket disconnected manually.');
	}
}

export function sendWebSocketMessage(type: string, payload: any) {
	if (ws && ws.readyState === WebSocket.OPEN) {
		ws.send(JSON.stringify({ type, payload }));
	} else {
		console.warn('WebSocket not open. Message not sent:', type, payload);
	}
}

function getActiveAccessToken(): string | null {
	if (auth.state.accessToken) {
		return auth.state.accessToken;
	}
	return null;
}
