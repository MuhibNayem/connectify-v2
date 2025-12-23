<script lang="ts">
	import { MapPin, Users, Star, Check, Loader2 } from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button';
	import { rsvpEvent, type Event, type RSVPStatus } from '$lib/api';

	let {
		event,
		onStatusChange
	}: {
		event: Event;
		onStatusChange?: (eventId: string, status: RSVPStatus) => void;
	} = $props();

	let currentStatus = $state<RSVPStatus | undefined>(event.my_status);
	let loading = $state(false);

	// Local stats state that updates optimistically
	let goingCount = $state(event.stats.going_count);
	let interestedCount = $state(event.stats.interested_count);

	function formatDate(dateStr: string) {
		const date = new Date(dateStr);
		return {
			month: date.toLocaleString('default', { month: 'short' }).toUpperCase(),
			day: date.getDate(),
			time: date.toLocaleTimeString('default', { hour: 'numeric', minute: '2-digit' })
		};
	}

	let dateInfo = $derived(formatDate(event.start_date));

	async function handleRSVP(e: MouseEvent, status: RSVPStatus) {
		e.preventDefault();
		e.stopPropagation();
		if (loading) return;

		const previousStatus = currentStatus;
		loading = true;

		// Optimistically update UI
		if (previousStatus === 'going') goingCount--;
		if (previousStatus === 'interested') interestedCount--;
		if (status === 'going') goingCount++;
		if (status === 'interested') interestedCount++;
		currentStatus = status;

		try {
			await rsvpEvent(event.id, status);
			onStatusChange?.(event.id, status);
		} catch (err) {
			console.error('Failed to RSVP:', err);
			// Rollback on error
			currentStatus = previousStatus;
			if (previousStatus === 'going') goingCount++;
			if (previousStatus === 'interested') interestedCount++;
			if (status === 'going') goingCount--;
			if (status === 'interested') interestedCount--;
		} finally {
			loading = false;
		}
	}

	function getButton() {
		if (currentStatus === 'going')
			return {
				label: 'Going',
				icon: Check,
				class: 'bg-green-600/30 text-green-300 border-green-500/30'
			};
		if (currentStatus === 'interested')
			return {
				label: 'Interested',
				icon: Star,
				class: 'bg-yellow-600/30 text-yellow-300 border-yellow-500/30'
			};
		return { label: 'Interested', icon: Star, class: 'bg-white/10 hover:bg-white/20' };
	}

	let buttonState = $derived(getButton());
</script>

<div
	class="glass-card group relative overflow-hidden rounded-xl border border-white/10 bg-white/5 transition-all hover:bg-white/10 hover:shadow-lg"
>
	<!-- Cover Image -->
	<div class="relative h-48 w-full overflow-hidden">
		<img
			src={event.cover_image ||
				'https://images.unsplash.com/photo-1540575467063-178a50c2df87?w=400'}
			alt={event.title}
			class="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105"
		/>
		<div
			class="absolute left-4 top-4 flex flex-col items-center rounded-lg bg-white/90 px-3 py-1 shadow-sm backdrop-blur-md"
		>
			<span class="text-xs font-bold text-red-500">{dateInfo.month}</span>
			<span class="text-xl font-black text-gray-900">{dateInfo.day}</span>
		</div>
		{#if event.category}
			<div
				class="absolute right-4 top-4 rounded-full bg-black/50 px-3 py-1 text-xs text-white backdrop-blur-md"
			>
				{event.category}
			</div>
		{/if}
		{#if event.is_online}
			<div
				class="absolute bottom-4 left-4 rounded-full bg-blue-500/80 px-3 py-1 text-xs font-medium text-white backdrop-blur-md"
			>
				Online Event
			</div>
		{/if}
	</div>

	<!-- Content -->
	<div class="p-4">
		<div class="mb-1 text-xs font-semibold text-red-400">{dateInfo.time}</div>
		<h3
			class="group-hover:text-primary mb-2 truncate text-lg font-bold text-white transition-colors"
		>
			<a href="/events/{event.id}" class="after:absolute after:inset-0">{event.title}</a>
		</h3>
		<div class="mb-4 space-y-1 text-sm text-gray-400">
			<div class="flex items-center gap-2">
				<MapPin size={14} />
				<span class="truncate">{event.is_online ? 'Online' : event.location || 'Location TBA'}</span
				>
			</div>
			<div class="flex items-center gap-2">
				<Users size={14} />
				<span>
					{goingCount} going
					{#if interestedCount > 0}
						• {interestedCount} interested
					{/if}
				</span>
			</div>
			{#if event.friends_going && event.friends_going.length > 0}
				<div class="flex items-center gap-2 pt-1">
					<div class="flex -space-x-2">
						{#each event.friends_going.slice(0, 3) as friend}
							<img
								src={friend.avatar || 'https://github.com/shadcn.png'}
								alt={friend.full_name || friend.username}
								class="h-5 w-5 rounded-full ring-1 ring-black"
								title={friend.full_name || friend.username}
							/>
						{/each}
					</div>
					<span class="text-xs text-blue-400">
						{event.friends_going.length === 1
							? `${event.friends_going[0].full_name || event.friends_going[0].username} is going`
							: event.friends_going.length === 2
								? `${event.friends_going[0].full_name || event.friends_going[0].username} and 1 other friend`
								: `${event.friends_going[0].full_name || event.friends_going[0].username} and ${event.friends_going.length - 1} other friends`}
					</span>
				</div>
			{/if}
		</div>

		<div class="flex items-center justify-between">
			<!-- Host Avatar -->
			<div class="flex items-center gap-2">
				<img
					class="h-7 w-7 rounded-full ring-2 ring-black"
					src={event.creator?.avatar || 'https://github.com/shadcn.png'}
					alt=""
				/>
				<span class="text-xs text-gray-400">by {event.creator?.username || 'Unknown'}</span>
			</div>

			<div class="relative z-10 flex gap-2">
				<Button
					variant="secondary"
					size="sm"
					class={buttonState.class}
					disabled={loading}
					onclick={(e: MouseEvent) =>
						handleRSVP(e, currentStatus === 'interested' ? 'going' : 'interested')}
				>
					{#if loading}
						<Loader2 size={14} class="mr-1 animate-spin" />
					{:else}
						<svelte:component this={buttonState.icon} size={14} class="mr-1" />
					{/if}
					{buttonState.label}
				</Button>
			</div>
		</div>
	</div>
</div>
