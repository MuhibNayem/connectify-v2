<script lang="ts">
	import { apiRequest } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Check } from '@lucide/svelte';
	import { onMount, onDestroy } from 'svelte';

	let {
		open = $bindable(false),
		userId,
		onSelect
	} = $props<{
		open: boolean;
		userId: string;
		onSelect: (selectedMedia: any[]) => void;
	}>();

	let media = $state<any[]>([]);
	let loading = $state(false);
	let selectedUrls = $state<Set<string>>(new Set());
	const LIMIT = 50;
	let page = $state(1);
	let hasMore = $state(false);
	let sentinel = $state<HTMLElement | null>(null);
	let observer: IntersectionObserver;

	async function fetchTimelineMedia(reset = false) {
		if (loading) return;
		loading = true;
		try {
			// 1. Get user albums to find 'timeline' album
			const albums = await apiRequest('GET', `/users/${userId}/albums`);
			const timelineAlbum = albums.find((a: any) => a.type === 'timeline');

			if (timelineAlbum) {
				if (reset) {
					media = [];
					page = 1;
					hasMore = true;
				}

				// 2. Fetch media from timeline album, filtered by type=image
				const response = await apiRequest(
					'GET',
					`/albums/${timelineAlbum.id}/media?limit=${LIMIT}&page=${page}&type=image`
				);

				const items = response.media || [];
				if (items && items.length > 0) {
					// Prepare new items checking for duplicates based on URL
					const existingUrls = new Set(media.map((m) => m.url));
					const uniqueNewItems: any[] = [];

					for (const item of items) {
						if (!existingUrls.has(item.url)) {
							uniqueNewItems.push(item);
							existingUrls.add(item.url);
						}
					}

					media = [...media, ...uniqueNewItems];

					const currentPage = response.page;
					const totalItems = response.total;
					const limit = response.limit;

					const isLastPage = currentPage * limit >= totalItems;
					hasMore = !isLastPage;
					if (hasMore) {
						page++;
					}
				} else {
					hasMore = false;
				}
			}
		} catch (err) {
			console.error('Failed to fetch timeline media', err);
			hasMore = false;
		} finally {
			loading = false;
		}
	}

	function toggleSelection(item: any) {
		// Create a new Set to trigger reactivity
		const newSelectedUrls = new Set(selectedUrls);
		if (newSelectedUrls.has(item.url)) {
			newSelectedUrls.delete(item.url);
		} else {
			newSelectedUrls.add(item.url);
		}
		selectedUrls = newSelectedUrls;
	}

	function handleConfirm() {
		const selectedItems = media.filter((item) => selectedUrls.has(item.url));
		onSelect(selectedItems);
		open = false;
		selectedUrls = new Set();
	}

	function setupIntersectionObserver() {
		if (observer) observer.disconnect();

		observer = new IntersectionObserver(
			(entries) => {
				if (entries[0].isIntersecting && hasMore && !loading) {
					fetchTimelineMedia();
				}
			},
			{
				rootMargin: '200px', // Load before reaching the very bottom
				threshold: 0.1
			}
		);

		if (sentinel) {
			observer.observe(sentinel);
		}
	}

	onMount(() => {
		fetchTimelineMedia(true);
		selectedUrls = new Set();
	});

	onDestroy(() => {
		if (observer) observer.disconnect();
	});

	// Re-attach observer when sentinel becomes available (e.g. after loading initial data)
	$effect(() => {
		if (sentinel && !loading && hasMore) {
			setupIntersectionObserver();
		}
	});
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="max-w-3xl">
		<Dialog.Header>
			<Dialog.Title>Select Photos</Dialog.Title>
			<Dialog.Description>Choose photos from your timeline to add to this album.</Dialog.Description
			>
		</Dialog.Header>

		<div class="h-[60vh] overflow-y-auto p-1">
			{#if media.length === 0 && !loading}
				<div class="text-muted-foreground flex h-full items-center justify-center">
					No photos found in your timeline.
				</div>
			{:else}
				<div class="grid grid-cols-3 gap-2 sm:grid-cols-4 md:grid-cols-5">
					{#each media as item (item.url)}
						<button
							class="relative aspect-square cursor-pointer overflow-hidden rounded-lg border-2 transition-all focus:outline-none {selectedUrls.has(
								item.url
							)
								? 'border-primary ring-primary ring-2 ring-offset-2'
								: 'border-transparent'}"
							onclick={() => toggleSelection(item)}
						>
							{#if item.type === 'video'}
								<video src={item.url} class="h-full w-full object-cover">
									<track kind="captions" />
								</video>
								<div class="absolute inset-0 flex items-center justify-center bg-black/20">
									<div class="rounded-full bg-black/50 p-1">▶</div>
								</div>
							{:else}
								<img src={item.url} alt="Media" class="h-full w-full object-cover" />
							{/if}

							{#if selectedUrls.has(item.url)}
								<div
									class="bg-primary absolute right-1 top-1 flex h-6 w-6 items-center justify-center rounded-full text-white shadow-sm"
								>
									<Check size={14} />
								</div>
							{/if}
						</button>
					{/each}
				</div>

				{#if loading}
					<div class="flex justify-center py-4">
						<div
							class="border-primary h-6 w-6 animate-spin rounded-full border-2 border-t-transparent"
						></div>
					</div>
				{/if}

				<!-- Sentinel for infinite scroll -->
				<div bind:this={sentinel} class="h-4 w-full"></div>
			{/if}
		</div>

		<Dialog.Footer>
			<Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
			<Button onclick={handleConfirm} disabled={selectedUrls.size === 0}>
				Add {selectedUrls.size} Photo{selectedUrls.size !== 1 ? 's' : ''}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
