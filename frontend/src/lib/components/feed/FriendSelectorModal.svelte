<script lang="ts">
	import { apiRequest } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import { X, Search, Check } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth.svelte';

	let {
		exclude = false,
		initialSelected = [],
		onSave,
		onClose
	} = $props<{
		exclude?: boolean;
		initialSelected?: string[];
		onSave: (selectedIds: string[]) => void;
		onClose: () => void;
	}>();

	let friends = $state<any[]>([]);
	let loading = $state(true);
	let searchQuery = $state('');
	let selected = $state<Set<string>>(new Set(initialSelected));

	let currentUserId = $derived(auth.state.user?.id);

	onMount(async () => {
		if (!currentUserId) return;
		await fetchFriends();
	});

	async function fetchFriends() {
		try {
			// Fetch accepted friendships
			const response = await apiRequest('GET', '/friendships?status=accepted', undefined, true);
			const friendships = response.data || [];

			// Resolve friend details
			const promises = friendships.map(async (f: any) => {
				const friendId = f.requester_id === currentUserId ? f.receiver_id : f.requester_id;
				// In a real app with many friends, we'd want a bulk fetch or backend expansion.
				// For now, fetching individual details (cache this in real app)
				const friend = await apiRequest('GET', `/users/${friendId}`, undefined, true);
				return {
					id: friend.id,
					username: friend.username,
					full_name: friend.full_name,
					avatar: friend.avatar
				};
			});

			friends = await Promise.all(promises);
		} catch (error) {
			console.error('Failed to fetch friends:', error);
		} finally {
			loading = false;
		}
	}

	function toggleSelection(id: string) {
		if (selected.has(id)) {
			selected.delete(id);
		} else {
			selected.add(id);
		}
		// Trigger reactivity manually if needed for Sets in Svelte 5,
		// but reassigning works best to be safe:
		selected = new Set(selected);
	}

	function handleSave() {
		onSave(Array.from(selected));
	}

	let filteredFriends = $derived(
		friends.filter(
			(f) =>
				f.username.toLowerCase().includes(searchQuery.toLowerCase()) ||
				(f.full_name && f.full_name.toLowerCase().includes(searchQuery.toLowerCase()))
		)
	);
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4 backdrop-blur-sm">
	<div class="bg-card w-full max-w-md overflow-hidden rounded-xl border border-white/10 shadow-2xl">
		<!-- Header -->
		<div class="flex items-center justify-between border-b border-white/10 p-4">
			<h3 class="text-lg font-semibold text-white">
				{exclude ? 'Hide story from...' : 'Share story with...'}
			</h3>
			<button class="text-white/70 hover:text-white" onclick={onClose}>
				<X size={20} />
			</button>
		</div>

		<!-- Search -->
		<div class="p-4 pb-2">
			<div class="bg-secondary/50 flex items-center rounded-lg px-3 py-2">
				<Search size={16} class="text-muted-foreground mr-2" />
				<input
					type="text"
					placeholder="Search friends"
					bind:value={searchQuery}
					class="text-foreground w-full bg-transparent text-sm focus:outline-none"
				/>
			</div>
		</div>

		<!-- List -->
		<div class="h-[300px] overflow-y-auto p-4 pt-2">
			{#if loading}
				<div class="text-muted-foreground py-8 text-center text-sm">Loading friends...</div>
			{:else if filteredFriends.length === 0}
				<div class="text-muted-foreground py-8 text-center text-sm">No friends found</div>
			{:else}
				<div class="space-y-2">
					{#each filteredFriends as friend (friend.id)}
						<button
							class="hover:bg-accent/50 flex w-full items-center justify-between rounded-lg p-2 transition-colors"
							onclick={() => toggleSelection(friend.id)}
						>
							<div class="flex items-center gap-3">
								<img
									src={friend.avatar || 'https://github.com/shadcn.png'}
									alt={friend.username}
									class="h-10 w-10 rounded-full object-cover"
								/>
								<div class="text-left">
									<div class="font-semibold text-white">{friend.username}</div>
									{#if friend.full_name}
										<div class="text-muted-foreground text-xs">{friend.full_name}</div>
									{/if}
								</div>
							</div>

							<div
								class="flex h-5 w-5 items-center justify-center rounded-full border transition-colors {selected.has(
									friend.id
								)
									? 'bg-primary border-primary'
									: 'border-white/50'}"
							>
								{#if selected.has(friend.id)}
									<Check size={12} class="text-black" strokeWidth={3} />
								{/if}
							</div>
						</button>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Footer -->
		<div class="flex justify-end gap-2 border-t border-white/10 p-4">
			<Button variant="ghost" onclick={onClose}>Cancel</Button>
			<Button onclick={handleSave}>Done</Button>
		</div>
	</div>
</div>
