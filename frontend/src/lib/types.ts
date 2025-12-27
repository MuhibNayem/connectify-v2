export interface Reaction {
	user_id: string;
	emoji: string;
	timestamp: string;
}

export interface Message {
	id: string;
	string_id?: string; // Mapped from Cassandra UUID
	_legacy_id?: string; // Original Mongo ObjectID for legacy WS events
	sender_id: string;
	sender_name?: string;
	receiver_id?: string;
	group_id?: string;
	group_name?: string;
	content: string;
	content_type: string;
	media_urls?: string[];
	seen_by?: string[];
	delivered_to?: string[];
	_optimistic_files?: File[];
	is_deleted: boolean;
	deleted_at?: string;
	is_edited?: boolean; // Added
	edited_at?: string; // Added
	reactions?: Reaction[];
	reply_to_message_id?: string; // Added
	product_id?: string; // Added for marketplace
	is_marketplace?: boolean; // Marketplace context flag
	// Embedded product data (populated by backend $lookup for optimization)
	product?: {
		id: string;
		title: string;
		price: number;
		currency: string;
		images?: string[];
		status: string;
	};
	created_at: string;
	updated_at?: string;
	// E2EE
	is_encrypted?: boolean;
	iv?: string;
	_is_decrypted?: boolean; // Client-side flag
	// Potentially add sender/receiver/group objects if populated by backend
	sender?: {
		id: string;
		username: string;
		avatar?: string;
		full_name?: string;
	};
}

export interface WebSocketEvent {
	type: string;
	data: any;
}

export interface ReactionEvent {
	message_id: string;
	user_id: string;
	emoji: string;
	action: 'add' | 'remove';
	timestamp: string;
}

export interface ReadReceiptEvent {
	message_ids: string[];
	reader_id: string;
	timestamp: string;
}

export interface MessageEditedEvent {
	message_id: string;
	editor_id: string;
	new_content: string;
	edited_at: string;
}

export interface MessageCreatedEvent extends Message { }

export interface User {
	id: string;
	username: string;
	email: string;
	avatar?: string;
	cover_picture?: string;
	full_name?: string;
	bio?: string;
	location?: string;
	phone_number?: string;
	date_of_birth?: string;
	gender?: string;
	privacy_settings?: PrivacySettings;
	notification_settings?: NotificationSettings;
	created_at?: string;
	// E2EE
	public_key?: string;
	encrypted_private_key?: string;
	key_backup_iv?: string;
	key_backup_salt?: string;
	is_encryption_enabled?: boolean;
}

export interface PrivacySettings {
	default_post_privacy: string;
	can_see_my_friends_list: string;
	can_send_me_friend_requests: string;
	can_tag_me_in_posts: string;
}

export interface NotificationSettings {
	email_notifications: boolean;
	push_notifications: boolean;
	notify_on_friend_request: boolean;
	notify_on_comment: boolean;
	notify_on_like: boolean;
	notify_on_tag: boolean;
	notify_on_message: boolean;
}

export interface PostAuthor {
	id: string;
	username: string;
	avatar?: string;
	full_name?: string;
}

export interface Post {
	id: string;
	user_id: string;
	author: PostAuthor;
	content: string;
	media?: { url: string; type: string }[];
	location?: string;
	privacy: string;
	comments: any[]; // You might want to define Comment type too
	mentions: string[];
	mentioned_users?: { id: string; username: string }[];
	specific_reaction_counts: { [key: string]: number };
	hashtags: string[];
	created_at: string;
	updated_at: string;
	total_reactions: number;
	total_comments: number;
	community_id?: string;
	status?: 'active' | 'pending' | 'declined';
}

export interface FeedResponse {
	posts: Post[];
	total: number;
	page: number;
	limit: number;
}
