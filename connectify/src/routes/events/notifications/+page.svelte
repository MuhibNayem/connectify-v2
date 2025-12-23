<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import EventsSidebar from '$lib/components/events/EventsSidebar.svelte';
	import {
		Bell,
		Calendar,
		Users,
		MessageCircle,
		Check,
		X,
		Loader2,
		UserPlus
	} from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button';
	import { getEventInvitations, respondToEventInvitation, type EventInvitation } from '$lib/api';
	import { formatDistanceToNow } from 'date-fns';
	import { websocketMessages } from '$lib/websocket';
	import { fade, fly } from 'svelte/transition';

	let invitations: EventInvitation[] = $state([]);
	let loading = $state(true);
	let responding = $state<string | null>(null);
	let removingIds = $state<Set<string>>(new Set());

	// WebSocket subscription for real-time updates
	let wsUnsubscribe: (() => void) | null = null;

	onMount(async () => {
		await loadInvitations();

		// Subscribe to WebSocket for real-time invitation updates
		wsUnsubscribe = websocketMessages.subscribe((event) => {
			if (!event) return;

			switch (event.type) {
				case 'EVENT_INVITATION_RESPONDED':
					// Remove the invitation from list when responded from elsewhere
					const invitationId = event.data?.invitation_id;
					if (invitationId) {
						removeInvitationWithAnimation(invitationId);
					}
					break;
			}
		});
	});

	onDestroy(() => {
		if (wsUnsubscribe) {
			wsUnsubscribe();
		}
	});

	async function loadInvitations() {
		loading = true;
		try {
			const res = await getEventInvitations(1, 50);
			invitations = res.invitations || [];
		} catch (err) {
			console.error('Failed to load invitations:', err);
		} finally {
			loading = false;
		}
	}

	function removeInvitationWithAnimation(invitationId: string) {
		// Mark for removal animation
		removingIds = new Set([...removingIds, invitationId]);

		// Remove after animation delay
		setTimeout(() => {
			invitations = invitations.filter((i) => i.id !== invitationId);
			removingIds = new Set([...removingIds].filter((id) => id !== invitationId));
		}, 300);
	}

	async function handleRespond(invitationId: string, accept: boolean) {
		// Find the invitation BEFORE removing it
		const invitation = invitations.find((i) => i.id === invitationId);
		const eventId = invitation?.event.id;

		responding = invitationId;
		try {
			await respondToEventInvitation(invitationId, accept);

			// Remove from list with animation after responding
			removeInvitationWithAnimation(invitationId);

			// If accepted, navigate to the event
			if (accept && eventId) {
				goto(`/events/${eventId}`);
			}
		} catch (err) {
			console.error('Failed to respond:', err);
			alert('Failed to respond to invitation');
		} finally {
			responding = null;
		}
	}

	function navigateToEvent(eventId: string) {
		goto(`/events/${eventId}`);
	}

	function formatTime(dateStr: string) {
		try {
			return formatDistanceToNow(new Date(dateStr), { addSuffix: true });
		} catch {
			return '';
		}
	}
</script>

<div class="bg-background text-foreground flex h-[calc(100vh-4rem)] w-full overflow-hidden">
	<EventsSidebar />

	<div class="flex-1 overflow-y-auto p-4 md:p-8">
		<div class="mx-auto max-w-2xl pb-20">
			<div class="mb-8">
				<h1 class="text-3xl font-bold">Event Notifications</h1>
				<p class="text-muted-foreground mt-1">Event invitations and updates</p>
			</div>

			<!-- Pending Invitations -->
			{#if invitations.length > 0}
				<div class="mb-8">
					<h2 class="mb-4 text-lg font-semibold">Pending Invitations ({invitations.length})</h2>
					<div class="space-y-3">
						{#each invitations as invitation (invitation.id)}
							<div
								class="glass-card bg-card overflow-hidden rounded-xl border border-white/10 transition-all duration-300 {removingIds.has(
									invitation.id
								)
									? 'scale-95 opacity-0'
									: ''}"
								out:fly={{ x: -100, duration: 300 }}
							>
								<!-- Clickable Event Info -->
								<button
									class="flex w-full gap-4 p-4 text-left transition-colors hover:bg-white/5"
									onclick={() => navigateToEvent(invitation.event.id)}
								>
									<div class="relative h-20 w-28 flex-shrink-0 overflow-hidden rounded-lg">
										<img
											src={invitation.event.cover_image ||
												'https://images.unsplash.com/photo-1540575467063-178a50c2df87?w=200'}
											alt=""
											class="h-full w-full object-cover"
										/>
									</div>
									<div class="flex-1">
										<div class="flex items-center gap-2 text-sm">
											<img
												src={invitation.inviter.avatar || 'https://github.com/shadcn.png'}
												alt=""
												class="h-5 w-5 rounded-full"
											/>
											<span class="text-muted-foreground">
												<span class="text-foreground font-medium"
													>{invitation.inviter.full_name || invitation.inviter.username}</span
												>
												invited you to
											</span>
										</div>
										<h3 class="mt-1 font-bold">{invitation.event.title}</h3>
										<div class="text-muted-foreground mt-1 flex items-center gap-3 text-sm">
											<span class="flex items-center gap-1">
												<Calendar size={12} />
												{new Date(invitation.event.start_date).toLocaleDateString(undefined, {
													month: 'short',
													day: 'numeric'
												})}
											</span>
											<span>{invitation.event.location || 'Online'}</span>
										</div>
										{#if invitation.message}
											<p class="text-muted-foreground mt-2 text-sm italic">
												"{invitation.message}"
											</p>
										{/if}
										<p class="text-muted-foreground mt-2 text-xs">
											{formatTime(invitation.created_at)}
										</p>
									</div>
								</button>

								<!-- Action Buttons -->
								<div class="flex gap-2 border-t border-white/10 p-3">
									<Button
										class="flex-1 gap-2"
										disabled={responding === invitation.id}
										onclick={() => handleRespond(invitation.id, true)}
									>
										{#if responding === invitation.id}
											<Loader2 class="h-4 w-4 animate-spin" />
										{:else}
											<Check size={16} />
										{/if}
										Accept & View Event
									</Button>
									<Button
										variant="ghost"
										class="text-muted-foreground gap-2 hover:text-red-400"
										disabled={responding === invitation.id}
										onclick={() => handleRespond(invitation.id, false)}
									>
										<X size={16} />
										Decline
									</Button>
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Empty State -->
			{#if !loading && invitations.length === 0}
				<div
					class="glass-card bg-card flex flex-col items-center justify-center rounded-xl border border-white/5 py-16"
				>
					<div
						class="bg-primary/10 text-primary mb-4 flex h-16 w-16 items-center justify-center rounded-full"
					>
						<Bell size={32} />
					</div>
					<h3 class="text-lg font-semibold">No notifications</h3>
					<p class="text-muted-foreground mt-1 text-center">
						You don't have any pending event invitations.<br />
						When friends invite you to events, they'll appear here.
					</p>
					<Button href="/events" variant="outline" class="mt-4">Discover Events</Button>
				</div>
			{/if}

			<!-- Loading State -->
			{#if loading}
				<div class="flex items-center justify-center py-16">
					<Loader2 class="animate-spin text-white" size={32} />
				</div>
			{/if}

			<!-- Tip Section -->
			<div class="mt-8 rounded-xl border border-white/10 bg-white/5 p-4">
				<div class="flex items-start gap-3">
					<UserPlus class="text-primary mt-0.5 shrink-0" size={20} />
					<div>
						<h4 class="font-semibold">Tip: Invite friends to your events</h4>
						<p class="text-muted-foreground text-sm">
							When you create or attend an event, you can invite your friends to join. Click the
							invite button on any event page to send invitations.
						</p>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>
