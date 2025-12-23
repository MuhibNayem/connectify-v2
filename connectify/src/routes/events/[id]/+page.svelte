<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import EventHero from '$lib/components/events/EventHero.svelte';
	import EventsSidebar from '$lib/components/events/EventsSidebar.svelte';
	import EventGuestList from '$lib/components/events/EventGuestList.svelte';
	import EventDiscussion from '$lib/components/events/EventDiscussion.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Calendar, MapPin, Loader2, Globe, Lock, Users, ExternalLink } from '@lucide/svelte';
	import { getEvent, type Event, type RSVPStatus } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';

	let event: Event | null = $state(null);
	let loading = $state(true);
	let error = $state('');
	let activeTab = $state<'discussion' | 'details'>('discussion');

	let currentUser = $derived(auth.state.user);

	onMount(async () => {
		try {
			const id = $page.params.id;
			if (id) {
				event = await getEvent(id);
			}
		} catch (err) {
			console.error('Failed to load event:', err);
			error = 'Failed to load event details.';
		} finally {
			loading = false;
		}
	});

	function handleStatusChange(
		status: RSVPStatus,
		goingDelta: number = 0,
		interestedDelta: number = 0
	) {
		if (event) {
			event = {
				...event,
				my_status: status,
				stats: {
					...event.stats,
					going_count: event.stats.going_count + goingDelta,
					interested_count: event.stats.interested_count + interestedDelta
				}
			};
		}
	}

	function addToCalendar() {
		if (!event) return;
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

	function openMap() {
		if (!event?.location) return;
		const url = `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(event.location)}`;
		window.open(url, '_blank');
	}

	// Check if current user can post (is attendee or host)
	let canPost = $derived(
		event?.is_host || event?.my_status === 'going' || event?.my_status === 'interested'
	);
</script>

<div class="bg-background text-foreground flex h-[calc(100vh-4rem)] w-full overflow-hidden">
	<EventsSidebar />

	<div class="flex-1 overflow-y-auto">
		<div class="pb-20">
			{#if loading}
				<div class="flex h-[50vh] items-center justify-center">
					<Loader2 class="animate-spin text-white" size={48} />
				</div>
			{:else if error || !event}
				<div class="flex h-[50vh] flex-col items-center justify-center gap-4 text-center">
					<h2 class="text-2xl font-bold">Event not found</h2>
					<p class="text-muted-foreground">
						{error || "The event you're looking for doesn't exist."}
					</p>
					<Button href="/events" variant="outline">Back to Events</Button>
				</div>
			{:else}
				<EventHero {event} onStatusChange={handleStatusChange} />

				<div class="mx-auto mt-8 grid max-w-5xl gap-8 px-4 md:grid-cols-3">
					<!-- Left Column: Details & Discussion -->
					<div class="space-y-6 md:col-span-2">
						<!-- About Section -->
						<div class="glass-card bg-card rounded-xl border border-white/5 p-6 shadow-sm">
							<h2 class="mb-4 text-xl font-bold">About Event</h2>
							<p class="text-muted-foreground whitespace-pre-line leading-relaxed">
								{event.description || 'No description provided.'}
							</p>

							<!-- Tags/Category -->
							<div class="mt-4 flex flex-wrap gap-2">
								{#if event.category}
									<span
										class="bg-primary/20 text-primary rounded-full px-3 py-1 text-xs font-medium"
									>
										{event.category}
									</span>
								{/if}
								<span class="flex items-center gap-1 rounded-full bg-white/10 px-3 py-1 text-xs">
									{#if event.privacy === 'public'}
										<Globe size={12} /> Public
									{:else}
										<Lock size={12} /> {event.privacy === 'private' ? 'Private' : 'Friends Only'}
									{/if}
								</span>
								{#if event.is_online}
									<span class="rounded-full bg-blue-500/20 px-3 py-1 text-xs text-blue-400">
										Online Event
									</span>
								{/if}
							</div>
						</div>

						<!-- Tabs -->
						<div class="flex gap-1 rounded-xl bg-white/5 p-1">
							<button
								class="flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-colors
									{activeTab === 'discussion'
									? 'bg-primary text-white'
									: 'text-muted-foreground hover:text-foreground'}"
								onclick={() => (activeTab = 'discussion')}
							>
								Discussion
							</button>
							<button
								class="flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-colors
									{activeTab === 'details' ? 'bg-primary text-white' : 'text-muted-foreground hover:text-foreground'}"
								onclick={() => (activeTab = 'details')}
							>
								Details
							</button>
						</div>

						<!-- Tab Content -->
						{#if activeTab === 'discussion'}
							<div class="glass-card bg-card rounded-xl border border-white/5 p-6 shadow-sm">
								<h2 class="mb-4 text-xl font-bold">Discussion</h2>
								<EventDiscussion eventId={event.id} {canPost} currentUserId={currentUser?.id} />
							</div>
						{:else}
							<!-- More Details -->
							<div
								class="glass-card bg-card space-y-6 rounded-xl border border-white/5 p-6 shadow-sm"
							>
								<div>
									<h3 class="mb-2 text-lg font-bold">Host</h3>
									<a
										href="/profile/{event.creator.username}"
										class="flex items-center gap-3 rounded-xl p-3 transition-colors hover:bg-white/5"
									>
										<img
											src={event.creator.avatar || 'https://github.com/shadcn.png'}
											alt=""
											class="h-12 w-12 rounded-full"
										/>
										<div>
											<div class="font-semibold">
												{event.creator.full_name || event.creator.username}
											</div>
											<div class="text-muted-foreground text-sm">@{event.creator.username}</div>
										</div>
									</a>
								</div>

								<div>
									<h3 class="mb-2 text-lg font-bold">Statistics</h3>
									<div class="grid grid-cols-3 gap-4 text-center">
										<div class="rounded-xl bg-white/5 p-4">
											<div class="text-2xl font-bold text-green-400">{event.stats.going_count}</div>
											<div class="text-muted-foreground text-xs">Going</div>
										</div>
										<div class="rounded-xl bg-white/5 p-4">
											<div class="text-2xl font-bold text-yellow-400">
												{event.stats.interested_count}
											</div>
											<div class="text-muted-foreground text-xs">Interested</div>
										</div>
										<div class="rounded-xl bg-white/5 p-4">
											<div class="text-2xl font-bold text-blue-400">{event.stats.share_count}</div>
											<div class="text-muted-foreground text-xs">Shares</div>
										</div>
									</div>
								</div>
							</div>
						{/if}
					</div>

					<!-- Right Column: Sidebar info -->
					<div class="space-y-6">
						<div
							class="glass-card bg-card space-y-4 rounded-xl border border-white/5 p-6 shadow-sm"
						>
							<h3 class="text-lg font-bold">Event Details</h3>

							<div class="flex gap-4">
								<Calendar class="text-muted-foreground shrink-0" />
								<div>
									<div class="font-semibold">
										{new Date(event.start_date).toLocaleDateString(undefined, {
											weekday: 'long',
											month: 'long',
											day: 'numeric',
											year: 'numeric'
										})}
									</div>
									<div class="text-muted-foreground text-sm">
										{new Date(event.start_date).toLocaleTimeString(undefined, {
											hour: 'numeric',
											minute: '2-digit'
										})}
										{#if event.end_date}
											- {new Date(event.end_date).toLocaleTimeString(undefined, {
												hour: 'numeric',
												minute: '2-digit'
											})}
										{/if}
									</div>
									<button
										class="mt-1 flex items-center gap-1 text-xs text-blue-400 hover:underline"
										onclick={addToCalendar}
									>
										<ExternalLink size={12} />
										Add to Calendar
									</button>
								</div>
							</div>

							<div class="flex gap-4">
								<MapPin class="text-muted-foreground shrink-0" />
								<div>
									<div class="font-semibold">
										{event.is_online ? 'Online Event' : event.location || 'Location TBA'}
									</div>
									<div class="text-muted-foreground text-sm">
										{event.is_online ? 'Join from anywhere' : 'In Person'}
									</div>
									{#if !event.is_online && event.location}
										<button
											class="mt-1 flex items-center gap-1 text-xs text-blue-400 hover:underline"
											onclick={openMap}
										>
											<ExternalLink size={12} />
											Show on Map
										</button>
									{/if}
								</div>
							</div>
						</div>

						<!-- Guest List -->
						<div class="glass-card bg-card rounded-xl border border-white/5 p-6 shadow-sm">
							<div class="mb-4 flex items-center justify-between">
								<h3 class="text-lg font-bold">Guest List</h3>
								<a
									href="/events/{event.id}/guests"
									class="text-primary text-sm font-medium hover:underline"
								>
									See All
								</a>
							</div>
							<div class="text-muted-foreground mb-4 flex justify-between text-sm">
								<span class="flex items-center gap-1">
									<Users size={14} class="text-green-400" />
									{event.stats.going_count} Going
								</span>
								<span>{event.stats.interested_count} Interested</span>
							</div>

							<EventGuestList
								eventId={event.id}
								compact={true}
								maxDisplay={8}
								isHost={event.is_host}
							/>
						</div>
					</div>
				</div>
			{/if}
		</div>
	</div>
</div>
