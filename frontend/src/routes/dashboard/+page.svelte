<script lang="ts">
	import { onMount } from 'svelte';
	// AppHeader is now in +layout.svelte
	import LeftSidebar from '$lib/components/layout/LeftSidebar.svelte';
	import RightSidebar from '$lib/components/layout/RightSidebar.svelte';
	import PostCreator from '$lib/components/feed/PostCreator.svelte';
	import PostCard from '$lib/components/feed/PostCard.svelte';
	import StoryReel from '$lib/components/feed/StoryReel.svelte';
	import { apiRequest } from '$lib/api';
	import { intersect } from '$lib/actions/intersect';
	import { websocketMessages } from '$lib/websocket';
	import Skeleton from '$lib/components/ui/skeleton/Skeleton.svelte';
	import MediaViewer from '$lib/components/ui/MediaViewer.svelte';

	// State
	let posts = $state<any[]>([]);
	let loadingPosts = $state(true);
	let errorPosts = $state<string | null>(null);
	let currentPage = $state(1);
	let hasMore = $state(true);

	// Media Viewer State
	let mediaViewerOpen = $state(false);
	let mediaViewerItems = $state<any[]>([]);
	let mediaViewerIndex = $state(0);

	function openMediaViewer(items: any[], index: number) {
		mediaViewerItems = items;
		mediaViewerIndex = index;
		mediaViewerOpen = true;
	}

	async function fetchPosts(page = 1) {
		if (!hasMore && page > 1) return;
		loadingPosts = true;
		errorPosts = null;
		try {
			const response = await apiRequest('GET', `/posts?page=${page}&limit=10`);
			if (response.posts && response.posts.length > 0) {
				posts = page === 1 ? response.posts : [...posts, ...response.posts];
				currentPage = page;
			} else {
				hasMore = false;
			}
		} catch (err: any) {
			errorPosts = err.message || 'Failed to load posts.';
			console.error('Fetch posts error:', err);
		} finally {
			loadingPosts = false;
		}
	}

	function handlePostCreated(event: CustomEvent) {
		const newPost = event.detail;
		// Add new post to the top of the list if it's not already there
		if (!posts.some((p) => p.id === newPost.id)) {
			posts = [newPost, ...posts];
		}
	}

	function handlePostUpdated(event: CustomEvent) {
		const updatedPost = event.detail;
		posts = posts.map((p) => (p.id === updatedPost.id ? updatedPost : p));
	}

	function handlePostDeleted(event: CustomEvent) {
		const deletedPost = event.detail;
		posts = posts.filter((p) => p.id !== deletedPost.id);
	}

	function loadMorePosts() {
		if (loadingPosts || !hasMore) return;
		fetchPosts(currentPage + 1);
	}

	onMount(() => {
		fetchPosts(1);

		const unsubscribe = websocketMessages.subscribe((event) => {
			if (event && event.type === 'PostCreated') {
				handlePostCreated({ detail: event.data } as CustomEvent);
			} else if (event && event.type === 'PostUpdated') {
				handlePostUpdated({ detail: event.data } as CustomEvent);
			} else if (event && event.type === 'PostDeleted') {
				handlePostDeleted({ detail: event.data } as CustomEvent);
			}
		});

		return () => {
			unsubscribe();
		};
	});
</script>

<MediaViewer
	open={mediaViewerOpen}
	media={mediaViewerItems}
	initialIndex={mediaViewerIndex}
	onClose={() => (mediaViewerOpen = false)}
/>

<div class="flex min-h-screen justify-center font-sans">
	<div
		class="grid w-full max-w-[1920px] grid-cols-1 md:grid-cols-[280px_1fr] lg:grid-cols-[360px_680px_360px] xl:justify-center"
	>
		<!-- Left Sidebar -->
		<aside
			class="sticky top-14 hidden h-[calc(100vh-56px)] overflow-y-auto px-4 py-6 hover:overflow-y-auto md:block"
		>
			<LeftSidebar />
		</aside>

		<!-- Main Content Area (News Feed) -->
		<main class="flex min-w-0 flex-1 flex-col items-center px-4 pb-10 pt-6">
			<div class="w-full max-w-[590px] space-y-5">
				<!-- Stories -->
				<StoryReel />

				<!-- Post Creator -->
				<div class="relative z-30 w-full">
					<PostCreator on:postCreated={handlePostCreated} />
				</div>

				<!-- Feed -->
				{#if loadingPosts && posts.length === 0}
					<div class="w-full space-y-5">
						{#each Array(3) as _, i (i)}
							<div class="glass-card w-full space-y-4 p-4">
								<div class="flex items-center space-x-3">
									<Skeleton class="bg-primary/10 h-10 w-10 rounded-full" />
									<div class="space-y-2">
										<Skeleton class="bg-primary/10 h-4 w-[200px]" />
										<Skeleton class="bg-primary/10 h-3 w-[150px]" />
									</div>
								</div>
								<div class="space-y-2 py-2">
									<Skeleton class="bg-primary/10 h-4 w-full" />
									<Skeleton class="bg-primary/10 h-4 w-[80%]" />
								</div>
								<div class="flex items-center justify-between pt-2">
									<Skeleton class="bg-primary/10 h-8 w-24" />
									<Skeleton class="bg-primary/10 h-8 w-24" />
								</div>
							</div>
						{/each}
					</div>
				{:else if !loadingPosts && posts.length === 0 && !errorPosts}
					<div class="glass-panel w-full rounded-xl p-8 text-center">
						<p class="text-muted-foreground">
							No posts to display. Be the first to share something!
						</p>
					</div>
				{:else}
					<div class="w-full space-y-5">
						{#each posts as post (post.id)}
							<PostCard
								{post}
								on:viewMedia={(e) => openMediaViewer(e.detail.media, e.detail.index)}
							/>
						{/each}
					</div>
				{/if}

				{#if hasMore && !loadingPosts}
					<div use:intersect on:intersect={loadMorePosts} class="h-10 w-full"></div>
				{/if}

				{#if loadingPosts && posts.length > 0}
					<div class="flex justify-center p-4">
						<div
							class="border-primary h-8 w-8 animate-spin rounded-full border-4 border-t-transparent"
						></div>
					</div>
				{/if}

				{#if !hasMore && posts.length > 0}
					<div class="flex flex-col items-center space-y-2 py-8 text-center">
						<div class="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-full">
							<span class="text-xl">✓</span>
						</div>
						<p class="text-muted-foreground">You're all caught up!</p>
					</div>
				{/if}

				{#if errorPosts}
					<div class="glass-panel rounded-xl border-red-500/20 bg-red-500/10 p-4 text-red-500">
						<p class="font-medium">Error loading posts</p>
						<p class="text-sm opacity-80">{errorPosts}</p>
					</div>
				{/if}
			</div>
		</main>

		<!-- Right Sidebar -->
		<aside class="sticky top-14 hidden h-[calc(100vh-56px)] overflow-y-auto px-4 py-6 lg:block">
			<RightSidebar />
		</aside>
	</div>
</div>
