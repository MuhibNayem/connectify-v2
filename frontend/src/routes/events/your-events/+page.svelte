<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import EventsSidebar from '$lib/components/events/EventsSidebar.svelte';
	import EventCard from '$lib/components/events/EventCard.svelte';
	import { Calendar, Plus, Loader2, Mail, Check, X, UserPlus } from '@lucide/svelte';
	import {
		getMyEvents,
		getEventInvitations,
		respondToEventInvitation,
		type Event,
		type EventInvitation,
		type RSVPStatus
	} from '$lib/api';

	let activeTab = $state<'going' | 'hosting' | 'invited' | 'past'>('going');
	let allEvents: Event[] = $state([]);
	let invitations: EventInvitation[] = $state([]);
	let loading = $state(true);
	let respondingId = $state<string | null>(null);

	onMount(async () => {
		await Promise.all([loadEvents(), loadInvitations()]);
	});

	async function loadEvents() {
		loading = true;
		try {
			allEvents = await getMyEvents(1, 100);
		} catch (err) {
			console.error('Failed to load my events:', err);
		} finally {
			loading = false;
		}
	}

	async function loadInvitations() {
		try {
			const res = await getEventInvitations(1, 50);
			invitations = res.invitations || [];
		} catch (err) {
			console.error('Failed to load invitations:', err);
		}
	}

	async function handleInvitationResponse(invitationId: string, accept: boolean) {
		respondingId = invitationId;
		try {
			const invitation = invitations.find((i) => i.id === invitationId);
			await respondToEventInvitation(invitationId, accept);
			invitations = invitations.filter((i) => i.id !== invitationId);

			if (accept && invitation) {
				// Refresh events and navigate to the event
				await loadEvents();
				goto(`/events/${invitation.event.id}`);
			}
		} catch (err) {
			console.error('Failed to respond:', err);
			alert('Failed to respond to invitation');
		} finally {
			respondingId = null;
		}
	}

	function handleStatusChange(eventId: string, status: RSVPStatus) {
		allEvents = allEvents.map((e) => {
			if (e.id === eventId) {
				return { ...e, my_status: status };
			}
			return e;
		});
	}

	let goingEvents = $derived(
		allEvents.filter(
			(e) => !e.is_host && e.my_status === 'going' && new Date(e.start_date) > new Date()
		)
	);
	let hostedEvents = $derived(
		allEvents.filter((e) => e.is_host && new Date(e.start_date) > new Date())
	);
	let pastEvents = $derived(allEvents.filter((e) => new Date(e.start_date) < new Date()));
	let interestedEvents = $derived(
		allEvents.filter(
			(e) => !e.is_host && e.my_status === 'interested' && new Date(e.start_date) > new Date()
		)
	);

	// Helper function to get tab count
	function getTabCount(tabId: string): number {
		switch (tabId) {
			case 'going':
				return goingEvents.length;
			case 'hosting':
				return hostedEvents.length;
			case 'invited':
				return invitations.length;
			case 'past':
				return pastEvents.length;
			default:
				return 0;
		}
	}

	const tabs = [
		{ id: 'going', label: 'Going' },
		{ id: 'hosting', label: 'Hosting' },
		{ id: 'invited', label: 'Invited' },
		{ id: 'past', label: 'Past' }
	] as const;
</script>

<div class="bg-background text-foreground flex h-[calc(100vh-4rem)] w-full overflow-hidden">
	<EventsSidebar />

	<div class="flex-1 overflow-y-auto p-4 md:p-8">
		<div class="mx-auto max-w-5xl pb-20">
			<div class="mb-8 flex items-center justify-between">
				<h1 class="text-3xl font-bold">Your Events</h1>
				<Button href="/events/create" class="gap-2">
					<Plus size={18} /> Create Event
				</Button>
			</div>

			<!-- Tabs -->
			<div class="mb-8 flex gap-1 rounded-xl bg-white/5 p-1">
				{#each tabs as tab}
					<button
						class="relative flex-1 rounded-lg px-4 py-2 text-sm font-semibold transition-colors
							{activeTab === tab.id ? 'bg-primary text-white' : 'text-muted-foreground hover:text-foreground'}"
						onclick={() => (activeTab = tab.id)}
					>
						{tab.label}
						{#if getTabCount(tab.id) > 0}
							<span class="bg-primary/30 text-primary ml-1 rounded-full px-1.5 text-xs">
								{getTabCount(tab.id)}
							</span>
						{/if}
					</button>
				{/each}
			</div>

			<!-- Content -->
			<div>
				{#if loading}
					<div class="flex justify-center py-20">
						<Loader2 class="animate-spin text-white" size={48} />
					</div>
				{:else}
					<!-- Going Tab -->
					{#if activeTab === 'going'}
						{#if goingEvents.length > 0}
							<div class="mb-8">
								<h2 class="mb-4 text-lg font-semibold">Events You're Attending</h2>
								<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
									{#each goingEvents as event}
										<EventCard {event} onStatusChange={handleStatusChange} />
									{/each}
								</div>
							</div>
						{/if}

						{#if interestedEvents.length > 0}
							<div>
								<h2 class="mb-4 text-lg font-semibold text-yellow-400">Interested</h2>
								<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
									{#each interestedEvents as event}
										<EventCard {event} onStatusChange={handleStatusChange} />
									{/each}
								</div>
							</div>
						{/if}

						{#if goingEvents.length === 0 && interestedEvents.length === 0}
							<div class="text-muted-foreground flex flex-col items-center justify-center py-20">
								<Calendar size={48} class="mb-4 opacity-50" />
								<p>You haven't RSVP'd to any events yet.</p>
								<Button variant="link" href="/events">Browse Events</Button>
							</div>
						{/if}
					{/if}

					<!-- Hosting Tab -->
					{#if activeTab === 'hosting'}
						{#if hostedEvents.length > 0}
							<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
								{#each hostedEvents as event}
									<EventCard {event} onStatusChange={handleStatusChange} />
								{/each}
							</div>
						{:else}
							<div class="text-muted-foreground flex flex-col items-center justify-center py-20">
								<Calendar size={48} class="mb-4 opacity-50" />
								<p>You are not hosting any events.</p>
								<Button variant="link" href="/events/create">Create One</Button>
							</div>
						{/if}
					{/if}

					<!-- Invited Tab -->
					{#if activeTab === 'invited'}
						{#if invitations.length > 0}
							<div class="space-y-4">
								{#each invitations as invitation}
									<div
										class="glass-card bg-card flex flex-col gap-4 rounded-xl border border-white/10 p-4 sm:flex-row sm:items-center"
									>
										<!-- Event Info -->
										<a href="/events/{invitation.event.id}" class="flex flex-1 gap-4">
											<div class="h-20 w-28 flex-shrink-0 overflow-hidden rounded-lg">
												<img
													src={invitation.event.cover_image ||
														'https://images.unsplash.com/photo-1540575467063-178a50c2df87?w=200'}
													alt=""
													class="h-full w-full object-cover"
												/>
											</div>
											<div>
												<div class="text-muted-foreground flex items-center gap-2 text-sm">
													<img
														src={invitation.inviter.avatar || 'https://github.com/shadcn.png'}
														alt=""
														class="h-5 w-5 rounded-full"
													/>
													<span
														>{invitation.inviter.full_name || invitation.inviter.username} invited you</span
													>
												</div>
												<h3 class="mt-1 font-bold">{invitation.event.title}</h3>
												<div class="text-muted-foreground mt-1 flex items-center gap-3 text-sm">
													<span
														>{new Date(invitation.event.start_date).toLocaleDateString(undefined, {
															month: 'short',
															day: 'numeric'
														})}</span
													>
													<span>{invitation.event.location || 'Online'}</span>
												</div>
											</div>
										</a>

										<!-- Actions -->
										<div class="flex gap-2 sm:flex-col">
											<Button
												size="sm"
												class="flex-1 gap-2"
												disabled={respondingId === invitation.id}
												onclick={() => handleInvitationResponse(invitation.id, true)}
											>
												{#if respondingId === invitation.id}
													<Loader2 class="h-4 w-4 animate-spin" />
												{:else}
													<Check size={14} />
												{/if}
												Accept
											</Button>
											<Button
												variant="ghost"
												size="sm"
												class="flex-1 gap-2"
												disabled={respondingId === invitation.id}
												onclick={() => handleInvitationResponse(invitation.id, false)}
											>
												<X size={14} />
												Decline
											</Button>
										</div>
									</div>
								{/each}
							</div>
						{:else}
							<div class="text-muted-foreground flex flex-col items-center justify-center py-20">
								<Mail size={48} class="mb-4 opacity-50" />
								<p>No pending invitations.</p>
								<p class="text-sm">When friends invite you to events, they'll appear here.</p>
							</div>
						{/if}
					{/if}

					<!-- Past Tab -->
					{#if activeTab === 'past'}
						{#if pastEvents.length > 0}
							<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
								{#each pastEvents as event}
									<div class="opacity-75">
										<EventCard {event} onStatusChange={handleStatusChange} />
									</div>
								{/each}
							</div>
						{:else}
							<div class="text-muted-foreground flex flex-col items-center justify-center py-20">
								<Calendar size={48} class="mb-4 opacity-50" />
								<p>No past events found.</p>
							</div>
						{/if}
					{/if}
				{/if}
			</div>
		</div>
	</div>
</div>
