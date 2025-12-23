<script lang="ts">
	import { getContext } from 'svelte';
	import type { Writable } from 'svelte/store';
	import { page } from '$app/stores';
	import { getPosts, type Community } from '$lib/api';
	import PostCard from '$lib/components/feed/PostCard.svelte';
	import PostCreator from '$lib/components/feed/PostCreator.svelte';
	import CommunitySidebar from '$lib/components/community/CommunitySidebar.svelte';
	import type { Post } from '$lib/types';

	// Get context from layout
	const communityStore = getContext<Writable<Community>>('community');
	let community = $derived($communityStore);
	let id = $derived($page.params.id);

	let posts = $state<Post[]>([]);
	let loading = $state(true);
	let error = $state('');

	async function loadFeed() {
		loading = true;
		try {
			// Fetch active posts for this community
			const res = await getPosts({ community_id: id, page: 1, limit: 20 }); // Backend defaults to status=active
			posts = res.posts || [];
		} catch (e: any) {
			console.error('Failed to load feed:', e);
			error = e.message;
		} finally {
			loading = false;
		}
	}

	function handlePostCreated(event: CustomEvent<Post>) {
		const newPost = event.detail;
		// If post is pending, maybe show a toast instead of adding to list instantly?
		// For now, if "pending", we shouldn't add it to "posts" array if this view shows active only.
		if (newPost.status === 'pending') {
			// TODO: Show toast "Post submitted for approval"
			alert('Post submitted for approval!');
		} else {
			posts = [newPost, ...posts];
		}
	}

	// Reload feed if community ID changes
	$effect(() => {
		if (id) {
			loadFeed();
		}
	});
</script>

<div class="grid grid-cols-1 items-start gap-6 lg:grid-cols-3">
	<!-- Left: Feed (cols-2) -->
	<div class="space-y-6 lg:col-span-2">
		{#if error}
			<div class="rounded-xl border border-red-200 bg-red-50 p-4 text-red-600 shadow-sm">
				{error}
			</div>
		{/if}

		<!-- Create Post (if member) -->
		{#if community?.is_member}
			<div class="relative z-10">
				<!-- Pass special prop to PostCreator to indicate approval might be needed? 
				     PostCreator just calls API. 
					 We might want to pass "requireApproval" to UI so button says "Submit"? -->
				<PostCreator communityId={id} on:postCreated={handlePostCreated} />
			</div>
		{/if}

		<!-- Posts Feed -->
		{#if loading}
			<div class="space-y-4">
				{#each Array(3) as _}
					<div class="h-64 animate-pulse rounded-xl border border-gray-100 bg-white shadow-sm" />
				{/each}
			</div>
		{:else if posts.length > 0}
			<div class="space-y-6">
				{#each posts as post (post.id)}
					<PostCard {post} />
				{/each}
			</div>
		{:else}
			<div
				class="rounded-xl bg-white p-12 text-center text-gray-500 shadow-sm ring-1 ring-gray-200"
			>
				<p class="text-lg">No posts yet.</p>
				<p class="text-sm">Be the first to share something with the group!</p>
			</div>
		{/if}
	</div>

	<!-- Right: Sidebar Widgets -->
	<div class="hidden space-y-6 lg:sticky lg:top-24 lg:block">
		{#if community}
			<CommunitySidebar {community} isAdmin={community.is_admin} />
		{/if}
	</div>
</div>
