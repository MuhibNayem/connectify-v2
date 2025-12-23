<script lang="ts">
	import { onMount } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { X, Search, Check, Loader2, Send, Users, Star, UserCheck } from '@lucide/svelte';
	import { getFriends, inviteFriendsToEvent, getEventAttendees } from '$lib/api';
	import type { User } from '$lib/types';

	let {
		eventId,
		eventTitle,
		open = $bindable(false),
		onClose
	}: {
		eventId: string;
		eventTitle: string;
		open?: boolean;
		onClose?: () => void;
	} = $props();

	interface FriendWithStatus extends User {
		eventStatus?: 'going' | 'interested' | 'invited' | null;
	}

	let friends: FriendWithStatus[] = $state([]);
	let selectedIds: Set<string> = $state(new Set());
	let searchQuery = $state('');
	let loading = $state(false);
	let sending = $state(false);
	let message = $state('');

	// Track already attending/interested user IDs
	let attendingUserIds = $state<Set<string>>(new Set());

	onMount(async () => {
		loading = true;
		try {
			// Fetch friends and attendees in parallel
			const [friendsList, attendeesRes] = await Promise.all([
				getFriends(),
				getEventAttendees(eventId, undefined, 1, 500) // Get all attendees
			]);

			// Build set of user IDs who are already going/interested
			const attendees = attendeesRes.attendees || [];
			const statusMap = new Map<string, string>();

			attendees.forEach((a: any) => {
				if (a.status === 'going' || a.status === 'interested') {
					attendingUserIds.add(a.user.id);
					statusMap.set(a.user.id, a.status);
				}
			});

			// Mark friends with their event status
			friends = friendsList.map((f: User) => ({
				...f,
				eventStatus: (statusMap.get(f.id) as 'going' | 'interested' | null) || null
			}));
		} catch (err) {
			console.error('Failed to load data:', err);
		} finally {
			loading = false;
		}
	});

	// Filter friends: show all but disable already attending
	let filteredFriends = $derived(
		friends.filter(
			(f) =>
				f.full_name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
				f.username?.toLowerCase().includes(searchQuery.toLowerCase())
		)
	);

	// Friends available for invitation (not already attending)
	let invitableFriends = $derived(filteredFriends.filter((f) => !f.eventStatus));

	function toggleSelect(id: string) {
		// Don't allow selecting already attending users
		const friend = friends.find((f) => f.id === id);
		if (friend?.eventStatus) return;

		if (selectedIds.has(id)) {
			selectedIds.delete(id);
		} else {
			selectedIds.add(id);
		}
		selectedIds = new Set(selectedIds);
	}

	function selectAll() {
		if (selectedIds.size === invitableFriends.length) {
			selectedIds.clear();
		} else {
			invitableFriends.forEach((f) => selectedIds.add(f.id));
		}
		selectedIds = new Set(selectedIds);
	}

	async function sendInvitations() {
		if (selectedIds.size === 0) return;

		sending = true;
		try {
			await inviteFriendsToEvent(eventId, Array.from(selectedIds), message || undefined);
			handleClose();
		} catch (err) {
			console.error('Failed to send invitations:', err);
			alert('Failed to send invitations');
		} finally {
			sending = false;
		}
	}

	function handleClose() {
		open = false;
		selectedIds.clear();
		message = '';
		searchQuery = '';
		onClose?.();
	}

	function getStatusLabel(status: string | null | undefined): string {
		switch (status) {
			case 'going':
				return 'Going';
			case 'interested':
				return 'Interested';
			default:
				return '';
		}
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
		onclick={handleClose}
	>
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="bg-card mx-4 w-full max-w-md overflow-hidden rounded-2xl border border-white/10 shadow-2xl"
			onclick={(e) => e.stopPropagation()}
		>
			<!-- Header -->
			<div class="flex items-center justify-between border-b border-white/10 p-4">
				<div>
					<h2 class="text-lg font-bold">Invite Friends</h2>
					<p class="text-muted-foreground text-sm">to {eventTitle}</p>
				</div>
				<button
					class="text-muted-foreground hover:text-foreground rounded-full p-1 transition-colors"
					onclick={handleClose}
				>
					<X size={20} />
				</button>
			</div>

			<!-- Search -->
			<div class="border-b border-white/10 p-4">
				<div class="relative">
					<Search class="text-muted-foreground absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2" />
					<Input
						placeholder="Search friends..."
						class="bg-secondary/30 border-none pl-9"
						bind:value={searchQuery}
					/>
				</div>
			</div>

			<!-- Friends List -->
			<div class="max-h-64 overflow-y-auto">
				{#if loading}
					<div class="flex items-center justify-center py-8">
						<Loader2 class="animate-spin text-white" size={24} />
					</div>
				{:else if filteredFriends.length === 0}
					<div class="text-muted-foreground py-8 text-center">
						<Users class="mx-auto mb-2 opacity-50" size={32} />
						<p>No friends found</p>
					</div>
				{:else}
					<div class="divide-y divide-white/5">
						<!-- Select All (only for invitable friends) -->
						{#if invitableFriends.length > 0}
							<button
								class="flex w-full items-center gap-3 px-4 py-3 text-sm transition-colors hover:bg-white/5"
								onclick={selectAll}
							>
								<div
									class="flex h-5 w-5 items-center justify-center rounded border {selectedIds.size ===
										invitableFriends.length && invitableFriends.length > 0
										? 'border-primary bg-primary'
										: 'border-white/30'}"
								>
									{#if selectedIds.size === invitableFriends.length && invitableFriends.length > 0}
										<Check size={12} class="text-white" />
									{/if}
								</div>
								<span class="font-medium">Select All ({invitableFriends.length} available)</span>
							</button>
						{/if}

						{#each filteredFriends as friend}
							<button
								class="flex w-full items-center gap-3 px-4 py-3 transition-colors {friend.eventStatus
									? 'cursor-not-allowed opacity-50'
									: 'hover:bg-white/5'}"
								onclick={() => toggleSelect(friend.id)}
								disabled={!!friend.eventStatus}
							>
								<div
									class="flex h-5 w-5 items-center justify-center rounded border {friend.eventStatus
										? 'border-green-500 bg-green-500/20'
										: selectedIds.has(friend.id)
											? 'border-primary bg-primary'
											: 'border-white/30'}"
								>
									{#if friend.eventStatus}
										<UserCheck size={12} class="text-green-400" />
									{:else if selectedIds.has(friend.id)}
										<Check size={12} class="text-white" />
									{/if}
								</div>
								<img
									src={friend.avatar || 'https://github.com/shadcn.png'}
									alt=""
									class="h-10 w-10 rounded-full object-cover"
								/>
								<div class="flex-1 text-left">
									<div class="font-medium">{friend.full_name || friend.username}</div>
									<div class="text-muted-foreground text-xs">@{friend.username}</div>
								</div>
								{#if friend.eventStatus}
									<span
										class="flex items-center gap-1 text-xs {friend.eventStatus === 'going'
											? 'text-green-400'
											: 'text-yellow-400'}"
									>
										{#if friend.eventStatus === 'going'}
											<Check size={12} />
										{:else}
											<Star size={12} />
										{/if}
										{getStatusLabel(friend.eventStatus)}
									</span>
								{/if}
							</button>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Message (Optional) -->
			<div class="border-t border-white/10 p-4">
				<Input
					placeholder="Add a message (optional)..."
					class="bg-secondary/30 border-none"
					bind:value={message}
				/>
			</div>

			<!-- Actions -->
			<div class="flex justify-end gap-2 border-t border-white/10 p-4">
				<Button variant="ghost" onclick={handleClose}>Cancel</Button>
				<Button
					class="gap-2"
					disabled={selectedIds.size === 0 || sending}
					onclick={sendInvitations}
				>
					{#if sending}
						<Loader2 class="h-4 w-4 animate-spin" />
					{:else}
						<Send size={16} />
					{/if}
					Send Invitations ({selectedIds.size})
				</Button>
			</div>
		</div>
	</div>
{/if}
