<script lang="ts">
	import { onMount } from 'svelte';
	import { createEventDispatcher } from 'svelte';
	import { auth } from '$lib/stores/auth.svelte';
	import { lightbox } from '$lib/stores/lightbox.svelte';
	import type { Message } from '$lib/types';
	import MessageActions from './MessageActions.svelte';
	import MessageReactions from './MessageReactions.svelte';
	import { getProduct, type Product } from '$lib/api/marketplace';
	import { fly } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';

	export let message: Message;
	export let conversationId: string = '';
	export let conversationKey: string = '';
	export let onProductClick: ((productId: string) => void) | undefined = undefined;

	const dispatch = createEventDispatcher();

	// Product state for marketplace messages
	// Use embedded product data if available, otherwise fallback to API call
	let linkedProduct: Product | null = null;
	let loadingProduct = false;

	onMount(async () => {
		dispatch('rendered', { messageId: message.id });

		// Use embedded product data from message response (optimized - no API call needed)
		if (message.product && message.product.id) {
			linkedProduct = message.product as Product;
		}
		// Fallback: Fetch product if message has product_id but no embedded data
		else if (message.product_id && !message.product) {
			loadingProduct = true;
			try {
				linkedProduct = await getProduct(message.product_id);
			} catch (e) {
				console.error('Failed to load linked product:', e);
			} finally {
				loadingProduct = false;
			}
		}
	});

	const isMe = auth.state.user?.id === message.sender_id;

	function parseContent(text: string) {
		if (!text) return '';
		// Replace URL with links
		// Replace @username with links
		return text.replace(
			/@(\w+)/g,
			'<a href="/profile/$1" class="font-semibold hover:underline">$1</a>'
		);
	}

	function getMediaType(url: string, index: number, msg: Message): MediaItem {
		// 1. Check optimistic file type if available
		if (msg._optimistic_files?.[index]) {
			const t = msg._optimistic_files[index].type;
			if (t.startsWith('image/'))
				return { url, type: 'image', name: msg._optimistic_files[index].name };
			if (t.startsWith('video/'))
				return { url, type: 'video', name: msg._optimistic_files[index].name };
			return { url, type: 'file', name: msg._optimistic_files[index].name };
		}

		// 2. Check extension
		let ext = '';
		try {
			const pathname = new URL(url, 'http://dummy.com').pathname;
			ext = pathname.split('.').pop()?.toLowerCase() || '';
		} catch {
			ext = url.split('.').pop()?.toLowerCase() || '';
		}

		const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'heic', 'heif', 'bmp', 'svg', 'tiff'];
		const videoExts = ['mp4', 'webm', 'ogg', 'mov', 'quicktime', 'm4v', 'avi', 'mkv'];

		if (imageExts.includes(ext)) return { url, type: 'image' };
		if (videoExts.includes(ext)) return { url, type: 'video' };

		// 3. Fallback to content_type if single file
		if (msg.media_urls?.length === 1) {
			if (msg.content_type === 'image' && !ext) return { url, type: 'image' };
			if (msg.content_type === 'video' && !ext) return { url, type: 'video' };
		}

		return { url, type: 'file' };
	}

	type MediaItem = { url: string; type: 'image' | 'video' | 'file'; name?: string };

	let mediaItems: MediaItem[] = [];
	let gridMedia: MediaItem[] = [];
	let attachments: MediaItem[] = [];

	$: mediaItems = message.media_urls?.map((url, i) => getMediaType(url, i, message)) || [];
	$: gridMedia = mediaItems.filter((m) => m.type === 'image' || m.type === 'video');
	$: attachments = mediaItems.filter((m) => m.type === 'file');

	function handleImgError(e: Event) {
		const target = e.currentTarget as HTMLImageElement;
		target.style.display = 'none';
		// Show fallback
		target.nextElementSibling?.classList.remove('hidden');
	}
</script>

<!-- FB-Style System Message (Group Activities) -->
{#if message.content_type === 'system'}
	<div class="w-full px-4 py-2" in:fly={{ y: 10, duration: 300, easing: quintOut }}>
		<div class="flex items-center justify-center gap-2">
			<div class="h-px flex-1 bg-gray-200"></div>
			<span class="rounded-full bg-gray-100 px-3 py-1 text-xs text-gray-500">
				{message.content}
			</span>
			<div class="h-px flex-1 bg-gray-200"></div>
		</div>
	</div>
{:else}
	<!-- Regular Message Bubble -->
	<div
		class="my-2 flex items-start gap-2.5"
		class:flex-row-reverse={isMe}
		in:fly={{ y: 20, duration: 400, easing: quintOut }}
	>
		<a href="/profile/{message.sender_id}" class="transition-opacity hover:opacity-80">
			<img
				class="h-8 w-8 rounded-full"
				src={(isMe ? auth.state.user?.avatar : message.sender?.avatar) ||
					`https://i.pravatar.cc/150?u=${message.sender_id}`}
				alt="{message.sender_name || 'User'}'s avatar"
			/>
		</a>
		<div class="flex w-full max-w-[320px] flex-col gap-1">
			<div class="flex items-center space-x-2" class:justify-end={isMe}>
				<a
					href="/profile/{message.sender_id}"
					class="text-sm font-semibold text-gray-900 hover:underline"
				>
					{message?.sender_name || 'User'}
				</a>
				<span class="text-xs font-normal text-gray-500"
					>{new Date(message.created_at).toLocaleTimeString()}</span
				>
				{#if isMe}
					{#if message.id.startsWith('temp-')}
						<!-- Sending (Hollow circle) -->
						<div class="h-4 w-4 rounded-full border-2 border-gray-400"></div>
					{:else}
						{@const otherSeenCount =
							message.seen_by?.filter((id) => id !== auth.state.user?.id).length || 0}
						{@const otherDeliveredCount =
							message.delivered_to?.filter((id) => id !== auth.state.user?.id).length || 0}

						{#if otherSeenCount > 0}
							<!-- Seen (Blue filled check or Avatar) -->
							<!-- FB style is avatar, but we'll use filled blue check for clarity across group/dm for now -->
							<svg class="h-4 w-4 text-blue-500" viewBox="0 0 24 24" fill="currentColor">
								<path
									d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
								/>
							</svg>
						{:else if otherDeliveredCount > 0}
							<!-- Delivered (Grey filled check) -->
							<svg class="h-4 w-4 text-gray-400" viewBox="0 0 24 24" fill="currentColor">
								<path
									d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
								/>
							</svg>
						{:else}
							<!-- Sent (Hollow circle with check) -->
							<svg
								class="h-4 w-4 text-gray-400"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
							>
								<circle cx="12" cy="12" r="10" />
								<path stroke-linecap="round" stroke-linejoin="round" d="M8 12l2 2 4-4" />
							</svg>
						{/if}
					{/if}
				{/if}
			</div>
			<div
				class="leading-1.5 flex flex-col border-gray-200 p-4"
				class:bg-blue-500={isMe}
				class:text-white={isMe}
				class:rounded-e-xl={!isMe}
				class:rounded-es-xl={!isMe}
				class:bg-gray-100={!isMe}
				class:rounded-s-xl={isMe}
				class:rounded-ee-xl={isMe}
			>
				<!-- Product Card for Marketplace Messages -->
				{#if message.product_id}
					<div
						class="mb-2 overflow-hidden rounded-lg border {isMe
							? 'border-blue-400 bg-blue-400/20'
							: 'border-gray-200 bg-white'}"
					>
						{#if loadingProduct}
							<div class="flex items-center gap-3 p-3">
								<div class="h-16 w-16 animate-pulse rounded bg-gray-200"></div>
								<div class="flex-1 space-y-2">
									<div class="h-4 w-3/4 animate-pulse rounded bg-gray-200"></div>
									<div class="h-3 w-1/2 animate-pulse rounded bg-gray-200"></div>
								</div>
							</div>
						{:else if linkedProduct}
							{#if onProductClick}
								<button
									type="button"
									onclick={() => onProductClick(linkedProduct!.id)}
									class="flex w-full items-start gap-3 p-3 text-left transition-all duration-200 {isMe
										? 'hover:bg-blue-600/50'
										: 'hover:bg-blue-50'} rounded-lg"
								>
									<img
										src={linkedProduct.images?.[0] || 'https://via.placeholder.com/64'}
										alt={linkedProduct.title}
										class="h-16 w-16 flex-shrink-0 rounded-lg object-cover"
									/>
									<div class="min-w-0 flex-1">
										<p class="font-semibold leading-tight {isMe ? 'text-white' : 'text-gray-900'}">
											{linkedProduct.title}
										</p>
										<p class="mt-1 text-sm font-bold {isMe ? 'text-blue-100' : 'text-blue-600'}">
											{linkedProduct.currency}
											{linkedProduct.price.toLocaleString()}
										</p>
										<span
											class="mt-1 inline-block rounded-full px-2 py-0.5 text-xs font-medium
										{linkedProduct.status === 'available'
												? 'bg-green-100 text-green-700'
												: linkedProduct.status === 'sold'
													? 'bg-red-100 text-red-600'
													: 'bg-gray-100 text-gray-600'}"
										>
											{linkedProduct.status === 'available'
												? '✓ Available'
												: linkedProduct.status === 'sold'
													? 'Sold'
													: linkedProduct.status}
										</span>
									</div>
								</button>
							{:else}
								<a
									href="/marketplace?product={linkedProduct.id}"
									class="flex items-start gap-3 p-3 transition-all duration-200 {isMe
										? 'hover:bg-blue-600/50'
										: 'hover:bg-blue-50'} rounded-lg"
								>
									<img
										src={linkedProduct.images?.[0] || 'https://via.placeholder.com/64'}
										alt={linkedProduct.title}
										class="h-16 w-16 flex-shrink-0 rounded-lg object-cover"
									/>
									<div class="min-w-0 flex-1">
										<p class="font-semibold leading-tight {isMe ? 'text-white' : 'text-gray-900'}">
											{linkedProduct.title}
										</p>
										<p class="mt-1 text-sm font-bold {isMe ? 'text-blue-100' : 'text-blue-600'}">
											{linkedProduct.currency}
											{linkedProduct.price.toLocaleString()}
										</p>
										<span
											class="mt-1 inline-block rounded-full px-2 py-0.5 text-xs font-medium
										{linkedProduct.status === 'available'
												? 'bg-green-100 text-green-700'
												: linkedProduct.status === 'sold'
													? 'bg-red-100 text-red-600'
													: 'bg-gray-100 text-gray-600'}"
										>
											{linkedProduct.status === 'available'
												? '✓ Available'
												: linkedProduct.status === 'sold'
													? 'Sold'
													: linkedProduct.status}
										</span>
									</div>
								</a>
							{/if}
						{:else}
							<div class="p-3 text-sm {isMe ? 'text-blue-200' : 'text-gray-400'}">
								Product no longer available
							</div>
						{/if}
					</div>
				{/if}

				<!-- Split media into Grid (Images/Videos) and List (Files) -->
				<!-- Split media into Grid (Images/Videos) and List (Files) -->
				<!-- Logic moved to script -->

				<!-- Media Grid -->
				{#if gridMedia.length > 0}
					<div
						class="mb-1 grid gap-0.5 overflow-hidden rounded-xl
					{gridMedia.length === 1 ? 'grid-cols-1' : 'grid-cols-2'}"
					>
						{#each gridMedia.slice(0, 4) as item, i}
							<div
								class="relative aspect-square w-full cursor-pointer
							{gridMedia.length === 3 && i === 0 ? 'col-span-2 aspect-[2/1]' : ''}
							"
								onclick={() => lightbox.open(gridMedia as any, i)}
								role="button"
								tabindex="0"
								onkeypress={(e) => e.key === 'Enter' && lightbox.open(gridMedia as any, i)}
							>
								{#if item.type === 'video'}
									<!-- svelte-ignore a11y-media-has-caption -->
									<video
										src={item.url}
										class="pointer-events-none h-full w-full bg-black object-cover"
									></video>
									<div
										class="absolute inset-0 flex items-center justify-center bg-black/10 transition-colors hover:bg-black/20"
									>
										<div class="rounded-full bg-black/50 p-3 text-white backdrop-blur-sm">
											<svg
												xmlns="http://www.w3.org/2000/svg"
												class="h-8 w-8"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
											>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													stroke-width="2"
													d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
												/>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													stroke-width="2"
													d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
												/>
											</svg>
										</div>
									</div>
								{:else}
									<img
										src={item.url}
										alt="Attachment"
										class="h-full w-full bg-gray-100 object-cover transition-opacity hover:opacity-90"
										onerror={handleImgError}
									/>
								{/if}

								<!-- Overlay for +X items -->
								{#if gridMedia.length > 4 && i === 3}
									<div
										class="absolute inset-0 flex items-center justify-center bg-black/50 text-xl font-bold text-white transition-colors hover:bg-black/60"
									>
										+{gridMedia.length - 3}
									</div>
								{/if}
							</div>
						{/each}
					</div>
				{/if}

				<!-- Attachment List -->
				{#if attachments.length > 0}
					<div class="mt-1 flex flex-col gap-1">
						{#each attachments as item, i}
							<a
								href={item.url}
								target="_blank"
								rel="noopener noreferrer"
								class="group flex items-center gap-3 rounded-xl border border-transparent bg-gray-100/80 p-3 transition-all hover:border-gray-300 hover:bg-gray-200/80 dark:bg-gray-800 dark:hover:border-gray-600 dark:hover:bg-gray-700"
							>
								<!-- Icon container -->
								<div
									class="flex h-12 w-12 items-center justify-center rounded-full bg-white text-blue-500 shadow-sm transition-transform group-hover:scale-105 dark:bg-gray-700"
								>
									<svg
										xmlns="http://www.w3.org/2000/svg"
										class="h-6 w-6"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
										/>
									</svg>
								</div>

								<!-- File Info -->
								<div class="flex min-w-0 flex-col">
									<span
										class="truncate text-sm font-semibold text-gray-900 dark:text-gray-100"
										title={item.name || item.url.split('/').pop()}
									>
										{item.name || item.url.split('/').pop()?.split('?')[0] || 'Document'}
									</span>
									<span class="text-xs font-medium text-gray-500 dark:text-gray-400">
										{(
											item.name?.split('.').pop() || item.url.split('.').pop()?.split('?')[0]
										)?.toUpperCase() || 'FILE'} · Download
									</span>
								</div>
							</a>
						{/each}
					</div>
				{/if}

				{#if message.content}
					<!-- Handle legacy single-media content in 'content' field if media_urls is empty -->
					{#if (!message.media_urls || message.media_urls.length === 0) && (message.content_type === 'image' || message.content_type === 'IMAGE')}
						<img src={message.content} alt="" class="mb-1 max-w-xs rounded-lg" />
					{:else if (!message.media_urls || message.media_urls.length === 0) && (message.content_type === 'video' || message.content_type === 'VIDEO')}
						<!-- svelte-ignore a11y-media-has-caption -->
						<video src={message.content} controls class="mb-1 max-w-xs rounded-lg"></video>
					{:else if (!message.media_urls || message.media_urls.length === 0) && (message.content_type === 'file' || message.content_type === 'FILE')}
						<a
							href={message.content}
							target="_blank"
							rel="noopener noreferrer"
							class="flex items-center space-x-2 rounded bg-white/10 p-2 text-sm font-normal underline"
						>
							<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"
								><path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13"
								/></svg
							>
							<span>Download File</span>
						</a>
					{:else}
						<p class="whitespace-pre-wrap break-words text-sm font-normal">
							{@html parseContent(message.content)}
						</p>
					{/if}
				{/if}
			</div>
			<div class="mt-1 flex items-center justify-between gap-2">
				<div class="flex items-center gap-2" class:flex-row-reverse={isMe}>
					{#if message.is_edited}
						<span class="text-xs font-normal text-gray-400">Edited</span>
					{/if}

					<!-- Display reactions using the new component -->
					<MessageReactions reactions={message.reactions || []} messageId={message.id} />
				</div>

				<!-- Message Actions (Edit/Delete/React) -->
				<MessageActions
					messageId={message.id}
					messageContent={message.content || ''}
					messageSenderId={message.sender_id}
					messageCreatedAt={message.created_at}
					{conversationId}
					{conversationKey}
					onEdited={(newContent) => (message.content = newContent)}
					on:deleted={() => dispatch('deleted', { id: message.id })}
				/>
			</div>
		</div>
	</div>
{/if}
