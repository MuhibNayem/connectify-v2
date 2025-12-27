<script lang="ts">
	import { apiRequest } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import { onMount } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { UserPlus, UserX } from '@lucide/svelte';

	let { limit = undefined } = $props();

	interface FriendRequest {
		id: string;
		requester_id: string;
		receiver_id: string;
		status: 'pending' | 'accepted' | 'rejected';
		requester_username?: string;
		receiver_username?: string;
		requester_avatar?: string;
		receiver_avatar?: string;
	}

	interface UserDetails {
		id: string;
		username: string;
		avatar?: string;
	}

	let incomingRequests = $state<FriendRequest[]>([]);
	let outgoingRequests = $state<FriendRequest[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	let currentUserId = $derived(auth.state.user?.id);

	// Derived filtered lists based on limit
	let displayedIncoming = $derived(limit ? incomingRequests.slice(0, limit) : incomingRequests);
	let displayedOutgoing = $derived(limit ? outgoingRequests.slice(0, limit) : outgoingRequests);

	onMount(async () => {
		if (!currentUserId) {
			error = 'User not authenticated.';
			loading = false;
			return;
		}
		await fetchRequests();
	});

	async function fetchRequests() {
		loading = true;
		error = null;
		try {
			const response = await apiRequest('GET', '/friendships?status=pending', undefined, true);
			const allPendingRequests: FriendRequest[] = response.data ?? [];

			const fetchedIncoming: FriendRequest[] = [];
			const fetchedOutgoing: FriendRequest[] = [];

			for (const req of allPendingRequests) {
				if (req.receiver_id === currentUserId) {
					const requesterDetails: UserDetails = await apiRequest(
						'GET',
						`/users/${req.requester_id}`,
						undefined,
						true
					);
					fetchedIncoming.push({
						...req,
						requester_username: requesterDetails.username,
						requester_avatar: requesterDetails.avatar
					});
				} else if (req.requester_id === currentUserId) {
					const receiverDetails: UserDetails = await apiRequest(
						'GET',
						`/users/${req.receiver_id}`,
						undefined,
						true
					);
					fetchedOutgoing.push({
						...req,
						receiver_username: receiverDetails.username,
						receiver_avatar: receiverDetails.avatar
					});
				}
			}

			incomingRequests = fetchedIncoming;
			outgoingRequests = fetchedOutgoing;
		} catch (e: any) {
			error = e.message || 'Failed to load friend requests.';
			console.error(e);
		} finally {
			loading = false;
		}
	}

	async function handleRespondToRequest(requestId: string, accept: boolean) {
		try {
			await apiRequest(
				'POST',
				`/friendships/requests/${requestId}/respond`,
				{ friendship_id: requestId, accept },
				true
			);
			// Optimistic update
			incomingRequests = incomingRequests.filter((r) => r.id !== requestId);
			// await fetchRequests(); // Refreshing full list might be overkill if we just remove it
		} catch (e: any) {
			alert(`Failed to respond to request: ${e.message}`);
			console.error(e);
		}
	}

	async function handleCancelRequest(requestId: string) {
		if (confirm('Are you sure you want to cancel this request?')) {
			try {
				await apiRequest('DELETE', `/friendships/${requestId}`, undefined, true);
				outgoingRequests = outgoingRequests.filter((r) => r.id !== requestId);
			} catch (e: any) {
				alert(`Failed to cancel request: ${e.message}`);
				console.error(e);
			}
		}
	}
</script>

<div class="space-y-8">
	{#if loading}
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
			{#each Array(limit || 4) as _}
				<div class="glass-card h-64 animate-pulse rounded-xl"></div>
			{/each}
		</div>
	{:else if error}
		<div class="glass-panel p-4 text-center text-red-500">{error}</div>
	{:else}
		<!-- Incoming Requests -->
		{#if displayedIncoming.length > 0}
			<div>
				{#if !limit}<h3 class="mb-4 text-xl font-bold">Friend Requests</h3>{/if}
				<div
					class="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5"
				>
					{#each displayedIncoming as request (request.id)}
						<div class="glass-card group flex flex-col overflow-hidden rounded-xl">
							<!-- Top Image Area (Placeholder pattern or avatar cover) -->
							<div class="from-primary/20 relative h-32 bg-gradient-to-br to-purple-500/20">
								{#if request.requester_avatar}
									<img
										src={request.requester_avatar}
										alt={request.requester_username}
										class="h-full w-full object-cover opacity-80"
									/>
								{:else}
									<div class="flex h-full w-full items-center justify-center">
										<UserPlus size={48} class="text-white/20" />
									</div>
								{/if}
							</div>

							<!-- Content -->
							<div class="flex flex-1 flex-col p-3">
								<h4 class="mb-1 truncate text-lg font-bold">{request.requester_username}</h4>
								<p class="text-muted-foreground mb-4 text-xs">12 mutual friends</p>
								<!-- Mock mutuals -->

								<div class="mt-auto space-y-2">
									<Button class="w-full" onclick={() => handleRespondToRequest(request.id, true)}
										>Confirm</Button
									>
									<Button
										variant="ghost"
										class="w-full bg-white/5 hover:bg-white/10"
										onclick={() => handleRespondToRequest(request.id, false)}>Delete</Button
									>
								</div>
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		{#if displayedIncoming.length === 0 && !loading && !displayedOutgoing.length}
			<div class="py-10 text-center opacity-60">
				<p>No new friend requests.</p>
			</div>
		{/if}

		<!-- Outgoing Requests (Only show if not limited or separate section?) -->
		{#if !limit && displayedOutgoing.length > 0}
			<div class="mt-8">
				<h3 class="mb-4 text-xl font-bold">Sent Requests</h3>
				<div
					class="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5"
				>
					{#each displayedOutgoing as request (request.id)}
						<div class="glass-card flex flex-col overflow-hidden rounded-xl">
							<div class="relative h-32 bg-gradient-to-br from-orange-400/20 to-red-500/20">
								{#if request.receiver_avatar}
									<img
										src={request.receiver_avatar}
										alt={request.receiver_username}
										class="h-full w-full object-cover opacity-80"
									/>
								{:else}
									<div class="flex h-full w-full items-center justify-center">
										<UserPlus size={48} class="text-white/20" />
									</div>
								{/if}
							</div>
							<div class="flex flex-1 flex-col p-3">
								<h4 class="mb-1 truncate text-lg font-bold">{request.receiver_username}</h4>
								<Button
									variant="secondary"
									class="mt-auto w-full"
									onclick={() => handleCancelRequest(request.id)}>Cancel Request</Button
								>
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	{/if}
</div>
