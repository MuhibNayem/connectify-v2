<!--
Component to display a list of conversations (both direct messages and groups).
Fetches friends and groups to populate the list.
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		getConversationSummaries,
		getFriends,
		searchFriends,
		type ConversationSummary,
		type User
	} from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import Skeleton from '$lib/components/ui/skeleton.svelte';
	import { presenceStore, type PresenceState } from '$lib/stores/presence';
	import { formatDistanceToNow } from 'date-fns';
	import CreateGroupModal from './CreateGroupModal.svelte';
	import { Plus, Search, MessageCircle } from '@lucide/svelte';
	import { websocketMessages } from '$lib/websocket';

	let conversations = $state<ConversationSummary[]>([]);
	let suggestedFriends = $state<User[]>([]); // Prefetched friends
	let searchResults = $state<User[]>([]);
	let searchQuery = $state('');
	let isSearching = $state(false);

	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let showCreateGroupModal = $state(false);
	let presenceState = $state<PresenceState>({});

	// Managed in onMount

	// Subscribe to WebSocket messages for real-time updates
	// WebSocket subscription managed in onMount to avoid dependency cycles

	onMount(() => {
		// Subscribe to Presence Store
		const unsubscribePresence = presenceStore.subscribe((value) => {
			presenceState = value;
		});

		// Subscribe to WebSocket messages
		const unsubscribeWS = websocketMessages.subscribe((event) => {
			if (!event) return;

			switch (event.type) {
				case 'MESSAGE_CREATED': {
					const newMessage = event.data;
					console.log('[ConversationList] WS Received:', {
						id: newMessage.id,
						is_marketplace: newMessage.is_marketplace
					});

					// Skip marketplace messages - they belong to marketplace inbox, not personal DMs
					if (newMessage.is_marketplace === true) {
						console.log('[ConversationList] Ignored marketplace message');
						break;
					}

					// Update the conversation's last message and timestamp
					let updated = false;
					conversations = conversations.map((conv) => {
						let shouldUpdate = false;

						// Extract raw ID from prefixed conv.id for comparison
						// conv.id is "user-xxx" or "group-xxx", extract the raw ID
						const convRawId = conv.id.includes('-')
							? conv.id.split('-').slice(1).join('-')
							: conv.id;

						// Check if message belongs to this conversation
						if (
							conv.is_group &&
							(`group-${newMessage.group_id}` === conv.id || newMessage.group_id === convRawId)
						) {
							shouldUpdate = true;
						} else if (!conv.is_group && !newMessage.group_id) {
							// For direct messages, check if either sender or receiver matches
							if (newMessage.sender_id === convRawId || newMessage.receiver_id === convRawId) {
								shouldUpdate = true;
							}
						}

						if (shouldUpdate) {
							updated = true;
							// Update last message info
							return {
								...conv,
								last_message_content: newMessage.content || 'Sent a file',
								last_message_timestamp: newMessage.created_at,
								last_message_sender_id: newMessage.sender_id,
								last_message_sender_name: newMessage.sender_name,
								last_message_is_encrypted: newMessage.is_encrypted,
								// Increment unread count if message is from someone else, reset to 0 if from current user
								unread_count:
									newMessage.sender_id !== auth.state.user?.id ? (conv.unread_count || 0) + 1 : 0
							};
						}
						return conv;
					});

					// Re-sort conversations by timestamp
					if (updated) {
						conversations = [...conversations].sort((a, b) => {
							const timeA = a.last_message_timestamp
								? new Date(a.last_message_timestamp).getTime()
								: 0;
							const timeB = b.last_message_timestamp
								? new Date(b.last_message_timestamp).getTime()
								: 0;
							return timeB - timeA;
						});
					} else {
						// New conversation started - optimistically add it
						const isSentByMe = newMessage.sender_id === auth.state.user?.id;
						const partnerId = isSentByMe ? newMessage.receiver_id : newMessage.sender_id;

						// Try to find partner info from suggestedFriends (already loaded)
						const partnerInfo = suggestedFriends.find((f) => f.id === partnerId);

						const newConv: ConversationSummary = {
							id: `user-${partnerId}`,
							name:
								partnerInfo?.username ||
								partnerInfo?.full_name ||
								(isSentByMe ? newMessage.receiver_name : newMessage.sender_name) ||
								'Chat',
							avatar:
								partnerInfo?.avatar ||
								(isSentByMe ? newMessage.receiver_avatar : newMessage.sender_avatar) ||
								'',
							is_group: false,
							last_message_content: newMessage.content || 'Sent a file',
							last_message_timestamp: newMessage.created_at,
							last_message_sender_id: newMessage.sender_id,
							last_message_sender_name: newMessage.sender_name,
							unread_count: newMessage.sender_id !== auth.state.user?.id ? 1 : 0
						};
						conversations = [newConv, ...conversations];
					}
					break;
				}
				case 'CONVERSATION_SEEN_UPDATE': {
					const { conversation_id, conversation_ui_id, user_id, is_group } = event.data;
					if (user_id === auth.state.user?.id) {
						const normalizedId =
							conversation_ui_id ||
							(is_group
								? `group-${conversation_id}`
								: conversation_id?.startsWith('user-')
									? conversation_id
									: `user-${conversation_id}`);
						if (normalizedId) {
							conversations = conversations.map((conv) =>
								conv.id === normalizedId ? { ...conv, unread_count: 0 } : conv
							);
						}
					}
					break;
				}
				case 'GROUP_UPDATED': {
					const updatedGroup = event.data;
					conversations = conversations.map((conv) => {
						if (conv.is_group && conv.id === updatedGroup.id) {
							return { ...conv, name: updatedGroup.name, avatar: updatedGroup.avatar };
						}
						return conv;
					});
					break;
				}
				case 'GROUP_CREATED': {
					const newGroup = event.data;
					if (!conversations.some((c) => c.id === `group-${newGroup.id}`)) {
						const newConversation: ConversationSummary = {
							id: `group-${newGroup.id}`,
							name: newGroup.name,
							avatar: newGroup.avatar,
							is_group: true,
							last_message_content: newGroup.creator?.username
								? `${newGroup.creator.username} created the group`
								: 'Group created',
							last_message_timestamp: newGroup.created_at,
							unread_count: 0
						};
						conversations = [newConversation, ...conversations];
					}
					break;
				}
			}
		});

		// Async Data Fetching
		refreshConversations();

		// Return cleanup function
		return () => {
			unsubscribePresence();
			unsubscribeWS();
		};
	});

	async function refreshConversations() {
		if (!auth.state.user) {
			error = 'User not authenticated.';
			isLoading = false;
			return;
		}

		try {
			const fetchedConversations = await getConversationSummaries();

			// Handle empty list or null
			const validConversations = fetchedConversations || [];

			// Sort conversations by timestamp of last message (newest first)
			const sorted = validConversations.sort((a, b) => {
				const timeA = a.last_message_timestamp ? new Date(a.last_message_timestamp).getTime() : 0;
				const timeB = b.last_message_timestamp ? new Date(b.last_message_timestamp).getTime() : 0;
				return timeB - timeA;
			});
			conversations = sorted;

			// Prefetch friends (as requested "pre fetched some friends")
			try {
				suggestedFriends = await getFriends();
			} catch (friendErr) {
				console.log('Failed to prefetch friends', friendErr);
			}
		} catch (e: any) {
			error = e.message || 'Failed to load conversations.';
		} finally {
			isLoading = false;
		}
	}

	async function handleSearch() {
		if (searchQuery.trim().length === 0) {
			searchResults = [];
			return;
		}
		isSearching = true;
		try {
			searchResults = await searchFriends(searchQuery);
		} catch (err) {
			console.error('Search failed', err);
		} finally {
			isSearching = false;
		}
	}

	let searchTimeout: NodeJS.Timeout;
	function onSearchInput() {
		clearTimeout(searchTimeout);
		searchTimeout = setTimeout(() => {
			handleSearch();
		}, 300);
	}

	function getConversationUrl(conv: ConversationSummary): string {
		// Backend now returns IDs with prefix: "user-<id>" or "group-<id>"
		return `/messages/${conv.id}`;
	}

	// Helper to display encrypted message placeholder
	function getDisplayContent(
		content: string | undefined,
		isEncrypted: boolean | undefined
	): string {
		if (!content) return '';
		if (isEncrypted) {
			return '🔒 Encrypted message';
		}
		return content;
	}
</script>

<div class="flex h-full flex-col border-r border-gray-200 bg-gray-50">
	<!-- Header -->
	<div class="flex flex-col space-y-3 border-b border-gray-200 p-4">
		<div class="flex items-center justify-between">
			<h1 class="text-xl font-bold text-gray-800">Chats</h1>
			<button
				class="rounded-full bg-blue-500 p-2 text-white shadow-lg hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-50"
				on:click={() => (showCreateGroupModal = true)}
			>
				<Plus class="h-6 w-6" />
			</button>
		</div>

		<!-- Search Bar -->
		<div class="relative">
			<input
				type="text"
				placeholder="Search friends..."
				class="w-full rounded-lg border border-gray-300 px-4 py-2 pl-10 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
				bind:value={searchQuery}
				on:input={onSearchInput}
			/>
			<Search class="absolute left-3 top-2.5 h-4 w-4 text-gray-400" />
		</div>
	</div>

	<!-- Conversation List or Search Results -->
	<div class="flex-1 overflow-y-auto">
		{#if isLoading}
			<div class="space-y-3 p-4">
				{#each Array(5) as _, i (i)}
					<div class="flex items-center space-x-3">
						<Skeleton class="h-12 w-12 rounded-full" />
						<div class="flex-1 space-y-2">
							<Skeleton class="h-4 w-3/4" />
							<Skeleton class="h-4 w-1/2" />
						</div>
					</div>
				{/each}
			</div>
		{:else if error}
			<div class="p-4 text-red-500">{error}</div>
		{:else if searchQuery.length > 0}
			<!-- Search Results -->
			<div class="bg-gray-50 px-4 py-2 text-xs font-semibold uppercase text-gray-500">
				{searchResults.length > 0
					? 'Friends Found'
					: isSearching
						? 'Searching...'
						: 'No friends found'}
			</div>
			<ul class="divide-y divide-gray-200">
				{#each searchResults as friend}
					<li>
						<a
							href="/messages/{friend.id}"
							class="flex items-center p-3 transition hover:bg-gray-100"
						>
							<div class="relative">
								<img
									class="mr-4 h-10 w-10 rounded-full object-cover"
									src={friend.avatar || `https://i.pravatar.cc/150?u=${friend.id}`}
									alt={friend.full_name}
								/>
								{#if presenceState[friend.id]?.status === 'online'}
									<span
										class="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border-2 border-white bg-green-500"
									></span>
								{/if}
							</div>
							<div>
								<p class="font-medium text-gray-900">{friend.full_name || friend.username}</p>
								<p class="text-xs text-gray-500">@{friend.username}</p>
							</div>
						</a>
					</li>
				{/each}
			</ul>
		{:else if conversations.length === 0}
			<!-- Empty State + Prefetched Friends -->
			<div class="px-4 py-8 text-center">
				<div
					class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-blue-100"
				>
					<MessageCircle class="h-8 w-8 text-blue-600" />
				</div>
				<h3 class="mb-2 text-lg font-medium text-gray-900">Your Inbox is Empty</h3>
				<p class="mb-6 text-sm text-gray-500">Start chatting with your friends!</p>
			</div>

			{#if suggestedFriends.length > 0}
				<div class="px-4 pb-2">
					<h4 class="mb-2 text-xs font-semibold uppercase text-gray-400">Your Friends</h4>
				</div>
				<ul class="divide-y divide-gray-100">
					{#each suggestedFriends as friend}
						{@const isActive = $page.params.id === `user-${friend.id}`}
						<li>
							<a
								href="/messages/user-{friend.id}"
								class="flex items-center space-x-3 px-4 py-3 transition {isActive
									? 'border-l-4 border-blue-500 bg-blue-50'
									: 'hover:bg-gray-50'}"
							>
								<div class="relative">
									<img
										src={friend.avatar || `https://i.pravatar.cc/150?u=${friend.id}`}
										alt={friend.full_name}
										class="h-10 w-10 rounded-full object-cover"
									/>
									{#if presenceState[friend.id]?.status === 'online'}
										<span
											class="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border-2 border-white bg-green-500"
										></span>
									{/if}
								</div>
								<div class="min-w-0 flex-1">
									<p class="truncate font-medium text-gray-900">
										{friend.full_name || friend.username}
									</p>
									<p class="truncate text-xs text-gray-500">@{friend.username}</p>
								</div>
								<div class="flex-shrink-0">
									<MessageCircle class="h-5 w-5 text-gray-400" />
								</div>
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		{:else}
			<ul class="divide-y divide-gray-200">
				{#each conversations as conv (conv.id)}
					{@const conversationId = $page.params.id}
					{@const isActive = conversationId === `${conv.is_group ? 'group' : 'user'}-${conv.id}`}
					{@const userId = conv.id.includes('-') ? conv.id.split('-')[1] : conv.id}
					{@const isOnline = !conv.is_group && presenceState[userId]?.status === 'online'}
					<li>
						<a
							href={getConversationUrl(conv)}
							class="flex items-center p-3 transition duration-150 ease-in-out hover:bg-gray-100 {isActive
								? 'bg-blue-50'
								: ''}"
						>
							<div class="relative">
								<img
									class="mr-4 h-12 w-12 rounded-full"
									src={conv.avatar ?? `https://i.pravatar.cc/150?u=${conv.id}`}
									alt="{conv.name}'s avatar"
								/>
								{#if isOnline}
									<span
										class="absolute bottom-0 right-4 h-3 w-3 rounded-full border-2 border-white bg-green-500"
									></span>
								{/if}
							</div>
							<div class="min-w-0 flex-1">
								<div class="flex items-center justify-between">
									<p class="truncate font-semibold text-gray-800">{conv.name}</p>
									{#if conv.unread_count > 0}
										<span
											class="ml-2 flex h-5 w-5 items-center justify-center rounded-full bg-blue-600 text-xs font-bold text-white"
										>
											{conv.unread_count > 99 ? '99+' : conv.unread_count}
										</span>
									{/if}
								</div>
								{#if conv.last_message_content}
									<div class="flex items-center text-sm text-gray-500">
										<p class="max-w-[140px] truncate">
											{#if conv.last_message_sender_id === auth.state.user?.id}
												<span class="font-medium text-gray-900">You: </span>
											{:else if conv.is_group && conv.last_message_sender_name}
												<span class="font-medium text-gray-900"
													>{conv.last_message_sender_name}:
												</span>
											{/if}
											{getDisplayContent(conv.last_message_content, conv.last_message_is_encrypted)}
										</p>
										{#if conv.last_message_timestamp}
											<span class="mx-1">•</span>
											<span>
												{formatDistanceToNow(new Date(conv.last_message_timestamp), {
													addSuffix: false
												})
													.replace('about ', '')
													.replace('less than a minute', 'just now')
													.replace(' minute', 'm')
													.replace(' minutes', 'm')
													.replace(' hour', 'h')
													.replace(' hours', 'h')
													.replace(' day', 'd')
													.replace(' days', 'd')}
											</span>
										{/if}
									</div>
								{/if}
							</div>
						</a>
					</li>
				{/each}
			</ul>
		{/if}
	</div>

	<CreateGroupModal
		bind:showModal={showCreateGroupModal}
		onGroupCreated={(createdGroup) => {
			// Optimistic update: Add group directly to list (zero extra API calls)
			const newConversation: ConversationSummary = {
				id: `group-${createdGroup.id}`,
				name: createdGroup.name,
				avatar: createdGroup.avatar,
				is_group: true,
				last_message_content: createdGroup.creator?.username
					? `${createdGroup.creator.username} created the group`
					: 'Group created',
				last_message_timestamp: createdGroup.created_at,
				unread_count: 0
			};
			conversations = [newConversation, ...conversations];
			// Auto-navigate to the newly created group
			goto(`/messages/${newConversation.id}`);
		}}
	/>
</div>
