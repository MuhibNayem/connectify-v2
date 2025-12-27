import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { auth } from './stores/auth.svelte';

export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api'; // Backend API URL

// --- Backend Model Interfaces ---

export interface MessageReaction {
	user_id: string;
	emoji: string;
	timestamp: string;
}

export interface Message {
	id: string;
	sender_id: string;
	sender_name?: string;
	receiver_id?: string;
	group_id?: string;
	group_name?: string;
	content?: string;
	content_type: string;
	media_urls?: string[];
	seen_by: string[];
	delivered_to?: string[];
	is_deleted: boolean;
	deleted_at?: string;
	is_edited: boolean;
	edited_at?: string;
	reactions?: MessageReaction[];
	reply_to_message_id?: string;
	product_id?: string;
	is_marketplace?: boolean;
	created_at: string;
	updated_at?: string;
}

export interface MessageRequest {
	receiver_id?: string;
	sender_id?: string; // Will be set by backend from auth
	group_id?: string;
	content?: string;
	content_type: string;
	media_urls?: string[];
	reply_to_message_id?: string;
	product_id?: string;
	is_marketplace?: boolean;
}

export interface MessageResponse {
	messages: Message[];
	total: number;
	page: number;
	limit: number;
	has_more: boolean;
}

export interface ConversationSummary {
	id: string;
	name: string;
	avatar?: string;
	is_group: boolean;
	last_message_content?: string;
	last_message_timestamp?: string;
	last_message_sender_id?: string;
	last_message_sender_name?: string;
	last_message_is_encrypted?: boolean;
	unread_count: number;
}

export type FriendshipStatus = 'pending' | 'accepted' | 'rejected' | 'blocked';

export interface Friendship {
	id: string;
	requester_id: string;
	receiver_id: string;
	status: FriendshipStatus;
	created_at: string;
	updated_at: string;
}

export interface UserShortResponse {
	id: string;
	username: string;
	email: string;
	avatar?: string;
	full_name?: string;
}

export interface PopulatedFriendship {
	id: string;
	requester_id: string;
	receiver_id: string;
	requester_info: UserShortResponse;
	receiver_info: UserShortResponse;
	status: FriendshipStatus;
	created_at: string;
	updated_at: string;
}

export interface FriendshipStatusResponse {
	are_friends: boolean;
	request_sent: boolean;
	request_received: boolean;
	is_blocked_by_viewer: boolean;
	has_blocked_viewer: boolean;
}

export interface GroupResponse {
	id: string;
	name: string;
	creator: UserShortResponse;
	members: UserShortResponse[];
	admins: UserShortResponse[];
	created_at: string;
	updated_at: string;
}

export interface CreateGroupRequest {
	name: string;
	member_ids: string[];
	avatar?: string;
}

export interface AddMemberRequest {
	user_id: string;
}

export interface UpdateGroupRequest {
	name?: string;
}

export interface SuccessResponse {
	success: boolean;
}

export interface ErrorResponse {
	error: string;
}

// Privacy settings type
export type PrivacyType = 'PUBLIC' | 'FRIENDS' | 'ONLY_ME' | 'FRIENDS_OF_FRIENDS' | 'CUSTOM';

// Album types
export type AlbumType = 'custom' | 'profile' | 'cover' | 'timeline';

export interface Album {
	id: string;
	user_id: string;
	name: string;
	description?: string;
	type: AlbumType;
	cover_url?: string;
	post_ids: string[];
	privacy: PrivacyType;
	created_at: string;
	updated_at: string;
}

export interface CreateAlbumRequest {
	name: string;
	description?: string;
	privacy?: PrivacyType;
}

export interface UpdateAlbumRequest {
	name?: string;
	description?: string;
	cover_url?: string;
	privacy?: PrivacyType;
}

export interface UnreadCountResponse {
	count: number;
}

// Define Notification type based on backend model
export interface Notification {
	id: string;
	recipient_id: string;
	sender_id: string;
	type: string; // e.g., "MENTION", "LIKE"
	target_id: string;
	target_type: string; // e.g., "post", "comment"
	content: string;
	data?: Record<string, any>; // Structured data for the notification
	read: boolean;
	created_at: string; // ISO 8601 string
	user_id?: string; // Optional, for events related to user actions
}

export interface NotificationListResponse {
	notifications: Notification[];
	total: number;
	page: number;
	limit: number;
}


// Generic API request function with authentication and token refresh
export async function apiRequest(
	method: string,
	path: string,
	data?: any,
	requiresAuth: boolean = true
): Promise<any> {
	const headers: HeadersInit = {};

	if (!(data instanceof FormData)) {
		headers['Content-Type'] = 'application/json';
	}

	if (requiresAuth && auth.state.accessToken) {
		headers['Authorization'] = `Bearer ${auth.state.accessToken}`;
	}

	const config: RequestInit = {
		method: method,
		headers: headers,
		body: data instanceof FormData ? data : (data ? JSON.stringify(data) : undefined)
	};

	try {
		let response = await fetch(`${API_BASE_URL}${path}`, config);

		// If unauthorized and requires auth, try to refresh token
		if (response.status === 401 && requiresAuth) {
			const refreshed = await auth.refresh();
			if (refreshed) {
				// Retry original request with new token
				if (auth.state.accessToken) {
					headers['Authorization'] = `Bearer ${auth.state.accessToken}`;
					config.headers = headers;
				}
				response = await fetch(`${API_BASE_URL}${path}`, config);
			} else {
				// If refresh failed, redirect to login
				if (browser) {
					goto('/');
				}
				throw new Error('Authentication expired. Please log in again.');
			}
		}

		if (!response.ok) {
			const errorData = await response.json();
			throw new Error(errorData.error || 'Something went wrong');
		}

		// For requests that don't return a body (e.g., 204 No Content)
		if (response.status === 204) {
			return;
		}

		return await response.json();
	} catch (error) {
		console.error('API Request Error:', error);
		throw error;
	}
}

// Notification API functions (they now use the new apiRequest)
export async function fetchNotifications(
	page: number = 1,
	limit: number = 10,
	readStatus?: boolean
): Promise<NotificationListResponse> {
	let path = `/notifications?page=${page}&limit=${limit}`;
	if (readStatus !== undefined) {
		path += `&read=${readStatus}`;
	}
	return apiRequest('GET', path, undefined, true);
}

export async function markNotificationAsRead(notificationId: string): Promise<void> {
	await apiRequest('PUT', `/notifications/${notificationId}/read`, undefined, true);
}

export async function getUnreadNotificationCount(): Promise<{ count: number }> {
	return apiRequest('GET', '/notifications/unread', undefined, true);
}


// Chat-related API functions

export async function getFriendships(status?: FriendshipStatus, page: number = 1, limit: number = 10): Promise<{ data: PopulatedFriendship[]; total: number; page: number; totalPages: number }> {
	let path = `/friendships?page=${page}&limit=${limit}`;
	if (status) {
		path += `&status=${status}`;
	}
	return apiRequest('GET', path, undefined, true);
}

export async function getFriends(): Promise<import('$lib/types').User[]> {
	const result = await getFriendships('accepted', 1, 100);
	const currentUserId = auth.state.user?.id;

	if (!currentUserId) return [];

	return result.data.map((f) => {
		if (f.requester_id === currentUserId) {
			return f.receiver_info as unknown as import('$lib/types').User;
		} else {
			return f.requester_info as unknown as import('$lib/types').User;
		}
	});
}



export async function searchFriends(query: string, limit: number = 20): Promise<import('$lib/types').User[]> {
	const params = new URLSearchParams();
	params.set('query', query);
	params.set('limit', String(limit));
	return apiRequest('GET', `/friendships/search?${params.toString()}`, undefined, true);
}

export async function sendFriendRequest(receiverId: string): Promise<Friendship> {
	return apiRequest('POST', '/friendships/requests', { receiver_id: receiverId }, true);
}

export async function respondToFriendRequest(friendshipId: string, accept: boolean): Promise<SuccessResponse> {
	return apiRequest('POST', `/friendships/requests/respond/${friendshipId}`, { accept }, true);
}

export async function checkFriendshipStatus(otherUserId: string): Promise<FriendshipStatusResponse> {
	return apiRequest('GET', `/friendships/check?other_user_id=${otherUserId}`, undefined, true);
}

export async function unfriendUser(friendId: string): Promise<SuccessResponse> {
	return apiRequest('DELETE', `/friendships/${friendId}`, undefined, true);
}

export async function blockUser(userId: string): Promise<SuccessResponse> {
	return apiRequest('POST', `/friendships/block/${userId}`, undefined, true);
}

export async function unblockUser(userId: string): Promise<SuccessResponse> {
	return apiRequest('DELETE', `/friendships/block/${userId}`, undefined, true);
}

export async function isUserBlocked(userId: string): Promise<{ is_blocked: boolean }> {
	return apiRequest('GET', `/friendships/block/${userId}/status`, undefined, true);
}

export async function getBlockedUsers(): Promise<{ blocked_users: UserShortResponse[] }> {
	return apiRequest('GET', '/friendships/blocked', undefined, true);
}

export async function getUserGroups(): Promise<GroupResponse[]> {
	return apiRequest('GET', '/groups', undefined, true);
}

export async function getMessages(params: { receiverID?: string; groupID?: string; conversationID?: string; page?: number; limit?: number; before?: string; marketplace?: boolean }): Promise<MessageResponse> {
	const query = new URLSearchParams();
	if (params.receiverID) query.set('receiverID', params.receiverID);
	if (params.groupID) query.set('groupID', params.groupID);
	if (params.conversationID) query.set('conversationID', params.conversationID);
	if (params.page) query.set('page', String(params.page));
	if (params.limit) query.set('limit', String(params.limit));
	if (params.before) query.set('before', params.before);
	if (params.marketplace) query.set('marketplace', 'true');

	return apiRequest('GET', `/messages?${query.toString()}`, undefined, true);
}

export async function sendMessage(payload: MessageRequest | FormData): Promise<Message> {
	return apiRequest('POST', '/messages', payload, true);
}

export async function markMessagesAsSeen(conversationId: string, messageIds: string[]): Promise<void> {
	await apiRequest('POST', '/messages/seen', { conversation_id: conversationId, message_ids: messageIds }, true);
}

export async function markMessagesAsDelivered(conversationId: string, messageIds: string[]): Promise<void> {
	await apiRequest('POST', '/messages/delivered', { conversation_id: conversationId, message_ids: messageIds }, true);
}

export async function getUnreadMessageCount(): Promise<UnreadCountResponse> {
	return apiRequest('GET', '/messages/unread', undefined, true);
}

export async function deleteMessage(messageId: string, conversationId: string): Promise<SuccessResponse> {
	return apiRequest('DELETE', `/messages/${messageId}?conversation_id=${conversationId}`, undefined, true);
}

export async function editMessage(messageId: string, content: string): Promise<Message> {
	return apiRequest('PUT', `/messages/${messageId}`, { content }, true);
}

export async function searchMessages(query: string, page: number = 1, limit: number = 20): Promise<Message[]> {
	const params = new URLSearchParams();
	params.set('q', query);
	params.set('page', String(page));
	params.set('limit', String(limit));
	return apiRequest('GET', `/messages/search?${params.toString()}`, undefined, true);
}

export async function addMessageReaction(messageId: string, emoji: string): Promise<SuccessResponse> {
	return apiRequest('POST', `/messages/${messageId}/react`, { emoji }, true);
}

export async function removeMessageReaction(messageId: string, emoji: string): Promise<SuccessResponse> {
	return apiRequest('DELETE', `/messages/${messageId}/react`, { emoji }, true);
}

export async function getConversationSummaries(): Promise<ConversationSummary[]> {
	return apiRequest('GET', '/conversations', undefined, true);
}

export async function markConversationAsSeen(
	conversationId: string,
	timestamp: string,
	isGroup: boolean,
	conversationKey?: string
): Promise<void> {
	const parts = conversationId.split('-');
	const id = parts.length > 1 ? parts[1] : parts[0];
	const payload: Record<string, any> = { timestamp, is_group: isGroup };
	if (conversationKey) {
		payload.conversation_key = conversationKey;
	}
	await apiRequest('POST', `/conversations/${id}/seen`, payload, true);
}

export interface GroupResponse {
	id: string;
	name: string;
	avatar?: string;
	creator: UserShortResponse;
	members: UserShortResponse[];
	pending_members?: UserShortResponse[];
	admins: UserShortResponse[];
	settings?: GroupSettings;
	created_at: string;
	updated_at: string;
}

export interface GroupSettings {
	requires_approval: boolean;
}

export interface CreateGroupRequest {
	name: string;
	member_ids: string[];
}

export interface AddMemberRequest {
	user_id: string;
}

export interface UpdateGroupRequest {
	name?: string;
	avatar?: string;
}

export interface UpdateGroupSettingsRequest {
	requires_approval: boolean;
}

export async function createGroup(payload: CreateGroupRequest): Promise<GroupResponse> {
	return apiRequest('POST', '/groups', payload, true);
}

export async function getGroupDetails(groupId: string): Promise<GroupResponse> {
	return apiRequest('GET', `/groups/${groupId}`, undefined, true);
}

export async function inviteMemberToGroup(groupId: string, userId: string): Promise<void> {
	return apiRequest('POST', `/groups/${groupId}/invite`, { user_id: userId }, true);
}

export async function approveGroupMember(groupId: string, userId: string): Promise<void> {
	return apiRequest('POST', `/groups/${groupId}/approve`, { user_id: userId }, true);
}

export async function rejectGroupMember(groupId: string, userId: string): Promise<void> {
	return apiRequest('POST', `/groups/${groupId}/reject`, { user_id: userId }, true);
}

export async function addAdminToGroup(groupId: string, userId: string): Promise<void> {
	return apiRequest('POST', `/groups/${groupId}/admins`, { user_id: userId }, true);
}

export async function removeAdminFromGroup(groupId: string, userId: string): Promise<void> {
	return apiRequest('DELETE', `/groups/${groupId}/admins/${userId}`, undefined, true);
}

export async function removeMemberFromGroup(groupId: string, userId: string): Promise<void> {
	return apiRequest('DELETE', `/groups/${groupId}/members/${userId}`, undefined, true);
}

export async function updateGroup(groupId: string, payload: UpdateGroupRequest): Promise<GroupResponse> {
	return apiRequest('PUT', `/groups/${groupId}`, payload, true);
}

export async function updateGroupSettings(groupId: string, settings: UpdateGroupSettingsRequest): Promise<void> {
	return apiRequest('PUT', `/groups/${groupId}/settings`, settings, true);
}

// Keep existing functions that are not directly covered by the new API spec or are client-specific
export async function register(userData: any): Promise<any> {
	return apiRequest('POST', '/auth/register', userData, false);
}

export async function updatePost(postId: string, data: any): Promise<any> {
	return apiRequest('PUT', `/posts/${postId}`, data, true);
}

// Note: updateUserProfile returns the updated user object
export async function updateUserProfile(data: any): Promise<import('$lib/types').User> {
	return apiRequest('PUT', '/users/me', data, true);
}

export async function updatePrivacySettings(data: any): Promise<void> {
	return apiRequest('PUT', '/users/me/privacy', data, true);
}

export async function updateNotificationSettings(data: any): Promise<void> {
	return apiRequest('PUT', '/users/me/notifications', data, true);
}

export async function uploadFiles(files: File[]): Promise<{ url: string; type: string }[]> {
	const formData = new FormData();
	files.forEach((f) => formData.append('files[]', f));
	return apiRequest('POST', '/upload', formData, true);
}

export async function getUserByID(userId: string): Promise<import('$lib/types').User> {
	return apiRequest('GET', `/users/${userId}`, undefined, true);
}

// E2EE Key Management
export interface UserKeys {
	public_key: string;
	encrypted_private_key?: string;
	key_backup_iv?: string;
	key_backup_salt?: string;
}

export async function updateUserKeys(keys: UserKeys): Promise<void> {
	return apiRequest('PUT', '/users/me/keys', keys, true);
}

// Note: getUserPublicKey logic effectively just reuses getUserProfile / getUserById
// but we add a semantic helper if needed. For now, we rely on the User object having public_key.



// Community Types & APIs

export type CommunityPrivacy = 'public' | 'closed' | 'secret';

export interface CommunitySettings {
	require_post_approval: boolean;
	require_join_approval: boolean;
	allow_member_posts?: boolean;
}

export interface CommunityStats {
	member_count: number;
	post_count: number;
}

export interface Community {
	id: string;
	name: string;
	description: string;
	slug: string;
	category: string;
	avatar: string;
	cover_image: string;
	privacy: CommunityPrivacy;
	visibility: 'visible' | 'hidden';
	settings: CommunitySettings;
	stats: CommunityStats;
	is_member?: boolean;
	is_admin?: boolean;
	is_pending?: boolean;
	created_at: string;
}

export interface CreateCommunityRequest {
	name: string;
	description: string;
	category: string;
	privacy: CommunityPrivacy;
	require_post_approval: boolean;
	require_join_approval: boolean;
	allow_member_posts?: boolean;
}

export interface UpdateCommunityRequest {
	name?: string;
	description?: string;
	category?: string;
	avatar?: string;
	cover_image?: string;
	privacy?: CommunityPrivacy;
	visibility?: 'visible' | 'hidden';
	require_post_approval?: boolean;
	require_join_approval?: boolean;
	allow_member_posts?: boolean;
}

export async function createCommunity(data: CreateCommunityRequest): Promise<Community> {
	return apiRequest('POST', '/communities', data, true);
}

export async function getCommunities(page: number = 1, limit: number = 10, query?: string): Promise<{ communities: Community[]; total: number; page: number; limit: number }> {
	let url = `/communities?page=${page}&limit=${limit}`;
	if (query) {
		url += `&q=${encodeURIComponent(query)}`;
	}
	return apiRequest('GET', url, undefined, true);
}

export async function getUserCommunities(userId: string = 'me'): Promise<Community[]> {
	return apiRequest('GET', `/communities/user/${userId}`, undefined, true);
}

export async function getCommunity(id: string): Promise<Community> {
	return apiRequest('GET', `/communities/${id}`, undefined, true);
}

export async function updateCommunitySettings(id: string, data: UpdateCommunityRequest): Promise<void> {
	return apiRequest('PUT', `/communities/${id}/settings`, data, true);
}

export async function joinCommunity(id: string): Promise<void> {
	return apiRequest('POST', `/communities/${id}/join`, undefined, true);
}

export async function leaveCommunity(id: string): Promise<void> {
	return apiRequest('POST', `/communities/${id}/leave`, undefined, true);
}

export async function approveCommunityMember(communityId: string, userId: string): Promise<void> {
	return apiRequest('POST', `/communities/${communityId}/approve`, { user_id: userId }, true);
}

export async function rejectCommunityMember(communityId: string, userId: string): Promise<void> {
	return apiRequest('POST', `/communities/${communityId}/reject`, { user_id: userId }, true);
}

export async function getCommunityMembers(communityId: string, page = 1, limit = 10): Promise<{ users: import('./types').User[], total: number }> {
	return apiRequest('GET', `/communities/${communityId}/members?page=${page}&limit=${limit}`, undefined, true);
}

export async function getCommunityAdmins(communityId: string): Promise<import('./types').User[]> {
	return apiRequest('GET', `/communities/${communityId}/admins`, undefined, true);
}

export async function getCommunityPendingMembers(communityId: string, page = 1, limit = 10): Promise<{ users: import('./types').User[], total: number }> {
	return apiRequest('GET', `/communities/${communityId}/pending-members?page=${page}&limit=${limit}`, undefined, true);
}


// Feed API
export async function getPosts(params: { page?: number; limit?: number; community_id?: string; user_id?: string; status?: string; sort?: string }): Promise<import('./types').FeedResponse> {
	const query = new URLSearchParams();
	if (params.page) query.set('page', String(params.page));
	if (params.limit) query.set('limit', String(params.limit));
	if (params.community_id) query.set('community_id', params.community_id);
	if (params.user_id) query.set('user_id', params.user_id);
	if (params.status) query.set('status', params.status);
	if (params.sort) query.set('sort', params.sort);
	return apiRequest('GET', `/posts?${query.toString()}`, undefined, true);
}

export async function createPost(data: FormData | any): Promise<import('./types').Post> {
	return apiRequest('POST', '/posts', data, true);
}

export async function updatePostStatus(postId: string, status: 'active' | 'pending' | 'declined'): Promise<void> {
	return apiRequest('PUT', `/posts/${postId}/status`, { status }, true);
}

// --- Event Types & APIs ---

export type EventPrivacy = 'public' | 'private' | 'friends';
export type RSVPStatus = 'going' | 'interested' | 'invited' | 'not_going';
export type EventInvitationStatus = 'pending' | 'accepted' | 'declined';

export interface EventStats {
	going_count: number;
	interested_count: number;
	invited_count: number;
	share_count: number;
}

export interface Event {
	id: string;
	title: string;
	description: string;
	start_date: string; // ISO Date
	end_date?: string; // ISO Date
	location: string;
	is_online: boolean;
	privacy: EventPrivacy;
	category: string;
	cover_image: string;
	creator: UserShortResponse;
	stats: EventStats;
	my_status?: RSVPStatus;
	is_host: boolean;
	friends_going?: UserShortResponse[]; // Friends who are going to this event
	created_at: string;
}

export interface CreateEventRequest {
	title: string;
	description: string;
	start_date: string;
	end_date?: string;
	location?: string;
	is_online: boolean;
	privacy: EventPrivacy;
	category?: string;
	cover_image?: string;
}

export interface UpdateEventRequest {
	title?: string;
	description?: string;
	start_date?: string;
	end_date?: string;
	location?: string;
	is_online?: boolean;
	privacy?: EventPrivacy;
	category?: string;
	cover_image?: string;
}

// Event Invitation Types
export interface EventShort {
	id: string;
	title: string;
	cover_image: string;
	start_date: string;
	location: string;
}

export interface EventInvitation {
	id: string;
	event: EventShort;
	inviter: UserShortResponse;
	status: EventInvitationStatus;
	message?: string;
	created_at: string;
}

// Event Discussion/Post Types
export interface EventPostReaction {
	user: UserShortResponse;
	emoji: string;
	timestamp: string;
}

export interface EventPost {
	id: string;
	author: UserShortResponse;
	content: string;
	media_urls?: string[];
	reactions: EventPostReaction[];
	created_at: string;
}

// Event Attendee Types
export interface EventAttendee {
	user: UserShortResponse;
	status: RSVPStatus;
	timestamp: string;
	is_host: boolean;
	is_co_host: boolean;
}

export interface AttendeesListResponse {
	attendees: EventAttendee[];
	total: number;
	page: number;
	limit: number;
}

// Event Category Types
export interface EventCategory {
	name: string;
	icon?: string;
	count: number;
}

// Birthday Types
export interface BirthdayUser {
	id: string;
	username: string;
	full_name: string;
	avatar: string;
	age: number;
	date: string;
}

export interface BirthdayResponse {
	today: BirthdayUser[];
	upcoming: BirthdayUser[];
}

// Search Types
export interface SearchEventsParams {
	q?: string;
	category?: string;
	period?: 'today' | 'tomorrow' | 'this_week' | 'this_weekend';
	start_date?: string;
	end_date?: string;
	online?: boolean;
	page?: number;
	limit?: number;
}

// ============ Event APIs ============

export async function createEvent(data: CreateEventRequest): Promise<Event> {
	return apiRequest('POST', '/events', data);
}

export async function getEvents(page = 1, limit = 10, params?: { category?: string; period?: string; q?: string }): Promise<{ events: Event[]; total: number }> {
	const query = new URLSearchParams({ page: String(page), limit: String(limit) });
	if (params?.category) query.set('category', params.category);
	if (params?.period) query.set('period', params.period);
	if (params?.q) query.set('q', params.q);
	return apiRequest('GET', `/events?${query.toString()}`);
}

export async function getMyEvents(page = 1, limit = 10): Promise<Event[]> {
	return apiRequest('GET', `/events/my-events?page=${page}&limit=${limit}`);
}

export async function getBirthdays(): Promise<BirthdayResponse> {
	return apiRequest('GET', '/events/birthdays');
}

export async function getEvent(id: string): Promise<Event> {
	return apiRequest('GET', `/events/${id}`);
}

export async function updateEvent(id: string, data: UpdateEventRequest): Promise<Event> {
	return apiRequest('PUT', `/events/${id}`, data);
}

export async function deleteEvent(id: string): Promise<void> {
	return apiRequest('DELETE', `/events/${id}`);
}

export async function rsvpEvent(id: string, status: RSVPStatus): Promise<void> {
	return apiRequest('POST', `/events/${id}/rsvp`, { status });
}

// Search & Discovery
export async function searchEvents(params: SearchEventsParams): Promise<{ events: Event[]; total: number; page: number; limit: number }> {
	const query = new URLSearchParams();
	if (params.q) query.set('q', params.q);
	if (params.category) query.set('category', params.category);
	if (params.period) query.set('period', params.period);
	if (params.start_date) query.set('start_date', params.start_date);
	if (params.end_date) query.set('end_date', params.end_date);
	if (params.online !== undefined) query.set('online', String(params.online));
	query.set('page', String(params.page || 1));
	query.set('limit', String(params.limit || 20));
	return apiRequest('GET', `/events/search?${query.toString()}`);
}

export async function getNearbyEvents(lat: number, lng: number, radius = 50, page = 1, limit = 20): Promise<{ events: Event[]; total: number }> {
	return apiRequest('GET', `/events/nearby?lat=${lat}&lng=${lng}&radius=${radius}&page=${page}&limit=${limit}`);
}

export async function getEventCategories(): Promise<{ categories: EventCategory[] }> {
	return apiRequest('GET', '/events/categories');
}

// Invitations
export async function inviteFriendsToEvent(eventId: string, friendIds: string[], message?: string): Promise<{ success: boolean }> {
	return apiRequest('POST', `/events/${eventId}/invite`, { friend_ids: friendIds, message });
}

export async function getEventInvitations(page = 1, limit = 10): Promise<{ invitations: EventInvitation[]; total: number }> {
	return apiRequest('GET', `/events/invitations?page=${page}&limit=${limit}`);
}

export async function respondToEventInvitation(invitationId: string, accept: boolean): Promise<{ success: boolean }> {
	return apiRequest('POST', `/events/invitations/${invitationId}/respond`, { accept });
}

// Attendees
export async function getEventAttendees(eventId: string, status?: RSVPStatus, page = 1, limit = 20): Promise<AttendeesListResponse> {
	const query = new URLSearchParams({ page: String(page), limit: String(limit) });
	if (status) query.set('status', status);
	return apiRequest('GET', `/events/${eventId}/attendees?${query.toString()}`);
}

// Discussion Posts
export async function createEventPost(eventId: string, content: string, mediaUrls?: string[]): Promise<EventPost> {
	return apiRequest('POST', `/events/${eventId}/posts`, { content, media_urls: mediaUrls });
}

export async function getEventPosts(eventId: string, page = 1, limit = 20): Promise<{ posts: EventPost[]; total: number }> {
	return apiRequest('GET', `/events/${eventId}/posts?page=${page}&limit=${limit}`);
}

export async function deleteEventPost(eventId: string, postId: string): Promise<void> {
	return apiRequest('DELETE', `/events/${eventId}/posts/${postId}`);
}

export async function reactToEventPost(eventId: string, postId: string, emoji: string): Promise<{ success: boolean }> {
	return apiRequest('POST', `/events/${eventId}/posts/${postId}/react`, { emoji });
}

// Co-hosts
export async function addEventCoHost(eventId: string, userId: string): Promise<{ success: boolean }> {
	return apiRequest('POST', `/events/${eventId}/co-hosts`, { user_id: userId });
}

export async function removeEventCoHost(eventId: string, userId: string): Promise<void> {
	return apiRequest('DELETE', `/events/${eventId}/co-hosts/${userId}`);
}

// Sharing
export async function shareEvent(eventId: string): Promise<{ success: boolean }> {
	return apiRequest('POST', `/events/${eventId}/share`);
}


// --- Album API Functions ---

export async function createAlbum(data: CreateAlbumRequest): Promise<Album> {
	return apiRequest('POST', '/albums', data);
}

export async function getAlbum(albumId: string): Promise<Album> {
	return apiRequest('GET', `/albums/${albumId}`);
}

export async function getUserAlbums(userId: string, limit: number = 20, offset: number = 0): Promise<Album[]> {
	return apiRequest('GET', `/users/${userId}/albums?limit=${limit}&offset=${offset}`);
}

export async function updateAlbum(albumId: string, data: UpdateAlbumRequest): Promise<Album> {
	return apiRequest('PUT', `/albums/${albumId}`, data);
}

export async function addMediaToAlbum(albumId: string, media: { url: string; type: string }[]): Promise<SuccessResponse> {
	return apiRequest('POST', `/albums/${albumId}/media`, { media });
}

export async function getAlbumMedia(albumId: string, limit: number = 20, page: number = 1, type?: string): Promise<{ media: any[]; total: number; page: number; limit: number }> {
	const params = new URLSearchParams({ limit: String(limit), page: String(page) });
	if (type) params.append('type', type);
	return apiRequest('GET', `/albums/${albumId}/media?${params.toString()}`);
}

// Event Recommendations & Trending
export interface EventRecommendation {
	event_id: string;
	score: number;
	event?: Event;
	friends_going: import('./types').User[];
	friend_count: number;
	reason: string;
}

export interface TrendingEvent {
	event_id: string;
	score: number;
	event?: Event;
}

export async function getEventRecommendations(limit: number = 10): Promise<EventRecommendation[]> {
	return apiRequest('GET', `/events/recommendations?limit=${limit}`);
}

export async function getTrendingEvents(limit: number = 10): Promise<TrendingEvent[]> {
	return apiRequest('GET', `/events/trending?limit=${limit}`);
}
