<script lang="ts">
	import { apiRequest } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import { onMount } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { presenceStore } from '$lib/stores/presence';
	import { MoreHorizontal, UserMinus } from '@lucide/svelte';

	let { limit = undefined } = $props();

	interface FriendUser {
		id: string;
		username: string;
		avatar?: string;
	}

	interface Friendship {
		id: string;
		requester_id: string;
		receiver_id: string;
		status: 'pending' | 'accepted' | 'rejected';
	}

	let friends = $state<FriendUser[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	let currentUserId = $derived(auth.state.user?.id);
	let displayedFriends = $derived(limit ? friends.slice(0, limit) : friends);

	onMount(async () => {
		if (!currentUserId) {
			error = 'User not authenticated.';
			loading = false;
			return;
		}
		await fetchFriends();
	});

	async function fetchFriends() {
		loading = true;
		error = null;
		try {
			const response = await apiRequest('GET', '/friendships?status=accepted', undefined, true);
			const acceptedFriendships: Friendship[] = response.data;

			const friendUserPromises = (acceptedFriendships ?? []).map(async (friendship) => {
				const friendId =
					friendship.requester_id === currentUserId
						? friendship.receiver_id
						: friendship.requester_id;
				const friendDetails = await apiRequest('GET', `/users/${friendId}`, undefined, true);
				return {
					id: friendDetails.id,
					username: friendDetails.username,
					avatar: friendDetails.avatar
				};
			});

			friends = await Promise.all(friendUserPromises);
		} catch (e: any) {
			error = e.message || 'Failed to load friend list.';
			console.error(e);
		} finally {
			loading = false;
		}
	}

	async function handleUnfriend(friendId: string) {
		if (
			confirm(
				`Are you sure you want to unfriend ${friends.find((f) => f.id === friendId)?.username}?`
			)
		) {
			try {
				await apiRequest('DELETE', `/friendships/${friendId}`, undefined, true);
				friends = friends.filter((f) => f.id !== friendId);
			} catch (e: any) {
				alert(`Failed to unfriend: ${e.message}`);
				console.error(e);
			}
		}
	}
</script>

<div class="space-y-4">
	{#if loading}
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
			{#each Array(limit || 8) as _}
				<div class="glass-card h-24 animate-pulse rounded-xl"></div>
			{/each}
		</div>
	{:else if error}
		<div class="glass-panel p-4 text-center text-red-500">{error}</div>
	{:else if friends.length === 0}
		<div class="glass-panel text-muted-foreground p-8 text-center">No friends to display.</div>
	{:else}
		<!-- Correct Grid Loop -->
		<div class="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
			{#each displayedFriends as friend (friend.id)}
				<div class="glass-card group flex items-center justify-between rounded-xl p-4">
					<div class="flex items-center space-x-4 overflow-hidden">
						<div class="relative h-16 w-16 flex-shrink-0 overflow-hidden rounded-full bg-white/10">
							{#if friend.avatar}
								<img src={friend.avatar} alt={friend.username} class="h-full w-full object-cover" />
							{:else}
								<div class="flex h-full w-full items-center justify-center text-xl font-bold">
									{friend.username[0]}
								</div>
							{/if}
							{#if $presenceStore[friend.id]?.status === 'online'}
								<div
									class="absolute bottom-0 right-0 h-4 w-4 rounded-full border-2 border-black bg-green-500"
								></div>
							{/if}
						</div>
						<div class="min-w-0">
							<h3 class="truncate text-lg font-bold">{friend.username}</h3>
							<button class="text-muted-foreground text-xs hover:underline"
								>12 Mutual Friends</button
							>
						</div>
					</div>

					<Button
						variant="ghost"
						size="icon"
						class="text-muted-foreground hover:bg-red-500/10 hover:text-red-500"
						onclick={() => handleUnfriend(friend.id)}
					>
						<UserMinus size={20} />
					</Button>
				</div>
			{/each}
		</div>
	{/if}
</div>
