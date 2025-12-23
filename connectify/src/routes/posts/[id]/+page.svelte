<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { apiRequest } from '$lib/api';
	import AppHeader from '$lib/components/layout/AppHeader.svelte';
	import LeftSidebar from '$lib/components/layout/LeftSidebar.svelte';
	import RightSidebar from '$lib/components/layout/RightSidebar.svelte';
	import PostCard from '$lib/components/feed/PostCard.svelte';
	import Skeleton from '$lib/components/ui/skeleton.svelte';

	let post: any = null;
	let loading = true;
	let error: string | null = null;
	let postId: string;

	// Reactive subscription to page params to handle navigation changes if any
	$: {
		postId = $page.params.id ?? '';
	}

	onMount(async () => {
		try {
			post = await apiRequest('GET', `/posts/${postId}`);
		} catch (e: any) {
			error = e.message || 'Failed to load post';
		} finally {
			loading = false;
		}
	});
</script>

<div class="flex min-h-screen flex-col bg-gray-100 font-sans">
	<AppHeader />

	<div class="grid flex-1 grid-cols-[auto_1fr_auto] pt-14">
		<!-- Left Sidebar -->
		<aside
			class="sticky top-14 hidden h-[calc(100vh-56px)] w-64 overflow-y-auto bg-white p-4 shadow-md md:block lg:w-72"
		>
			<LeftSidebar />
		</aside>

		<!-- Main Content -->
		<main class="overflow-y-auto p-4">
			<div class="mx-auto flex max-w-2xl flex-col items-center space-y-6">
				{#if loading}
					<div class="w-full max-w-2xl space-y-3 rounded-lg bg-white p-4 shadow-md">
						<div class="flex items-center space-x-3">
							<Skeleton class="h-10 w-10 rounded-full" />
							<div class="space-y-2">
								<Skeleton class="h-4 w-[200px]" />
								<Skeleton class="h-4 w-[150px]" />
							</div>
						</div>
						<div class="space-y-2">
							<Skeleton class="h-4 w-full" />
							<Skeleton class="h-4 w-[80%]" />
						</div>
					</div>
				{:else if error}
					<div class="text-center text-red-500">
						<p>{error}</p>
						<a href="/dashboard" class="text-blue-600 hover:underline">Back to Feed</a>
					</div>
				{:else if post}
					<PostCard {post} isDetailedView={true} />
				{/if}
			</div>
		</main>

		<!-- Right Sidebar -->
		<aside
			class="sticky top-14 hidden h-[calc(100vh-56px)] w-72 overflow-y-auto bg-white p-4 shadow-md lg:block"
		>
			<RightSidebar />
		</aside>
	</div>
</div>
