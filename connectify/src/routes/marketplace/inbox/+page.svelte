<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import Skeleton from '$lib/components/ui/skeleton/Skeleton.svelte';
	import * as Icons from '@lucide/svelte';
	import { getMarketplaceConversations, getProduct } from '$lib/api/marketplace';
	import { getUserByID, type ConversationSummary } from '$lib/api';
	import type { Product } from '$lib/api/marketplace';
	import ChatWindow from '$lib/components/messages/ChatWindow.svelte';
	import ProductDetailsModal from '$lib/components/marketplace/ProductDetailsModal.svelte';
	import { formatDistanceToNow } from 'date-fns';
	import { auth } from '$lib/stores/auth.svelte';
	import { page } from '$app/stores';
	import { replaceState } from '$app/navigation';
	import { presenceStore, type PresenceState } from '$lib/stores/presence';
	import { websocketMessages } from '$lib/websocket';

	// Inbox State
	let conversations = $state<ConversationSummary[]>([]);
	let selectedConversationId = $state<string | null>(null);
	let loadingConversations = $state(true);

	// Presence state (same pattern as ConversationList)
	let presenceState = $state<PresenceState>({});

	// Product modal state (for when clicking product in chat)
	let selectedProduct = $state<Product | null>(null);

	// Pre-fill state for new conversations (from URL params)
	let pendingProduct = $state<Product | null>(null);
	let pendingMessage = $state<string>('');

	// Cleanup functions
	let unsubscribePresence: (() => void) | null = null;
	let unsubscribeWS: (() => void) | null = null;

	async function loadConversations() {
		loadingConversations = true;
		try {
			conversations = await getMarketplaceConversations();
		} catch (err) {
			console.error('Failed to load conversations:', err);
		} finally {
			loadingConversations = false;
		}
	}

	onMount(async () => {
		await loadConversations();

		// Restore conversation from URL query param (if present)
		const chatParam = $page.url.searchParams.get('chat');
		if (chatParam) {
			selectedConversationId = chatParam;
		}

		// Subscribe to Presence Store (same as ConversationList)
		unsubscribePresence = presenceStore.subscribe((value) => {
			presenceState = value;
		});

		// Subscribe to WebSocket messages for real-time updates (same pattern as ConversationList)
		unsubscribeWS = websocketMessages.subscribe((event) => {
			if (!event) return;

			switch (event.type) {
				case 'MARKETPLACE_MESSAGE_CREATED': {
					const newMessage = event.data;

					console.log('[MarketplaceInbox] WS Received:', {
						id: newMessage.id,
						is_marketplace: newMessage.is_marketplace
					});

					// Only process marketplace messages
					if (newMessage.is_marketplace !== true) {
						console.log('[MarketplaceInbox] Ignored non-marketplace message');
						return;
					}

					console.log('[MarketplaceInbox] Received MESSAGE_CREATED:', newMessage);

					// Update the conversation's last message and timestamp
					let updated = false;
					conversations = conversations.map((conv) => {
						// For direct messages, check if either sender or receiver is this conversation
						if (newMessage.sender_id === conv.id || newMessage.receiver_id === conv.id) {
							updated = true;
							return {
								...conv,
								last_message_content: newMessage.content || 'Sent a file',
								last_message_timestamp: newMessage.created_at,
								last_message_sender_id: newMessage.sender_id,
								last_message_sender_name: newMessage.sender_name,
								last_message_is_encrypted: newMessage.is_encrypted,
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
					}
					break;
				}
				case 'CONVERSATION_SEEN_UPDATE': {
					const { conversation_id, conversation_ui_id, user_id, is_group } = event.data;
					if (user_id !== auth.state.user?.id || is_group) {
						break;
					}
					const normalizedId =
						conversation_ui_id ||
						(conversation_id?.startsWith('user-') ? conversation_id : `user-${conversation_id}`);
					if (!normalizedId) break;
					conversations = conversations.map((conv) => {
						const convId = conv.id.startsWith('user-') ? conv.id : `user-${conv.id}`;
						if (convId === normalizedId) {
							return { ...conv, unread_count: 0 };
						}
						return conv;
					});
					break;
				}
			}
		});

		// Check URL for seller/product params (from "Message Seller" click)
		const sellerId = $page.url.searchParams.get('seller');
		const productId = $page.url.searchParams.get('product_id');
		const productTitle = $page.url.searchParams.get('product_title');

		if (sellerId && productId) {
			// Select the conversation with this seller (with user- prefix for ChatWindow)
			const prefixedSellerId = sellerId.startsWith('user-') ? sellerId : `user-${sellerId}`;
			selectedConversationId = prefixedSellerId;

			// Check if this seller is already in our conversation list (raw ID comparison)
			const rawSellerId = sellerId.startsWith('user-') ? sellerId.slice(5) : sellerId;
			const existingConv = conversations.find((c) => c.id === rawSellerId || c.id === sellerId);

			if (!existingConv) {
				// Create a placeholder conversation for this seller
				try {
					const sellerInfo = await getUserByID(sellerId);
					const placeholderConv: ConversationSummary = {
						id: sellerId,
						name: sellerInfo.username || 'Seller',
						avatar: sellerInfo.avatar,
						is_group: false,
						unread_count: 0,
						last_message_content: 'New conversation',
						last_message_timestamp: undefined
					};
					conversations = [placeholderConv, ...conversations];
				} catch (err) {
					console.error('Failed to fetch seller info:', err);
					// Still create placeholder with minimal info
					const placeholderConv: ConversationSummary = {
						id: sellerId,
						name: productTitle ? `${productTitle} Seller` : 'Seller',
						is_group: false,
						unread_count: 0,
						last_message_content: 'New conversation'
					};
					conversations = [placeholderConv, ...conversations];
				}
			}

			// Fetch product details for the pending product attachment
			try {
				pendingProduct = await getProduct(productId);
				pendingMessage = `Hi, is this still available?`;
			} catch (err) {
				console.error('Failed to load product for pre-fill:', err);
				pendingMessage = `Hi, I'm interested in: ${productTitle || 'your product'}`;
			}

			// Clear URL params
			const url = new URL($page.url);
			url.searchParams.delete('seller');
			url.searchParams.delete('product_id');
			url.searchParams.delete('product_title');
			replaceState(url, {});
		}
	});

	onDestroy(() => {
		if (unsubscribePresence) unsubscribePresence();
		if (unsubscribeWS) unsubscribeWS();
	});

	function handleCloseProductModal() {
		selectedProduct = null;
	}

	async function handleMessageSeller(event: CustomEvent) {
		const { product } = event.detail;
		if (!auth.state.user) {
			alert('Please log in to message seller');
			return;
		}
		selectedProduct = null;
		await loadConversations();
	}

	// Called when a message is sent from ChatWindow (sender side update)
	function handleMessageSent() {
		loadConversations();
	}

	// Handle conversation selection with URL update
	function selectConversation(convId: string) {
		selectedConversationId = convId;
		// Update URL with query parameter (avoids 404 on refresh)
		const url = new URL($page.url);
		url.searchParams.set('chat', convId);
		replaceState(url, {});
	}
</script>

<div class="flex h-screen overflow-hidden bg-[#f0f2f5]">
	<!-- Product Details Modal -->
	{#if selectedProduct}
		<ProductDetailsModal
			product={selectedProduct}
			on:close={handleCloseProductModal}
			on:message={handleMessageSeller}
		/>
	{/if}

	<!-- Sidebar with Tabs and Conversation List -->
	<div class="flex h-full w-80 flex-shrink-0 flex-col border-r border-gray-200 bg-white">
		<div class="p-4">
			<h1 class="mb-4 text-2xl font-bold text-gray-900">Marketplace</h1>

			<!-- Navigation Tabs -->
			<div class="mb-4 flex gap-2 overflow-x-auto pb-2">
				<a
					href="/marketplace"
					class="whitespace-nowrap rounded-full bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200"
				>
					Browse
				</a>
				<a
					href="/marketplace/selling"
					class="whitespace-nowrap rounded-full bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200"
				>
					Selling
				</a>
				<a
					href="/marketplace/inbox"
					class="whitespace-nowrap rounded-full bg-blue-100 px-4 py-2 text-sm font-medium text-blue-600 transition-colors"
				>
					Inbox
				</a>
			</div>

			<div class="border-t border-gray-200"></div>
		</div>

		<!-- Conversation List -->
		<div class="flex-1 overflow-y-auto">
			{#if loadingConversations}
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
			{:else if conversations.length === 0}
				<div class="p-8 text-center text-gray-500">
					<Icons.MessageSquare size={32} class="mx-auto mb-2 opacity-50" />
					<p>No messages yet.</p>
					<p class="mt-2 text-sm">Start a conversation by messaging a seller!</p>
					<a
						href="/marketplace"
						class="mt-4 inline-block rounded-full bg-blue-600 px-4 py-2 text-sm font-medium text-white"
					>
						Browse Products
					</a>
				</div>
			{:else}
				<ul class="divide-y divide-gray-100">
					{#each conversations as conv}
						{@const convId = conv.id.startsWith('user-') ? conv.id : `user-${conv.id}`}
						{@const userId = conv.id.includes('-') ? conv.id.split('-')[1] : conv.id}
						{@const isOnline = presenceState[userId]?.status === 'online'}
						<button
							class="flex w-full items-center gap-3 p-4 text-left transition-colors hover:bg-gray-50 {selectedConversationId ===
							convId
								? 'bg-blue-50'
								: ''}"
							onclick={() => selectConversation(convId)}
						>
							<div class="relative">
								<img
									src={conv.avatar || `https://ui-avatars.com/api/?name=${conv.name}`}
									alt=""
									class="h-12 w-12 rounded-full bg-gray-200 object-cover"
								/>
								<!-- Online status indicator (using presenceStore) -->
								{#if isOnline}
									<span
										class="absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-white bg-green-500"
									></span>
								{/if}
							</div>
							<div class="min-w-0 flex-1">
								<div class="flex items-baseline justify-between">
									<span class="truncate font-semibold text-gray-900">{conv.name}</span>
									{#if conv.last_message_timestamp}
										<span class="ml-2 whitespace-nowrap text-xs text-gray-500">
											{formatDistanceToNow(new Date(conv.last_message_timestamp), {
												addSuffix: false
											})}
										</span>
									{/if}
								</div>
								<p class="mt-0.5 truncate text-sm text-gray-500">
									{conv.last_message_content || 'Started a chat'}
								</p>
							</div>
							{#if conv.unread_count && conv.unread_count > 0}
								<span
									class="flex h-5 min-w-5 items-center justify-center rounded-full bg-blue-600 px-1.5 text-xs font-medium text-white"
								>
									{conv.unread_count}
								</span>
							{/if}
						</button>
					{/each}
				</ul>
			{/if}
		</div>
	</div>

	<!-- Chat Window -->
	<div class="relative flex h-full flex-1 flex-col bg-gray-50">
		{#if selectedConversationId}
			<ChatWindow
				conversationId={selectedConversationId}
				isMarketplace={true}
				initialProduct={pendingProduct}
				initialMessage={pendingMessage}
				onMessageSent={handleMessageSent}
				onProductClick={async (productId) => {
					try {
						const product = await getProduct(productId);
						selectedProduct = product;
					} catch (err) {
						console.error('Failed to load product:', err);
					}
				}}
			/>
		{:else}
			<div class="flex h-full flex-col items-center justify-center text-gray-400">
				<Icons.MessageCircle size={48} class="mb-4 opacity-50" />
				<p>Select a conversation to start chatting</p>
			</div>
		{/if}
	</div>
</div>
