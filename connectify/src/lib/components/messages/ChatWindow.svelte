<!--
This is the main chat window component.
It orchestrates the display of messages and the message input field.
-->
<script lang="ts">
	import {
		getMessages,
		sendMessage,
		markMessagesAsDelivered,
		markMessagesAsSeen,
		getConversationSummaries,
		type ConversationSummary,
		markConversationAsSeen,
		updateUserKeys,
		updateUserProfile,
		type UserKeys
	} from '$lib/api';
	import type { Message as MessageModel } from '$lib/types';
	import { formatDistanceToNow } from 'date-fns';
	import { tick, onMount } from 'svelte';

	import Message from '$lib/components/messages/Message.svelte';
	import Lightbox from '$lib/components/messages/Lightbox.svelte';
	import MessageInput from './MessageInput.svelte';
	import { websocketMessages } from '$lib/websocket';
	import { auth } from '$lib/stores/auth.svelte';
	import { presenceStore, type PresenceState } from '$lib/stores/presence';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { trackVisibility } from '$lib/actions/trackVisibility';
	import GroupInfoModal from './GroupInfoModal.svelte';
	import * as crypto from '$lib/crypto';
	import * as keyStore from '$lib/key_store';
	import { voiceCallService } from '$lib/stores/voice-call.svelte';
	import { Phone, Video } from '@lucide/svelte';

	let showGroupInfo = $state(false);

	import { getProduct } from '$lib/api/marketplace'; // Import getProduct if needed or assume passed product object

	let {
		conversationId,
		initialProduct = null,
		initialMessage = '',
		isMarketplace = false,
		onProductClick = undefined,
		onMessageSent = undefined
	} = $props<{
		conversationId: string;
		initialProduct?: any; // Product object
		initialMessage?: string;
		isMarketplace?: boolean; // If true, only fetch marketplace messages
		onProductClick?: (productId: string) => void; // Callback when product is clicked in chat
		onMessageSent?: () => void; // Callback when a message is sent
	}>();

	let conversationType = $derived(conversationId.split('-')[0]);
	let currentChatId = $derived(conversationId.split('-')[1]);
	function deriveConversationKey() {
		if (!conversationId) return '';
		if (conversationId.startsWith('dm_') || conversationId.startsWith('group_')) {
			return conversationId;
		}
		const [type, id] = conversationId.split('-');
		if (type === 'group' && id) {
			return `group_${id}`;
		}
		if (type === 'user' && id && auth.state.user?.id) {
			const myId = auth.state.user.id;
			return myId < id ? `dm_${myId}_${id}` : `dm_${id}_${myId}`;
		}
		return '';
	}
	let conversationKey = $derived(deriveConversationKey());

	let messages = $state<any[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let chatContainer: HTMLElement;
	let isOpponentTyping = $state(false);
	let typingTimeout: ReturnType<typeof setTimeout>;
	let isSending = $state(false);

	// Pre-fill State
	let pendingProduct = $state<any>(initialProduct);
	let messageInputDraft = $state(initialMessage);

	// Reset pending state when conversation changes, unless it matches the initial prop intent
	$effect(() => {
		if (conversationId && initialProduct) {
			// If we switched to a new conversation with an intent to send a product
			pendingProduct = initialProduct;
			messageInputDraft = initialMessage;
		} else if (conversationId) {
			pendingProduct = null;
			messageInputDraft = '';
		}
	});

	// Pagination State
	let page = $state(1);
	let limit = $state(50);
	let hasMore = $state(true);
	let isFetchingMore = $state(false);
	let initialLoadComplete = $state(false); // To differntiate initial load vs pagination

	// Marketplace: Track the product_id from the conversation to persist it for all messages
	let marketplaceProductId = $state<string | null>(null);

	let conversationPartner = $state<ConversationSummary | null>(null);
	let presenceState = $state<PresenceState>({});

	// Subscribe to presence updates
	$effect(() => {
		const unsubscribe = presenceStore.subscribe((val) => {
			presenceState = val;
		});
		return unsubscribe;
	});

	// E2EE State
	let isEncrypted = $state(false);
	let myKeyPair = $state<crypto.KeyPair | null>(null);
	let partnerPublicKey = $state<CryptoKey | null>(null);
	let myFingerprint = $state<string>('');
	let partnerFingerprint = $state<string>('');
	let isSettingUpKeys = $state(false);
	let showRecoverySetup = $state(false);
	let recoveryPassword = $state('');

	// Declared here to be accessible
	let showSecurityInfo = $state(false);

	// Queue for messages that have been rendered but not yet marked as delivered
	let deliveredQueue = new Set<string>();

	function getTransportMessageId(message: MessageModel): string | undefined {
		return message?.string_id || message?.id || message?._legacy_id;
	}

	function normalizeIncomingMessage(raw: any): MessageModel {
		if (!raw) return raw;
		const legacyId = raw.id;
		return {
			...raw,
			id: raw.string_id || legacyId,
			_legacy_id: legacyId,
			seen_by: raw.seen_by || [],
			delivered_to: raw.delivered_to || []
		};
	}

	function candidateMessageIds(message: MessageModel): string[] {
		const ids = [message.id, message.string_id, message._legacy_id].filter(Boolean) as string[];
		return Array.from(new Set(ids));
	}

	function messageMatchesIdentifier(message: MessageModel, identifier?: string): boolean {
		if (!identifier) return false;
		return candidateMessageIds(message).includes(identifier);
	}

	function eventMatchesMessage(message: MessageModel, ids?: string[]): boolean {
		if (!ids || ids.length === 0) return false;
		const candidates = candidateMessageIds(message);
		return ids.some((id) => candidates.includes(id));
	}

	// Restore Keys Logic

	// E2EE Functions
	async function initE2EE() {
		// 1. Load my private key
		const priv = await keyStore.loadPrivateKey();
		const pub = await keyStore.loadPublicKey();
		if (priv && pub) {
			myKeyPair = { privateKey: priv, publicKey: pub };
			// Compute my fingerprint
			const pubExp = await crypto.exportPublicKey(pub);
			myFingerprint = await crypto.computeFingerprint(pubExp);
		}
	}

	async function setupKeys() {
		if (!recoveryPassword) {
			alert('Please enter a recovery password to secure your keys.');
			return;
		}
		isSettingUpKeys = true;
		try {
			// 1. Generate
			const keys = await crypto.generateKeyPair();

			// 2. Encrypt Private Key for Backup
			const backup = await crypto.encryptPrivateKeyWithPassword(keys.privateKey, recoveryPassword);

			// 3. Export Public Key
			const pubKeyBase64 = await crypto.exportPublicKey(keys.publicKey);

			// 4. Upload to Server
			const payload: UserKeys = {
				public_key: pubKeyBase64,
				encrypted_private_key: backup.encryptedPrivateKey,
				key_backup_iv: backup.iv,
				key_backup_salt: backup.salt
			};
			await updateUserKeys(payload);

			// 5. Save Locally
			await keyStore.savePrivateKey(keys.privateKey);
			await keyStore.savePublicKey(keys.publicKey);

			myKeyPair = keys;
			myFingerprint = await crypto.computeFingerprint(pubKeyBase64);
			showRecoverySetup = false;
			alert('Encryption enabled! Keys generated and backed up securely.');
		} catch (e) {
			console.error('Failed to setup keys:', e);
			alert('Failed to setup encryption keys.');
		} finally {
			isSettingUpKeys = false;
		}
	}

	// Restore Keys Logic
	let showRestoreModal = $state(false);
	let isRestoringKeys = $state(false);

	async function restoreKeys() {
		if (!recoveryPassword) {
			alert('Please enter your recovery password.');
			return;
		}
		isRestoringKeys = true;
		try {
			// Get fresh user profile to ensure we have backup data
			// We can use auth.state.user but to be safe lets assume it's there or refresh it
			// Ideally we assume auth state is up to date or we fetch "me"
			const { updateUserProfile } = await import('$lib/api'); // Just ensuring we have API access
			// Actually we need 'getUserProfile' but that updates 'auth.state.user' usually.
			// Let's use auth.state.user directly.
			const user = auth.state.user;

			if (!user?.encrypted_private_key || !user?.key_backup_iv || !user?.key_backup_salt) {
				alert('No backup found on server. Cannot restore.');
				return;
			}

			const privateKey = await crypto.decryptPrivateKeyWithPassword(
				user.encrypted_private_key,
				user.key_backup_iv,
				user.key_backup_salt,
				recoveryPassword
			);

			// If success (no error thrown), import public key too
			// We can import public key from user.public_key string
			if (!user.public_key) throw new Error('Public Key missing from profile');
			const publicKey = await crypto.importPublicKey(user.public_key);

			// Save Locally
			await keyStore.savePrivateKey(privateKey);
			await keyStore.savePublicKey(publicKey);

			myKeyPair = { privateKey, publicKey };
			myFingerprint = await crypto.computeFingerprint(user.public_key);
			showRestoreModal = false;
			alert('Keys restored successfully! Encryption enabled.');
			isEncrypted = true;
		} catch (e) {
			console.error('Failed to restore keys:', e);
			alert('Failed to restore keys. Incorrect password or corrupted backup.');
		} finally {
			isRestoringKeys = false;
		}
	}

	// Key Rotation Logic
	let showRotateModal = $state(false);
	let isRotatingKeys = $state(false);
	let rotationPassword = $state('');

	async function rotateKeys() {
		if (!rotationPassword) {
			alert('Please enter a new recovery password to secure your rotated keys.');
			return;
		}
		isRotatingKeys = true;
		try {
			// 1. Generate NEW Key Pair
			const newKeys = await crypto.generateKeyPair();

			// 2. Encrypt NEW Private Key with the new password
			const backup = await crypto.encryptPrivateKeyWithPassword(
				newKeys.privateKey,
				rotationPassword
			);

			// 3. Export NEW Public Key
			const pubKeyBase64 = await crypto.exportPublicKey(newKeys.publicKey);

			// 4. Upload to Server (replaces old keys)
			const payload: UserKeys = {
				public_key: pubKeyBase64,
				encrypted_private_key: backup.encryptedPrivateKey,
				key_backup_iv: backup.iv,
				key_backup_salt: backup.salt
			};
			await updateUserKeys(payload);

			// 5. Clear OLD local keys and save NEW ones
			await keyStore.clearKeys();
			await keyStore.savePrivateKey(newKeys.privateKey);
			await keyStore.savePublicKey(newKeys.publicKey);

			// 6. Update state
			myKeyPair = newKeys;
			myFingerprint = await crypto.computeFingerprint(pubKeyBase64);
			rotationPassword = '';
			showRotateModal = false;
			alert(
				"Keys rotated successfully! Your previous encrypted messages may become unreadable if you don't have the old keys."
			);
		} catch (e) {
			console.error('Failed to rotate keys:', e);
			alert('Failed to rotate keys.');
		} finally {
			isRotatingKeys = false;
		}
	}

	async function toggleEncryption() {
		if (isEncrypted) {
			// Instead of disabling immediately, show security details
			showSecurityInfo = true;
			return;
		}

		// Enabling Encryption
		if (!myKeyPair) {
			// Check if we have a backup on the server to restore
			const user = auth.state.user;
			// Reload user to be sure?
			// For now, assume auth.state.user is recent enough.
			// Check if encrypted_private_key exists
			if (user && user.encrypted_private_key) {
				showRestoreModal = true;
			} else {
				showRecoverySetup = true; // Trigger setup flow
			}
			return;
		}

		// Check if partner has keys (For DMs)
		// We need to fetch partner's public key from the 'conversationPartner' or fetch user profile again
		// Ideally 'getConversationSummaries' or 'getUser' should return public_key.
		// For now, let's assume we can try to fetch it or fail.
		// NOTE: API needs to return public_key for the user.
		// We will assume conversationPartner MIGHT have it if we fetch profile.
		// Actually, let's lazy load it here.
		// For MVP, we'll try to enable it, but if we can't find partner key, we warn.
		// For MVP, we'll try to enable it, but if we can't find partner key, we warn.
		isEncrypted = true;
	}

	// Auto-enable encryption if keys are available, and user hasn't globally disabled it
	$effect(() => {
		const globalEncryptionEnabled = auth.state.user?.is_encryption_enabled !== false; // Default to true if undefined
		if (
			myKeyPair &&
			partnerPublicKey &&
			!isEncrypted &&
			conversationType === 'user' &&
			globalEncryptionEnabled
		) {
			console.log('[E2EE] Auto-enabling encryption (Global setting enabled)');
			isEncrypted = true;
		}
	});

	// Auto-prompt to restore keys if missing locally but exist on server
	$effect(() => {
		// Only run if we are in a user chat (where E2EE matters) logic could be general though.
		const user = auth.state.user;
		// If we do NOT have local keys...
		if (!myKeyPair && !isSettingUpKeys && !isRestoringKeys) {
			// And we DO have a backup on server...
			if (user?.encrypted_private_key && user?.key_backup_iv) {
				// And we haven't already shown it (maybe track with a flag if user dismissed it?)
				// For now, if messages are failing to decrypt, it helps to show it.
				// Let's rely on user action or specific conditions.
				// Showing it immediately on load might be intrusive if they just want to read plain texts.
				// BUT if they have encrypted messages in the list, they can't see them.

				// Let's check if there are encrypted messages that we can't read?
				const hasEncryptedMessages = messages.some((m) => m.is_encrypted && !m._is_decrypted);
				if (hasEncryptedMessages) {
					// Debounce or ensure we don't spam
					if (!showRestoreModal && !showRecoverySetup) {
						// console.log('[E2EE] Prompting restore due to encrypted messages...');
						// showRestoreModal = true;
						// Commented out auto-show to avoid annoyance loop. relying on the "Unlock keys" button or banner?
						// Better: Show a banner in the chat header?
					}
				}
			}
		}
	});

	// Helper to decrypt a single message
	// Helper to decrypt a single message
	async function decryptMessageContent(msg: any): Promise<string> {
		if (!msg.is_encrypted || !msg.iv) return msg.content;
		if (!myKeyPair) return '[Encrypted] (Unlock keys to read)';
		if (!partnerPublicKey) return '[Encrypted] (Missing partner key)';

		try {
			// Check if message mentions a specific key fingerprint?
			// Current protocol doesn't attach key ID to message, so we assume current key.
			// If decryption fails, it's likely a key mismatch (old message vs new key).

			const sharedKey = await crypto.deriveSharedSecret(myKeyPair.privateKey, partnerPublicKey);
			const decrypted = await crypto.decryptMessage(msg.content, msg.iv, sharedKey);

			if (decrypted.startsWith('[Decryption Error]')) {
				return '[Encrypted] (Key mismatch - Old message?)';
			}
			return decrypted;
		} catch (e) {
			console.error('Decryption failed:', e);
			return '[Encrypted] (Decryption failed)';
		}
	}

	// Reactive effect to decrypt messages
	// We use a map to track decrypted state to avoid infinite loops or re-decrypting
	let decryptedCache = new Set<string>();
	let isDecrypting = false;

	$effect(() => {
		if (conversationId) {
			decryptedCache.clear();
			isDecrypting = false; // Reset lock on conversation change
		}
	});

	$effect(() => {
		// Explicitly depend on these reactive values
		const hasKeys = myKeyPair !== null;
		const hasPartnerKey = partnerPublicKey !== null;
		const messageCount = messages.length;

		if (messageCount > 0 && hasKeys && hasPartnerKey && !isDecrypting) {
			console.log(
				'[E2EE] Running decryption for',
				messageCount,
				'messages | Keys:',
				!!hasKeys,
				'Partner:',
				!!hasPartnerKey
			);

			// Use Promise.all for better async handling
			const decryptAll = async () => {
				if (isDecrypting) return;
				isDecrypting = true;

				try {
					// Loop to process all pending decryption tasks, handling incoming updates (like pagination)
					// efficiently without race conditions.
					while (true) {
						// Check for work using the latest 'messages' state
						const hasWork = messages.some(
							(msg) => msg.is_encrypted && !msg._is_decrypted && !decryptedCache.has(msg.id)
						);

						if (!hasWork) break;

						// Optimization: Derive shared key ONCE for the whole batch
						// We assume key doesn't change mid-loop, or if it does, effect re-trigger is blocked anyway,
						// but using current keys for current batch is correct.
						const sharedKey = await crypto.deriveSharedSecret(
							myKeyPair!.privateKey,
							partnerPublicKey!
						);

						// Identify messages needing decryption and prepare jobs
						const jobs: Promise<{ id: string; content: string }>[] = [];

						for (let i = 0; i < messages.length; i++) {
							const msg = messages[i];
							if (msg.is_encrypted && !msg._is_decrypted && !decryptedCache.has(msg.id)) {
								decryptedCache.add(msg.id);
								jobs.push(
									crypto
										.decryptMessage(msg.content, msg.iv, sharedKey)
										.then((plaintext) => {
											if (plaintext.startsWith('[Decryption Error]')) {
												return { id: msg.id, content: '[Encrypted] (Key mismatch)' };
											}
											return { id: msg.id, content: plaintext };
										})
										.catch((e) => {
											console.error('[E2EE] Decryption failed for msg', msg.id, e);
											return { id: msg.id, content: '[Encrypted] (Decryption Error)' };
										})
								);
							}
						}

						if (jobs.length > 0) {
							// Wait for all to finish (Parallel execution)
							const results = await Promise.all(jobs);

							// Apply updates in a batch
							const updatedMessages = [...messages];
							let hasUpdates = false;

							results.forEach((res) => {
								const index = updatedMessages.findIndex((m) => m.id === res.id);
								if (index !== -1) {
									const msg = updatedMessages[index];
									updatedMessages[index] = { ...msg, content: res.content, _is_decrypted: true };
									hasUpdates = true;
								}
							});

							if (hasUpdates) {
								messages = updatedMessages;
								console.log('[E2EE] Batch Decrypted:', results.length);
							}
						}
					}
				} finally {
					isDecrypting = false;
				}
			};
			decryptAll();
		}
	});

	let deliveredDebounceTimer: ReturnType<typeof setTimeout>;

	function handleMessageRendered(event: CustomEvent<{ messageId: string }>) {
		const messageId = event.detail.messageId;
		// Only mark as delivered if the current user is the receiver and not the sender
		const message = messages.find((m) => m.id === messageId);
		if (message && message.sender_id !== auth.state.user?.id) {
			if (!conversationKey) {
				return;
			}
			const transportId = getTransportMessageId(message);
			if (!transportId) {
				return;
			}
			deliveredQueue.add(transportId);
			clearTimeout(deliveredDebounceTimer);
			deliveredDebounceTimer = setTimeout(async () => {
				if (deliveredQueue.size > 0) {
					try {
						await markMessagesAsDelivered(conversationKey, Array.from(deliveredQueue));
						console.log(
							'markMessagesAsDelivered API call successful for IDs:',
							Array.from(deliveredQueue)
						);
					} catch (error) {
						console.error('markMessagesAsDelivered API call failed:', error);
					}
					deliveredQueue.clear();
				}
			}, 500); // Debounce for 500ms
		}
	}

	$effect(() => {
		if (conversationId) {
			// Reset pagination state on conversation change
			page = 1;
			hasMore = true;
			messages = [];
			initialLoadComplete = false;
			marketplaceProductId = null; // Reset for new conversation
			fetchMessages(1);
			loadConversationPartner();
		}
	});

	async function loadConversationPartner() {
		try {
			const summaries = await getConversationSummaries();
			const [type, id] = conversationId.split('-');
			// Match by full conversation ID format (summaries have 'group-{id}' or 'user-{id}' format)
			const partner = summaries.find((s) => s.id === conversationId || s.id === id);
			if (partner) {
				conversationPartner = partner;
			} else if (type === 'user' && id) {
				// Partner not found in summaries (might be non-friend marketplace seller)
				// Fetch user info directly
				const { getUserByID } = await import('$lib/api');

				const user = await getUserByID(id);
				if (user) {
					conversationPartner = {
						id: user.id,
						name: user.username || user.email || 'User',
						avatar: user.avatar,
						is_group: false,
						unread_count: 0
					};
				}
			} else if (type === 'group' && id) {
				// Group not found in summaries, fetch group info directly
				const { apiRequest } = await import('$lib/api');
				try {
					const group = await apiRequest('GET', `/groups/${id}`);
					if (group) {
						conversationPartner = {
							id: id,
							name: group.name || 'Group',
							avatar: group.avatar,
							is_group: true,
							unread_count: 0
						};
					}
				} catch (e) {
					console.error('Failed to fetch group info:', e);
				}
			}

			// E2EE: Fetch Public Key if available
			if (type === 'user' && id) {
				const { getUserByID } = await import('$lib/api');

				const user = await getUserByID(id);

				// Only enable encryption if the user has keys AND has explicitly enabled it
				// If is_encryption_enabled is undefined, we assume true if keys exist (legacy compatibility)
				// or false? Let's assume true if keys exist but safe check if explicitly false.
				const isEnabled = user.is_encryption_enabled !== false;

				if (user.public_key) {
					// Always load key for decryption
					partnerPublicKey = await crypto.importPublicKey(user.public_key);
					partnerFingerprint = await crypto.computeFingerprint(user.public_key);
					console.log('Partner Public Key loaded.');

					// Check preference for SENDING
					if (!isEnabled && isEncrypted) {
						isEncrypted = false;
						console.log('Encryption disabled by partner preference');
					}
				} else {
					partnerPublicKey = null;
					if (isEncrypted) isEncrypted = false;
					console.log('Partner has no public key');
				}
			}
		} catch (e) {
			console.error('Failed to load conversation partner details:', e);
		}
	}

	async function fetchMessages(pageNum: number) {
		if (pageNum === 1) {
			isLoading = true;
			error = null;
		} else {
			isFetchingMore = true;
		}

		try {
			// Guard against undefined conversationId (e.g., empty inbox)
			if (!conversationId) {
				isLoading = false;
				return;
			}
			const [type, id] = conversationId.split('-');
			if (!type || !id) {
				isLoading = false;
				return;
			}
			let params: {
				receiverID?: string;
				groupID?: string;
				conversationID?: string;
				page: number;
				limit: number;
				marketplace?: boolean;
			} = {
				page: pageNum,
				limit: 50,
				marketplace: isMarketplace
			};

			if (id.startsWith('dm_')) {
				params.conversationID = id;
			} else if (type === 'group') {
				params.groupID = id;
			} else if (type === 'user') {
				params.receiverID = id;
			} else {
				throw new Error('Invalid conversation ID format.');
			}

			const response = await getMessages(params);
			if (response && Array.isArray(response.messages)) {
				// Ensure seen_by and delivered_to are always arrays
				const newMessages = response.messages.map((msg: any) => ({
					...msg,
					id: msg.string_id || msg.id, // Use Cassandra UUID if available
					seen_by: msg.seen_by || [],
					delivered_to: msg.delivered_to || []
				}));
				// .reverse(); // Backend now returns chronological (oldest first)

				if (newMessages.length < limit) {
					hasMore = false;
				}

				if (pageNum === 1) {
					messages = newMessages;
					initialLoadComplete = true;

					// In marketplace mode, extract product_id from messages to persist it
					if (isMarketplace && !marketplaceProductId) {
						const msgWithProduct = newMessages.find((m: any) => m.product_id);
						if (msgWithProduct) {
							marketplaceProductId = msgWithProduct.product_id;
						}
					}

					// Scroll to bottom on initial load
					tick().then(() => {
						if (chatContainer) chatContainer.scrollTop = chatContainer.scrollHeight;
					});
				} else {
					// Prepend older messages
					// distinct messages only to prevent key (id) collisions
					const existingIds = new Set(messages.map((m) => m.id));
					const uniqueNewMessages = newMessages.filter((m) => !existingIds.has(m.id));

					if (uniqueNewMessages.length > 0) {
						// In marketplace mode, also check paginated messages for product_id
						if (isMarketplace && !marketplaceProductId) {
							const msgWithProduct = uniqueNewMessages.find((m: any) => m.product_id);
							if (msgWithProduct) {
								marketplaceProductId = msgWithProduct.product_id;
							}
						}

						const previousScrollHeight = chatContainer.scrollHeight;
						messages = [...uniqueNewMessages, ...messages];

						// Maintain scroll position
						tick().then(() => {
							if (chatContainer) {
								const newScrollHeight = chatContainer.scrollHeight;
								chatContainer.scrollTop = newScrollHeight - previousScrollHeight;
							}
						});
					}
				}
			} else {
				if (pageNum === 1) messages = [];
				hasMore = false;
			}
		} catch (e: any) {
			console.error('Failed to load messages:', e);
			error = e.message || 'Failed to load messages.';
		} finally {
			if (pageNum === 1) {
				isLoading = false;
			} else {
				isFetchingMore = false;
			}
		}
	}

	function handleScroll() {
		if (!chatContainer) return;

		const { scrollTop } = chatContainer;

		// If scrolled to top (approx < 50px) and we have more messages and aren't currently fetching
		if (scrollTop < 50 && hasMore && !isFetchingMore && !isLoading && initialLoadComplete) {
			console.log('Load more messages triggering...');
			page += 1;
			fetchMessages(page);
		}
	}

	function startTypingTimeout() {
		clearTimeout(typingTimeout);
		typingTimeout = setTimeout(() => {
			isOpponentTyping = false;
		}, 3000); // Hide after 3 seconds of no new typing events
	}

	async function handleSendMessage(content: string, files: File[] = [], productId?: string) {
		if ((!content.trim() && files.length === 0) || isSending) return;

		isSending = true;

		const [type, id] = conversationId.split('-');
		const tempId = `temp-${Date.now()}`;
		const createdAt = new Date().toISOString();

		let encryptedContent = content;
		let iv = '';
		let isMsgEncrypted = false;

		// E2EE: Encrypt if enabled (Performed BEFORE optimistic update)
		if (isEncrypted && myKeyPair && partnerPublicKey && type === 'user') {
			try {
				const sharedKey = await crypto.deriveSharedSecret(myKeyPair.privateKey, partnerPublicKey);
				const encrypted = await crypto.encryptMessage(content, sharedKey);
				encryptedContent = encrypted.ciphertext;
				iv = encrypted.iv;
				isMsgEncrypted = true;
			} catch (e) {
				console.error('Encryption failed:', e);
				alert('Failed to encrypt message. Sending failed.');
				isSending = false;
				return;
			}
		}

		// Optimistically add message to Chat Window
		const optimisticMessage: MessageModel = {
			id: tempId,
			sender_id: auth.state.user?.id || '',
			sender_name: auth.state.user?.username || 'You',
			content: content, // Display plaintext locally
			content_type:
				files.length > 0
					? files.length > 1
						? 'multiple'
						: files[0].type.startsWith('image/')
							? 'image'
							: files[0].type.startsWith('video/')
								? 'video'
								: 'file'
					: 'text',
			media_urls: files.length > 0 ? files.map((f) => URL.createObjectURL(f)) : undefined,
			created_at: createdAt,
			is_deleted: false,
			is_edited: false,
			is_encrypted: isMsgEncrypted, // Set flag correctly
			iv: iv,
			_is_decrypted: isMsgEncrypted ? true : undefined, // It's our own message, so it's "decrypted"
			seen_by: [],
			delivered_to: [],
			_optimistic_files: files,
			...(type === 'user' && { receiver_id: id }),
			...(type === 'group' && { group_id: id })
		};

		// Immutable update to ensure reactivity
		messages = [...messages, optimisticMessage];

		// Optimistic Broadcast for ConversationList
		// We construct a payload that mimics the server response
		const optimisticEventPayload = {
			...optimisticMessage,
			// Ensure we send what ConversationList expects
			receiver_id: type === 'user' ? id : undefined,
			group_id: type === 'group' ? id : undefined
		};

		console.log('[ChatWindow] Emitting Optimistic MESSAGE_CREATED:', optimisticEventPayload);
		websocketMessages.set({
			type: 'MESSAGE_CREATED',
			data: optimisticEventPayload
		});

		try {
			let payload: any;

			if (files.length > 0) {
				const formData = new FormData();
				formData.append('content', encryptedContent);
				// Default content type if mixed, backend will refine
				formData.append('content_type', 'text');
				if (isMsgEncrypted) {
					formData.append('is_encrypted', 'true');
					formData.append('iv', iv);
				}

				if (type === 'group') {
					formData.append('group_id', id);
				} else {
					// Use raw ID from conversationId split (id is already without prefix)
					formData.append('receiver_id', id);
				}

				files.forEach((file) => {
					formData.append('files', file);
				});

				// Marketplace context: always send is_marketplace flag
				if (isMarketplace) {
					formData.append('is_marketplace', 'true');
				}

				payload = formData;
			} else {
				payload = {
					content: encryptedContent,
					content_type: 'text',
					is_encrypted: isMsgEncrypted,
					iv: iv
				};
				if (type === 'group') {
					payload['group_id'] = id;
				} else {
					// Use raw ID from conversationId split (id is already without prefix)
					payload['receiver_id'] = id;
				}

				// Marketplace context: always send is_marketplace flag
				if (isMarketplace) {
					payload['is_marketplace'] = true;
				}

				// Product attachment (optional metadata for product card display)
				if (productId) {
					payload['product_id'] = productId;
					payload['content_type'] = 'product';
				}
			}

			const serverMessage = await sendMessage(payload);

			// Server confirmed. We can optionally re-emit to update IDs,
			// but ConversationList usually cares about latest timestamp.
			// Updating ID in messages array:
			if (messages.some((m) => m.id === serverMessage.id)) {
				messages = messages.filter((msg) => msg.id !== tempId);
			} else {
				messages = messages.map((msg) =>
					msg.id === tempId
						? {
								...serverMessage,
								_is_decrypted: isMsgEncrypted ? true : undefined,
								content: content
							}
						: msg
				); // Keep plaintext content
			}

			// Notify parent that a message was sent (for conversation list refresh)
			if (onMessageSent) {
				onMessageSent();
			}

			// We DO NOT re-emit to websocketMessages here to avoid jumpiness,
			// unless we really want to confirm the ID.
			// Actually, real WS event will come too.
		} catch (e) {
			console.error('Send message failed:', e);
			messages = messages.filter((msg) => msg.id !== tempId);
		} finally {
			isSending = false;
		}
	}

	// Scroll to the bottom of the chat container when messages change
	// Scroll to the bottom of the chat container when messages change ONLY if we are near the bottom or it's a new message sent by us
	// For pagination updates (prepending), we handle scroll in fetchMessages manually.
	$effect(() => {
		if (chatContainer && messages && !isFetchingMore && initialLoadComplete) {
			// Basic auto-scroll logic for new incoming messages:
			// If we are at the bottom, stay at bottom.
			// If we sent the message (isSending or last message is ours), scroll to bottom.
			// implementation left simple here as requested for pagination task mainly.
			// Actually, the original code had:
			// chatContainer.scrollTop = chatContainer.scrollHeight;
			// We should preserve this behavior for new messages but NOT for pagination prepends.
			// The check '!isFetchingMore' partly handles this, but 'messages' dependency triggers on ANY change.
			// Let's refine: creating a snapshot before update is hard in $effect.
			// Instead, let's rely on fetchMessages handling the scroll for pagination, and this effect handling NEW messages (append).
			// Problem: $effect triggers on ALL message changes.
			// We only want to auto-scroll to bottom if:
			// 1. It's the initial load (handled in fetchMessages) -> Actually fetchMessages handles it now.
			// 2. A new message arrived (appended)
			// If we just prepended messages, isFetchingMore was true during the fetch.
			// utilize the tick() inside fetchMessages for prepending scroll fix.
			// prevent this effect from overriding the scroll fix when isFetchingMore was just true?
			// Svelte 5 effects track dependencies finely.
			// Simplest approach: Use a flag or check if last message changed?
			// For now, let's just disabling this rigorous auto-scroll for every message change
			// can avoid jumping to bottom when loading old history.
			// The simple rule: only auto-scroll if we are ALREADY at the bottom, OR if it's a fresh load (page 1).
		}
	});

	// We can rely on a simpler mechanic:
	// If a new message is added (length increased) and it's at the END, scroll to bottom.
	// If length increased but at START (pagination), maintain position.

	let lastMessageCount = 0;
	$effect(() => {
		if (messages.length > lastMessageCount) {
			const isPrepended = messages.length - lastMessageCount === limit; // rough heuristic or check IDs?
			// Check if the FIRST message ID changed vs LAST message ID changed?
			// Actually, `fetchMessages` handles the scroll for PREPENDING.
			// So we just need to handle the APPENDING case here (incoming websocket or sent message).

			// If we are not fetching more, assume it's an append.
			if (!isFetchingMore && initialLoadComplete) {
				// Check if user is near bottom or if it's their own message
				if (chatContainer) {
					const { scrollTop, scrollHeight, clientHeight } = chatContainer;
					const isNearBottom = scrollHeight - scrollTop - clientHeight < 100;
					const lastMsg = messages[messages.length - 1];
					const isMyMessage = lastMsg?.sender_id === auth.state.user?.id;

					if (isNearBottom || isMyMessage) {
						chatContainer.scrollTop = chatContainer.scrollHeight;
					}
				}
			}
		}
		lastMessageCount = messages.length;
	});

	// Handle real-time updates from WebSocket
	onMount(() => {
		const unsubscribe = websocketMessages.subscribe((event) => {
			if (!event) return;
			console.log(
				'[ChatWindow] WebSocket event received:',
				event.type,
				event.data,
				'Current Conversation:',
				conversationId
			);

			const [type, currentChatId] = conversationId.split('-');

			switch (event.type) {
				case 'MESSAGE_CREATED':
				case 'MARKETPLACE_MESSAGE_CREATED': {
					const newMessage = event.data;

					// Strict context check: Ensure message context matches window context
					const isMsgMarketplace = newMessage.is_marketplace === true;
					if (isMarketplace !== isMsgMarketplace) {
						// Mismatch: Ignore
						break;
					}

					console.log('MESSAGE_CREATED event received:', newMessage);

					// Check if the new message belongs to the current conversation
					let belongsToCurrentChat = false;
					if (type === 'group' && newMessage.group_id === currentChatId) {
						belongsToCurrentChat = true;
					} else if (
						type === 'user' &&
						(!newMessage.group_id || newMessage.group_id === '000000000000000000000000') && // Ensure it's not a group message (handle zero ID)
						(newMessage.receiver_id === currentChatId || newMessage.sender_id === currentChatId)
					) {
						belongsToCurrentChat = true;
					}
					console.log('Current conversation type:', type, 'ID:', currentChatId);
					console.log('Message belongs to current chat:', belongsToCurrentChat);

					if (belongsToCurrentChat) {
						// Check if message already exists in array (basic safety)
						const messageExists = messages.find((m) => m.id === newMessage.id);

						if (!messageExists) {
							// Immutable push
							messages = [...messages, newMessage];
						}
					}
					break;
				}
				case 'GROUP_UPDATED': {
					const updatedGroup = event.data;
					if (type === 'group' && currentChatId === updatedGroup.id && conversationPartner) {
						console.log('GROUP_UPDATED event received for current chat:', updatedGroup);
						conversationPartner = {
							...conversationPartner,
							name: updatedGroup.name,
							avatar: updatedGroup.avatar
						};
					}
					break;
				}
				case 'MESSAGE_DELETED':
				case 'MARKETPLACE_MESSAGE_DELETED': {
					const deletedMessage = event.data;
					messages = messages.map((m) => {
						if (m.id === deletedMessage.id) {
							return {
								...m,
								content: '[deleted]',
								content_type: 'deleted',
								is_deleted: true,
								media_urls: []
							};
						}
						return m;
					});
					break;
				}
				case 'MESSAGE_EDITED_UPDATE': {
					const { message_id, new_content } = event.data;
					console.log('MESSAGE_EDITED_UPDATE event received:', new_content);
					messages = messages.map((m) => {
						if (m.id === message_id) {
							return {
								...m,
								content: new_content,
								is_edited: true
							};
						}
						return m;
					});
					break;
				}
				case 'MESSAGE_REACTION_UPDATE': {
					const { message_id, user_id, emoji, action } = event.data;

					messages = messages.map((m) => {
						if (m.id === message_id) {
							const currentReactions = m.reactions || [];

							if (action === 'add') {
								// Check if already exists to avoid duplicates
								if (currentReactions.some((r: any) => r.user_id === user_id && r.emoji === emoji)) {
									return m;
								}
								return {
									...m,
									reactions: [
										...currentReactions,
										{ user_id, emoji, timestamp: new Date().toISOString() }
									]
								};
							} else if (action === 'remove') {
								return {
									...m,
									reactions: currentReactions.filter(
										(r: any) => !(r.user_id === user_id && r.emoji === emoji)
									)
								};
							}
						}
						return m;
					});
					break;
				}
				case 'CONVERSATION_SEEN_UPDATE': {
					const { conversation_id, user_id, timestamp, is_group } = event.data;
					const [type, id] = conversationId.split('-');

					// Check relevance: matched group ID OR matched user ID (for DMs)
					const isRelevant = id === conversation_id || (!is_group && id === user_id);

					if (isRelevant) {
						const seenTime = new Date(timestamp).getTime();
						messages = messages.map((msg) => {
							// Update if message is older/equal to seen timestamp AND not yet seen by this user
							if (new Date(msg.created_at).getTime() <= seenTime) {
								if (msg.seen_by?.includes(user_id)) return msg;

								return {
									...msg,
									seen_by: [...(msg.seen_by || []), user_id]
								};
							}
							return msg;
						});
					}
					break;
				}
				case 'MESSAGE_DELIVERED_UPDATE': {
					const { message_ids, deliverer_id } = event.data;
					messages = messages.map((msg) => {
						if (message_ids.includes(msg.id)) {
							if (msg.delivered_to?.includes(deliverer_id)) return msg;

							return {
								...msg,
								delivered_to: [...(msg.delivered_to || []), deliverer_id]
							};
						}
						return msg;
					});
					break;
				}
				case 'MESSAGE_READ_UPDATE': {
					const { message_ids, reader_id } = event.data;
					messages = messages.map((msg) => {
						if (message_ids.includes(msg.id)) {
							if (msg.seen_by?.includes(reader_id)) return msg;

							return {
								...msg,
								seen_by: [...(msg.seen_by || []), reader_id]
							};
						}
						return msg;
					});
					break;
				}
				case 'TYPING': {
					const {
						user_id,
						conversation_id,
						is_typing,
						is_marketplace: eventIsMarketplace
					} = event.data;
					// Only show typing indicator if the event is for the current conversation
					// and the typing user is not the current authenticated user
					// Also check marketplace context matches

					// Skip if marketplace context doesn't match
					const eventMarketplaceFlag = eventIsMarketplace || false;
					if (eventMarketplaceFlag !== isMarketplace) {
						break; // Ignore typing from different context (marketplace vs personal)
					}

					let isRelevant = false;
					if (type === 'group' && conversation_id === conversationId) {
						isRelevant = true;
					} else if (type === 'user' && user_id === currentChatId) {
						// For DMs, the conversation_id sent is 'MY_ID' (targeted at me).
						// We care if the SENDER (user_id) is the person we are currently looking at.
						isRelevant = true;
					}

					if (isRelevant && user_id !== auth.state.user?.id) {
						isOpponentTyping = is_typing;
						if (is_typing) {
							startTypingTimeout();
						} else {
							clearTimeout(typingTimeout);
						}
					}
					break;
				}
			}
		});

		return unsubscribe;
	});

	let seenDebounceTimer: any;
	let seenQueue = new Set<string>();

	function handleMessageVisible(message: any) {
		// Only mark messages as seen if they are NOT from the current user
		if (message.sender_id === auth.state.user?.id) return;
		if (!conversationKey) return;

		const messageIdentifier = getTransportMessageId(message);
		if (!messageIdentifier) return;

		// Add to queue
		seenQueue.add(messageIdentifier);

		clearTimeout(seenDebounceTimer);
		seenDebounceTimer = setTimeout(async () => {
			const [type, id] = conversationId.split('-');

			// 1. Mark individual messages as seen (for read receipts)
			if (seenQueue.size > 0) {
				const ids = Array.from(seenQueue);
				try {
					await markMessagesAsSeen(conversationKey, ids);
					// Locally update message status if needed?
					// The websocket event will handle it usually, but we could do optimistic update.
				} catch (e) {
					console.error('Failed to mark messages as seen:', e);
				}
				seenQueue.clear();
			}

			// 2. Mark conversation as seen (to reset unread count)
			// We can do this less frequently or just once per batch.
			markConversationAsSeen(
				conversationId,
				new Date().toISOString(),
				type === 'group',
				conversationKey
			);
		}, 1000); // 1s debounce
	}

	function handleMessageDeleted(event: CustomEvent) {
		const deletedMsgId = event.detail.id;
		messages = messages.map((m) => {
			if (m.id === deletedMsgId) {
				return {
					...m,
					content: '[deleted]',
					content_type: 'deleted',
					is_deleted: true,
					media_urls: []
				};
			}
			return m;
		});
	}
	// Init E2EE on load
	$effect(() => {
		initE2EE();
	});
</script>

<div class="flex h-full flex-col">
	<!-- Chat Header -->
	<header class="flex items-center space-x-4 border-b border-gray-200 bg-white p-4">
		{#if conversationPartner}
			<div class="relative">
				<Avatar class="h-10 w-10">
					<AvatarImage src={conversationPartner.avatar} alt={conversationPartner.name} />
					<AvatarFallback>{conversationPartner.name.charAt(0).toUpperCase()}</AvatarFallback>
				</Avatar>
				{#if !conversationPartner.is_group && presenceState[conversationPartner.id.replace('user-', '')]?.status === 'online'}
					<span
						class="absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-white bg-green-500"
					></span>
				{/if}
			</div>
			<div>
				<h3 class="font-semibold text-gray-900">{conversationPartner.name}</h3>
				{#if conversationPartner.is_group}
					<button
						class="text-xs text-zinc-500 hover:text-zinc-700"
						onclick={() => (showGroupInfo = true)}
					>
						View Group Info
					</button>
				{:else}
					<p class="text-xs text-zinc-400">
						{#if presenceState[conversationPartner.id.replace('user-', '')]?.status === 'online'}
							<span class="text-emerald-500">Online</span>
						{:else if presenceState[conversationPartner.id.replace('user-', '')]?.last_seen}
							Last seen {formatDistanceToNow(
								new Date(
									presenceState[conversationPartner.id.replace('user-', '')].last_seen * 1000
								),
								{
									addSuffix: true
								}
							)}
						{:else}
							Offline
						{/if}
					</p>
				{/if}
			</div>

			{#if conversationType === 'user'}
				<div class="ml-auto flex items-center gap-2">
					<button
						onclick={() => voiceCallService.startCall(currentChatId, 'audio')}
						class="p-2 text-zinc-400 transition-colors hover:text-zinc-600"
						title="Voice Call"
					>
						<Phone size={20} />
					</button>
					<button
						onclick={() => voiceCallService.startCall(currentChatId, 'video')}
						class="p-2 text-zinc-400 transition-colors hover:text-zinc-600"
						title="Video Call"
					>
						<Video size={20} />
					</button>
				</div>
			{/if}
			{#if conversationPartner.is_group}
				<div class="ml-auto flex items-center gap-2">
					<button
						class="rounded-full p-2 text-gray-500 hover:bg-gray-100 hover:text-gray-700"
						onclick={() => (showGroupInfo = true)}
						aria-label="Group Info"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="1.5"
							stroke="currentColor"
							class="h-6 w-6"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0Z"
							/>
						</svg>
					</button>
				</div>
			{:else}
				<div class="ml-auto flex items-center gap-2">
					<!-- E2EE Toggle -->
					{#if conversationType === 'user' && myKeyPair && !partnerPublicKey}
						<button
							class="flex cursor-pointer items-center gap-2 rounded-full bg-yellow-100 px-3 py-1 text-sm font-medium text-yellow-700 hover:bg-yellow-200"
							onclick={loadConversationPartner}
							title="Partner has not set up E2EE yet. Click to check again."
						>
							<svg
								xmlns="http://www.w3.org/2000/svg"
								viewBox="0 0 20 20"
								fill="currentColor"
								class="h-4 w-4"
							>
								<path
									fill-rule="evenodd"
									d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z"
									clip-rule="evenodd"
								/>
							</svg>
							Secure (Waiting...)
						</button>
					{:else}
						<button
							class={`flex items-center gap-2 rounded-full px-3 py-1 text-sm font-medium transition-colors ${
								isEncrypted
									? 'bg-green-100 text-green-700'
									: 'bg-gray-100 text-gray-600 hover:bg-gray-200'
							}`}
							onclick={toggleEncryption}
							title={isEncrypted ? 'Encryption Enabled' : 'Enable End-to-End Encryption'}
						>
							<svg
								xmlns="http://www.w3.org/2000/svg"
								viewBox="0 0 20 20"
								fill="currentColor"
								class="h-4 w-4"
							>
								<path
									fill-rule="evenodd"
									d="M10 1a4.5 4.5 0 00-4.5 4.5V9H5a2 2 0 00-2 2v6a2 2 0 002 2h10a2 2 0 002-2v-6a2 2 0 00-2-2h-.5V5.5A4.5 4.5 0 0010 1zm3 8V5.5a3 3 0 10-6 0V9h6z"
									clip-rule="evenodd"
								/>
							</svg>
							{isEncrypted ? 'Secure' : 'Encrypt'}
						</button>
					{/if}
				</div>
			{/if}
		{:else}
			<h2 class="text-lg font-bold">Chat</h2>
		{/if}
	</header>

	{#if showGroupInfo}
		<GroupInfoModal
			showModal={showGroupInfo}
			groupId={currentChatId}
			onClose={() => (showGroupInfo = false)}
		/>
	{/if}

	<!-- E2EE Restore Banner -->
	{#if !myKeyPair && auth.state.user?.encrypted_private_key && conversationType === 'user'}
		<div class="bg-blue-50 p-2 text-center text-sm text-blue-700">
			You have an encrypted backup.
			<button class="font-bold underline" onclick={() => (showRestoreModal = true)}>
				Restore Keys
			</button>
			to read secure messages.
		</div>
	{/if}

	<!-- Message Display Area -->
	<div
		bind:this={chatContainer}
		class="flex-1 overflow-y-auto bg-gray-50 p-4"
		onscroll={handleScroll}
	>
		{#if isFetchingMore}
			<div class="flex justify-center p-2">
				<span class="text-xs text-gray-400">Loading older messages...</span>
			</div>
		{/if}
		{#if isLoading}
			<!-- Skeleton Loader for Messages -->
			<div class="animate-pulse space-y-4">
				<!-- Left-aligned skeleton (received message) -->
				<div class="flex items-start gap-2.5">
					<div class="h-8 w-8 rounded-full bg-gray-200"></div>
					<div class="flex w-full max-w-[280px] flex-col gap-1">
						<div class="h-3 w-20 rounded bg-gray-200"></div>
						<div class="h-16 rounded-lg bg-gray-200"></div>
					</div>
				</div>
				<!-- Right-aligned skeleton (sent message) -->
				<div class="flex flex-row-reverse items-start gap-2.5">
					<div class="h-8 w-8 rounded-full bg-gray-200"></div>
					<div class="flex w-full max-w-[280px] flex-col items-end gap-1">
						<div class="h-3 w-20 rounded bg-gray-200"></div>
						<div class="h-12 w-48 rounded-lg bg-blue-100"></div>
					</div>
				</div>
				<!-- Another left-aligned skeleton -->
				<div class="flex items-start gap-2.5">
					<div class="h-8 w-8 rounded-full bg-gray-200"></div>
					<div class="flex w-full max-w-[280px] flex-col gap-1">
						<div class="h-3 w-16 rounded bg-gray-200"></div>
						<div class="h-20 w-64 rounded-lg bg-gray-200"></div>
					</div>
				</div>
				<!-- Another right-aligned skeleton -->
				<div class="flex flex-row-reverse items-start gap-2.5">
					<div class="h-8 w-8 rounded-full bg-gray-200"></div>
					<div class="flex w-full max-w-[280px] flex-col items-end gap-1">
						<div class="h-3 w-24 rounded bg-gray-200"></div>
						<div class="h-10 w-40 rounded-lg bg-blue-100"></div>
					</div>
				</div>
			</div>
		{:else if error}
			<p class="text-red-500">{error}</p>
		{:else}
			{#each messages as message (message.id)}
				<div use:trackVisibility={{ onVisible: () => handleMessageVisible(message) }}>
					<Message
						{message}
						on:rendered={handleMessageRendered}
						on:deleted={handleMessageDeleted}
						{conversationId}
						{conversationKey}
						{onProductClick}
					/>
					<!-- We can pass conversationId or callbacks to Message if needed for specific actions -->
				</div>
			{/each}

			{#if isOpponentTyping}
				<div class="my-2 flex items-center gap-2.5">
					<!-- Typing indicator -->
					<div class="flex items-center space-x-1">
						<span class="h-2 w-2 animate-pulse rounded-full bg-gray-400"></span>
						<span class="h-2 w-2 animate-pulse rounded-full bg-gray-400 delay-75"></span>
						<span class="h-2 w-2 animate-pulse rounded-full bg-gray-400 delay-150"></span>
					</div>
				</div>
			{/if}
		{/if}
	</div>

	<!-- Pending Product Attachment Preview -->
	{#if pendingProduct}
		<div class="flex items-center justify-between border-t border-gray-200 bg-gray-50 px-4 py-2">
			<div class="flex items-center gap-3">
				<img
					src={pendingProduct.images?.[0] || 'https://via.placeholder.com/50'}
					alt="Product"
					class="h-12 w-12 rounded border border-gray-300 object-cover"
				/>
				<div>
					<p class="text-sm font-semibold text-gray-900">{pendingProduct.title}</p>
					<p class="text-xs text-gray-500">${pendingProduct.price}</p>
				</div>
			</div>
			<button
				class="text-gray-400 hover:text-gray-600"
				onclick={() => {
					pendingProduct = null;
					messageInputDraft = '';
				}}
			>
				✕
			</button>
		</div>
	{/if}

	<MessageInput
		bind:value={messageInputDraft}
		{conversationId}
		{isMarketplace}
		onSend={async (content, files) => {
			// Inject product ID to content payload if pending
			// Since our MessageInput just returns content/files, we need to handle the sending manually OR modify handleSendMessage
			// Let's modify handleSendMessage to accept optional product_id
			await handleSendMessage(content, files, pendingProduct?.id);
			pendingProduct = null; // Clear after send
			messageInputDraft = '';
		}}
		onTyping={() => {
			// typing logic handled in MessageInput itself
		}}
	/>

	<Lightbox />

	<!-- Key Setup Modal (Generation) -->
	{#if showRecoverySetup}
		<div class="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
			<div class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
				<h3 class="mb-4 text-xl font-bold">Setup Secure Encryption</h3>
				<p class="mb-4 text-gray-600">
					To rely on end-to-end encryption, we need to generate a secure key pair for your device.
					Please enter a <strong>Recovery Password</strong>. This password will encrypt your
					specific private key so you can recover it on another device. We do NOT store your raw
					private key.
				</p>
				<input
					type="password"
					placeholder="Enter Recovery Password"
					bind:value={recoveryPassword}
					class="mb-4 w-full rounded border p-2"
				/>
				<div class="flex justify-end gap-2">
					<button
						class="rounded px-4 py-2 text-gray-600 hover:bg-gray-100"
						onclick={() => (showRecoverySetup = false)}
					>
						Cancel
					</button>
					<button
						class="rounded bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 disabled:opacity-50"
						onclick={setupKeys}
						disabled={isSettingUpKeys || !recoveryPassword}
					>
						{isSettingUpKeys ? 'Generating...' : 'Generate & Enable'}
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- Key Restore Modal -->
	{#if showRestoreModal}
		<div class="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
			<div class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
				<h3 class="mb-4 text-xl font-bold">Restore Encryption Keys</h3>
				<p class="mb-4 text-gray-600">
					We found an existing key backup for your account. Please enter your <strong
						>Recovery Password</strong
					>
					to decrypt and restore your keys on this device.
				</p>
				<input
					type="password"
					placeholder="Enter Recovery Password"
					bind:value={recoveryPassword}
					class="mb-4 w-full rounded border p-2"
				/>
				<div class="flex justify-end gap-2">
					<button
						class="rounded px-4 py-2 text-gray-600 hover:bg-gray-100"
						onclick={() => (showRestoreModal = false)}
					>
						Cancel
					</button>
					<button
						class="rounded bg-green-600 px-4 py-2 text-white hover:bg-green-700 disabled:opacity-50"
						onclick={restoreKeys}
						disabled={isRestoringKeys || !recoveryPassword}
					>
						{isRestoringKeys ? 'Restoring...' : 'Restore Keys'}
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- Key Rotate Modal -->
	{#if showRotateModal}
		<div class="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
			<div class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
				<h3 class="mb-4 text-xl font-bold text-orange-600">⚠️ Rotate Encryption Keys</h3>
				<p class="mb-4 text-gray-600">
					This will generate <strong>NEW</strong> encryption keys. Your old keys will be destroyed.
					<br /><br />
					<strong class="text-red-600">Warning:</strong> All previously encrypted messages will become
					unreadable unless you backed up your old keys externally.
				</p>
				<input
					type="password"
					placeholder="Enter NEW Recovery Password"
					bind:value={rotationPassword}
					class="mb-4 w-full rounded border p-2"
				/>
				<div class="flex justify-end gap-2">
					<button
						class="rounded px-4 py-2 text-gray-600 hover:bg-gray-100"
						onclick={() => (showRotateModal = false)}
					>
						Cancel
					</button>
					<button
						class="rounded bg-orange-600 px-4 py-2 text-white hover:bg-orange-700 disabled:opacity-50"
						onclick={rotateKeys}
						disabled={isRotatingKeys || !rotationPassword}
					>
						{isRotatingKeys ? 'Rotating...' : 'Rotate Keys'}
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- Security Info / Verification Modal -->
	{#if showSecurityInfo}
		<div class="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
			<div class="w-full max-w-lg rounded-lg bg-white p-6 shadow-xl">
				<div class="mb-4 flex items-center justify-between">
					<h3 class="text-xl font-bold text-gray-800">🔒 Security Verification</h3>
					<button
						class="text-gray-500 hover:text-gray-700"
						onclick={() => (showSecurityInfo = false)}
					>
						✕
					</button>
				</div>

				<p class="mb-6 text-sm text-gray-600">
					End-to-End Encryption is active. You can verify the security of this chat by comparing the
					fingerprints below with your partner's device.
				</p>

				<div class="mb-6 space-y-4">
					<div class="rounded-lg bg-gray-50 p-4">
						<p class="mb-1 text-xs font-semibold uppercase text-gray-500">Your Fingerprint</p>
						<code class="break-all font-mono text-sm text-blue-800"
							>{myFingerprint || 'Loading...'}</code
						>
					</div>

					<div class="rounded-lg bg-gray-50 p-4">
						<p class="mb-1 text-xs font-semibold uppercase text-gray-500">Partner's Fingerprint</p>
						{#if partnerFingerprint}
							<code class="break-all font-mono text-sm text-green-800">{partnerFingerprint}</code>
						{:else}
							<p class="text-sm text-yellow-600">Not Available (Partner key missing)</p>
						{/if}
					</div>
				</div>

				<div class="flex justify-between border-t pt-4">
					<div class="flex gap-2">
						{#if auth.state.user?.is_encryption_enabled !== false}
							<button
								class="text-red-500 hover:text-red-700"
								onclick={async () => {
									try {
										const updatedUser = await updateUserProfile({
											is_encryption_enabled: false
										});
										auth.updateUser(updatedUser);
										isEncrypted = false;
										showSecurityInfo = false;
									} catch (e) {
										console.error('Failed to disable encryption:', e);
									}
								}}
							>
								Disable Encryption (Permanently)
							</button>
						{:else}
							<button
								class="text-green-600 hover:text-green-800"
								onclick={async () => {
									try {
										const updatedUser = await updateUserProfile({
											is_encryption_enabled: true
										});
										auth.updateUser(updatedUser);
										// It will auto-enable via the effect
										showSecurityInfo = false;
									} catch (e) {
										console.error('Failed to enable encryption:', e);
									}
								}}
							>
								Enable Encryption
							</button>
						{/if}
						<button
							class="text-orange-500 hover:text-orange-700"
							onclick={() => {
								showSecurityInfo = false;
								showRotateModal = true;
							}}
						>
							Rotate Keys
						</button>
					</div>
					<button
						class="rounded bg-blue-600 px-6 py-2 text-white hover:bg-blue-700"
						onclick={() => (showSecurityInfo = false)}
					>
						Close
					</button>
				</div>
			</div>
		</div>
	{/if}
</div>
