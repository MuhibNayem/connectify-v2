<script lang="ts">
	import {
		MapPin,
		Share2,
		MoreHorizontal,
		Check,
		Star,
		X,
		Edit2,
		Trash2,
		Users2,
		CalendarPlus,
		Loader2
	} from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button';
	import { rsvpEvent, deleteEvent, type Event, type RSVPStatus } from '$lib/api';
	import { goto } from '$app/navigation';
	import EventShareModal from './EventShareModal.svelte';
	import EventInviteModal from './EventInviteModal.svelte';
	import { websocketMessages } from '$lib/websocket';

	let {
		event,
		onStatusChange,
		onDelete
	}: {
		event: Event;
		onStatusChange?: (status: RSVPStatus, goingDelta: number, interestedDelta: number) => void;
		onDelete?: () => void;
	} = $props();

	let currentStatus = $state<RSVPStatus | undefined>(event.my_status);
	let loading = $state(false);
	let showShareModal = $state(false);
	let showInviteModal = $state(false);
	let showMoreMenu = $state(false);
	let deleting = $state(false);

	// Local stats state that updates optimistically
	let goingCount = $state(event.stats.going_count);
	let interestedCount = $state(event.stats.interested_count);

	$effect(() => {
		const msg = $websocketMessages;
		if (msg?.type === 'EVENT_RSVP_UPDATE' && msg.data.event_id === event.id) {
			if (msg.data.stats) {
				goingCount = msg.data.stats.going_count;
				interestedCount = msg.data.stats.interested_count;
			}
		}
	});

	function formatFullDate(start: string, end?: string) {
		const s = new Date(start);
		const e = end ? new Date(end) : undefined;

		const dateStr = s.toLocaleDateString('en-US', {
			weekday: 'long',
			month: 'long',
			day: 'numeric',
			year: 'numeric'
		});

		let timeStr = s.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
		if (e) {
			timeStr += ` - ${e.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' })}`;
		}

		return { dateStr, timeStr };
	}

	let dateInfo = $derived(formatFullDate(event.start_date, event.end_date));

	async function handleRSVP(status: RSVPStatus) {
		if (loading) return;

		const previousStatus = currentStatus;
		loading = true;

		// Optimistically update UI
		let goingDelta = 0;
		let interestedDelta = 0;

		if (previousStatus === 'going') {
			goingCount--;
			goingDelta--;
		}
		if (previousStatus === 'interested') {
			interestedCount--;
			interestedDelta--;
		}
		if (status === 'going') {
			goingCount++;
			goingDelta++;
		}
		if (status === 'interested') {
			interestedCount++;
			interestedDelta++;
		}
		currentStatus = status;

		try {
			await rsvpEvent(event.id, status);
			onStatusChange?.(status, goingDelta, interestedDelta);
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

	async function handleDelete() {
		if (!confirm('Are you sure you want to delete this event? This action cannot be undone.'))
			return;

		deleting = true;
		try {
			await deleteEvent(event.id);
			onDelete?.();
			goto('/events');
		} catch (err) {
			console.error('Failed to delete event:', err);
			alert('Failed to delete event');
		} finally {
			deleting = false;
		}
	}

	function getButtonClass(status: RSVPStatus) {
		if (currentStatus === status) {
			if (status === 'going') return 'bg-green-600 hover:bg-green-700 text-white';
			if (status === 'interested') return 'bg-yellow-600 hover:bg-yellow-700 text-white';
		}
		return 'bg-white/10 hover:bg-white/20';
	}

	function addToCalendar() {
		const start = new Date(event.start_date);
		const end = event.end_date
			? new Date(event.end_date)
			: new Date(start.getTime() + 2 * 60 * 60 * 1000);

		const formatForCalendar = (d: Date) =>
			d
				.toISOString()
				.replace(/-|:|\.\d+/g, '')
				.slice(0, -1);

		const url = `https://www.google.com/calendar/render?action=TEMPLATE&text=${encodeURIComponent(event.title)}&dates=${formatForCalendar(start)}/${formatForCalendar(end)}&location=${encodeURIComponent(event.location || '')}&details=${encodeURIComponent(event.description || '')}`;

		window.open(url, '_blank');
	}
</script>

<div class="relative w-full overflow-hidden rounded-b-3xl bg-black shadow-2xl">
	<!-- Background Blur -->
	<div
		class="absolute inset-0 bg-cover bg-center opacity-50 blur-3xl"
		style="background-image: url('{event.cover_image ||
			'https://images.unsplash.com/photo-1540575467063-178a50c2df87?w=800'}')"
	></div>

	<!-- Main Cover -->
	<div class="relative mx-auto max-w-5xl">
		<div class="aspect-[21/9] w-full overflow-hidden md:rounded-b-2xl">
			<img
				src={event.cover_image ||
					'https://images.unsplash.com/photo-1540575467063-178a50c2df87?w=800'}
				alt={event.title}
				class="h-full w-full object-cover"
			/>
			{#if event.is_online}
				<div
					class="absolute left-4 top-4 rounded-full bg-blue-500 px-4 py-1.5 text-sm font-semibold text-white backdrop-blur-md"
				>
					🌐 Online Event
				</div>
			{/if}
		</div>

		<!-- Glass Info Overlay (Floating overlap) -->
		<div class="relative -mt-20 px-4 pb-8 md:px-8">
			<div
				class="glass-card flex flex-col gap-6 rounded-2xl border border-white/10 bg-black/60 p-6 backdrop-blur-xl md:flex-row md:items-end md:justify-between"
			>
				<!-- Date Badge & Title -->
				<div class="flex flex-col gap-4 md:flex-row md:items-start">
					<div
						class="hidden flex-col items-center rounded-xl bg-white/10 px-4 py-3 text-white backdrop-blur-md md:flex"
					>
						<span class="text-sm font-bold uppercase text-red-400"
							>{new Date(event.start_date).toLocaleString('default', { month: 'short' })}</span
						>
						<span class="text-3xl font-black">{new Date(event.start_date).getDate()}</span>
					</div>

					<div class="space-y-1">
						<div class="font-bold uppercase tracking-wide text-red-400 md:hidden">
							{dateInfo.dateStr}
						</div>
						<h1 class="text-3xl font-bold text-white md:text-5xl">{event.title}</h1>
						<div class="flex items-center gap-2 text-gray-300">
							<span class="font-medium">{dateInfo.dateStr}</span>
							<span class="h-1 w-1 rounded-full bg-gray-500"></span>
							<span>{dateInfo.timeStr}</span>
						</div>
						<div class="flex items-center gap-2 text-gray-300">
							<MapPin size={16} />
							<span>{event.is_online ? 'Online Event' : event.location || 'Location TBA'}</span>
						</div>
						<div class="flex items-center gap-2 pt-2 text-sm text-gray-400">
							<span>Hosted by</span>
							<a
								href="/profile/{event.creator.id}"
								class="flex items-center gap-1 text-white hover:underline"
							>
								<img
									src={event.creator.avatar || 'https://github.com/shadcn.png'}
									alt=""
									class="h-5 w-5 rounded-full"
								/>
								<span class="font-semibold"
									>{event.creator.full_name || event.creator.username}</span
								>
							</a>
						</div>
						<!-- Stats -->
						<div class="flex items-center gap-4 pt-2 text-sm text-gray-400">
							<span class="flex items-center gap-1">
								<Check size={14} class="text-green-400" />
								{goingCount} going
							</span>
							<span class="flex items-center gap-1">
								<Star size={14} class="text-yellow-400" />
								{interestedCount} interested
							</span>
						</div>
						<!-- Friends Going -->
						{#if event.friends_going && event.friends_going.length > 0}
							<div class="flex items-center gap-2 pt-2">
								<div class="flex -space-x-2">
									{#each event.friends_going.slice(0, 4) as friend}
										<img
											src={friend.avatar || 'https://github.com/shadcn.png'}
											alt={friend.full_name || friend.username}
											class="h-6 w-6 rounded-full ring-2 ring-black"
											title={friend.full_name || friend.username}
										/>
									{/each}
								</div>
								<span class="text-sm text-blue-400">
									{event.friends_going.length === 1
										? `${event.friends_going[0].full_name || event.friends_going[0].username} is going`
										: event.friends_going.length <= 3
											? event.friends_going.map((f) => f.full_name || f.username).join(', ') +
												' are going'
											: `${event.friends_going[0].full_name || event.friends_going[0].username} and ${event.friends_going.length - 1} other friends are going`}
								</span>
							</div>
						{/if}
					</div>
				</div>

				<!-- Actions -->
				<div class="flex w-full flex-col gap-3 md:w-auto md:flex-row md:items-center">
					<!-- RSVP Buttons -->
					<div class="flex gap-2">
						<Button
							class="flex-1 gap-2 {getButtonClass('going')}"
							disabled={loading}
							onclick={() => handleRSVP('going')}
						>
							{#if loading && currentStatus !== 'going'}
								<Loader2 class="h-4 w-4 animate-spin" />
							{:else}
								<Check size={16} />
							{/if}
							Going
						</Button>
						<Button
							variant="secondary"
							class="flex-1 gap-2 {getButtonClass('interested')}"
							disabled={loading}
							onclick={() => handleRSVP('interested')}
						>
							<Star size={16} />
							Interested
						</Button>
					</div>

					<!-- Action Icons -->
					<div class="flex gap-2">
						<Button
							variant="ghost"
							size="icon"
							class="bg-white/5 hover:bg-white/10"
							title="Invite Friends"
							onclick={() => (showInviteModal = true)}
						>
							<Users2 size={20} />
						</Button>
						<Button
							variant="ghost"
							size="icon"
							class="bg-white/5 hover:bg-white/10"
							title="Add to Calendar"
							onclick={addToCalendar}
						>
							<CalendarPlus size={20} />
						</Button>
						<Button
							variant="ghost"
							size="icon"
							class="bg-white/5 hover:bg-white/10"
							title="Share"
							onclick={() => (showShareModal = true)}
						>
							<Share2 size={20} />
						</Button>

						<!-- More Menu (for hosts) -->
						<div class="relative">
							<Button
								variant="ghost"
								size="icon"
								class="bg-white/5 hover:bg-white/10"
								onclick={() => (showMoreMenu = !showMoreMenu)}
							>
								<MoreHorizontal size={20} />
							</Button>

							{#if showMoreMenu}
								<div
									class="bg-background/95 absolute right-0 top-full z-50 mt-1 w-40 overflow-hidden rounded-lg border border-white/10 shadow-xl backdrop-blur-lg"
								>
									{#if event.is_host}
										<a
											href="/events/{event.id}/edit"
											class="flex w-full items-center gap-2 px-4 py-2.5 text-sm transition-colors hover:bg-white/10"
										>
											<Edit2 size={16} />
											Edit Event
										</a>
										<button
											class="flex w-full items-center gap-2 px-4 py-2.5 text-sm text-red-400 transition-colors hover:bg-white/10"
											onclick={handleDelete}
											disabled={deleting}
										>
											{#if deleting}
												<Loader2 class="h-4 w-4 animate-spin" />
											{:else}
												<Trash2 size={16} />
											{/if}
											Delete Event
										</button>
									{:else}
										<button
											class="flex w-full items-center gap-2 px-4 py-2.5 text-sm transition-colors hover:bg-white/10"
											onclick={() => handleRSVP('not_going')}
										>
											<X size={16} />
											Not Going
										</button>
									{/if}
								</div>
								<!-- svelte-ignore a11y_click_events_have_key_events -->
								<!-- svelte-ignore a11y_no_static_element_interactions -->
								<div class="fixed inset-0 z-40" onclick={() => (showMoreMenu = false)}></div>
							{/if}
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>

<!-- Modals -->
<EventShareModal bind:open={showShareModal} eventId={event.id} eventTitle={event.title} />
<EventInviteModal bind:open={showInviteModal} eventId={event.id} eventTitle={event.title} />
