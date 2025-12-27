<script lang="ts">
	import { onMount } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Loader2, Search, Users, Crown, Star, Check, X, UserPlus } from '@lucide/svelte';
	import { getEventAttendees, type EventAttendee, type RSVPStatus } from '$lib/api';
	import { goto } from '$app/navigation';

	let {
		eventId,
		isHost = false,
		compact = false,
		maxDisplay = 0
	}: {
		eventId: string;
		isHost?: boolean;
		compact?: boolean;
		maxDisplay?: number;
	} = $props();

	let attendees: EventAttendee[] = $state([]);
	let loading = $state(true);
	let total = $state(0);
	let activeTab: RSVPStatus | '' = $state('going');
	let searchQuery = $state('');
	let page = $state(1);
	let hasMore = $state(false);

	onMount(async () => {
		await loadAttendees();
	});

	async function loadAttendees() {
		loading = true;
		try {
			const response = await getEventAttendees(
				eventId,
				activeTab || undefined,
				1,
				maxDisplay || 20
			);
			attendees = response.attendees || [];
			total = response.total || 0;
			hasMore = attendees.length < total;
		} catch (err) {
			console.error('Failed to load attendees:', err);
		} finally {
			loading = false;
		}
	}

	async function loadMore() {
		page++;
		try {
			const response = await getEventAttendees(eventId, activeTab || undefined, page, 20);
			attendees = [...attendees, ...(response.attendees || [])];
			hasMore = attendees.length < total;
		} catch (err) {
			page--;
		}
	}

	async function changeTab(tab: RSVPStatus | '') {
		activeTab = tab;
		page = 1;
		await loadAttendees();
	}

	let filteredAttendees = $derived(
		attendees.filter(
			(a) =>
				!searchQuery ||
				a.user.full_name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
				a.user.username?.toLowerCase().includes(searchQuery.toLowerCase())
		)
	);

	function getStatusIcon(status: RSVPStatus) {
		switch (status) {
			case 'going':
				return Check;
			case 'interested':
				return Star;
			default:
				return null;
		}
	}

	function getStatusColor(status: RSVPStatus) {
		switch (status) {
			case 'going':
				return 'text-green-400';
			case 'interested':
				return 'text-yellow-400';
			default:
				return 'text-muted-foreground';
		}
	}

	const tabs: { value: RSVPStatus | ''; label: string }[] = [
		{ value: 'going', label: 'Going' },
		{ value: 'interested', label: 'Interested' },
		{ value: 'invited', label: 'Invited' },
		{ value: '', label: 'All' }
	];
</script>

<div class="space-y-4">
	{#if !compact}
		<!-- Tabs -->
		<div class="flex gap-1 rounded-xl bg-white/5 p-1">
			{#each tabs as tab}
				<button
					class="flex-1 rounded-lg px-3 py-2 text-sm font-medium transition-colors
                        {activeTab === tab.value
						? 'bg-primary text-white'
						: 'text-muted-foreground hover:text-foreground'}"
					onclick={() => changeTab(tab.value)}
				>
					{tab.label}
				</button>
			{/each}
		</div>

		<!-- Search -->
		<div class="relative">
			<Search class="text-muted-foreground absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2" />
			<Input
				placeholder="Search guests..."
				class="border-white/10 bg-white/5 pl-9"
				bind:value={searchQuery}
			/>
		</div>
	{/if}

	<!-- Attendees List -->
	{#if loading}
		<div class="flex items-center justify-center py-8">
			<Loader2 class="animate-spin text-white" size={24} />
		</div>
	{:else if filteredAttendees.length === 0}
		<div class="text-muted-foreground py-8 text-center">
			<Users class="mx-auto mb-2 opacity-50" size={32} />
			<p>No guests yet</p>
		</div>
	{:else}
		<div class="space-y-2">
			{#each filteredAttendees as attendee}
				<a
					href="/profile/{attendee.user.id}"
					class="flex items-center gap-3 rounded-xl p-3 transition-colors hover:bg-white/5"
				>
					<div class="relative">
						<img
							src={attendee.user.avatar || 'https://github.com/shadcn.png'}
							alt=""
							class="h-12 w-12 rounded-full object-cover"
						/>
						{#if attendee.is_host}
							<div class="absolute -right-1 -top-1 rounded-full bg-yellow-500 p-1">
								<Crown size={10} class="text-white" />
							</div>
						{:else if attendee.is_co_host}
							<div class="absolute -right-1 -top-1 rounded-full bg-blue-500 p-1">
								<Crown size={10} class="text-white" />
							</div>
						{/if}
					</div>
					<div class="flex-1">
						<div class="flex items-center gap-2">
							<span class="font-medium">{attendee.user.full_name || attendee.user.username}</span>
							{#if attendee.is_host}
								<span class="rounded bg-yellow-500/20 px-1.5 py-0.5 text-xs text-yellow-400">
									Host
								</span>
							{:else if attendee.is_co_host}
								<span class="rounded bg-blue-500/20 px-1.5 py-0.5 text-xs text-blue-400">
									Co-host
								</span>
							{/if}
						</div>
						<div class="text-muted-foreground text-xs">@{attendee.user.username}</div>
					</div>
					{#if getStatusIcon(attendee.status)}
						<svelte:component
							this={getStatusIcon(attendee.status)}
							size={18}
							class={getStatusColor(attendee.status)}
						/>
					{/if}
				</a>
			{/each}
		</div>

		{#if hasMore && !compact}
			<div class="text-center">
				<Button variant="ghost" onclick={loadMore}>Load More</Button>
			</div>
		{/if}

		{#if compact && total > maxDisplay}
			<a
				href="/events/{eventId}/guests"
				class="text-primary block text-center text-sm font-medium hover:underline"
			>
				See all {total} guests
			</a>
		{/if}
	{/if}
</div>
