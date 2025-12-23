<script lang="ts">
	import { onMount, setContext } from 'svelte';
	import { writable } from 'svelte/store';
	import { page } from '$app/stores';
	import { getCommunity, type Community } from '$lib/api';
	import CommunityHeader from '$lib/components/community/CommunityHeader.svelte';
	import { fade } from 'svelte/transition';

	let id = $derived($page.params.id);
	let community = writable<Community | null>(null);
	let loading = writable(true);
	let error = writable('');

	setContext('community', community);
	setContext('communityLoading', loading);

	async function loadCommunity() {
		$loading = true;
		try {
			const res = await getCommunity(id);
			$community = res;
		} catch (e: any) {
			console.error('Failed to load community:', e);
			$error = e.message;
		} finally {
			$loading = false;
		}
	}

	// Use a reactive block to trigger load, guarded by ID check
	// Actually, onMount is safest for initial, but navigation?
	// Using key block in parent or just reactive loading here.

	// Simple approach:
	// Use a reactive block to trigger load, guarded by ID check
	$effect(() => {
		if (id && (!$community || $community.id !== id)) {
			loadCommunity();
		}
	});
</script>

{#if $loading}
	<div class="flex h-96 w-full items-center justify-center">
		<div class="border-primary h-12 w-12 animate-spin rounded-full border-b-2"></div>
	</div>
{:else if $error}
	<div class="p-10 text-center text-red-500">
		Error loading community: {$error}
	</div>
{:else if $community}
	<div class="min-h-screen bg-gray-50" in:fade>
		<CommunityHeader
			community={$community}
			isMember={$community.is_member}
			isAdmin={$community.is_admin}
			isPending={$community.is_pending}
		/>

		<main class="mx-auto max-w-7xl px-4 pb-10 sm:px-6 lg:px-8">
			<slot />
		</main>
	</div>
{/if}
