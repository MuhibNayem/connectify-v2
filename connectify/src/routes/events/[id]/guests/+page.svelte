<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import EventsSidebar from '$lib/components/events/EventsSidebar.svelte';
	import EventGuestList from '$lib/components/events/EventGuestList.svelte';
	import { Button } from '$lib/components/ui/button';
	import { ArrowLeft, Loader2, Users } from '@lucide/svelte';
	import { getEvent, type Event } from '$lib/api';

	let event: Event | null = $state(null);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		try {
			const id = $page.params.id;
			if (id) {
				event = await getEvent(id);
			}
		} catch (err) {
			console.error('Failed to load event:', err);
			error = 'Failed to load event.';
		} finally {
			loading = false;
		}
	});
</script>

<div class="bg-background text-foreground flex h-[calc(100vh-4rem)] w-full overflow-hidden">
	<EventsSidebar />

	<div class="flex-1 overflow-y-auto">
		<div class="mx-auto max-w-3xl px-4 py-8">
			{#if loading}
				<div class="flex h-[50vh] items-center justify-center">
					<Loader2 class="animate-spin text-white" size={48} />
				</div>
			{:else if error || !event}
				<div class="flex h-[50vh] flex-col items-center justify-center gap-4 text-center">
					<h2 class="text-2xl font-bold">Event not found</h2>
					<p class="text-muted-foreground">{error}</p>
					<Button href="/events" variant="outline">Back to Events</Button>
				</div>
			{:else}
				<!-- Header -->
				<div class="mb-8">
					<div class="flex items-center gap-4">
						<Button variant="ghost" size="icon" href="/events/{event.id}">
							<ArrowLeft size={20} />
						</Button>
						<div>
							<h1 class="text-2xl font-bold">Guest List</h1>
							<p class="text-muted-foreground">{event.title}</p>
						</div>
					</div>

					<!-- Stats Summary -->
					<div class="mt-6 grid grid-cols-3 gap-4">
						<div class="rounded-xl border border-white/10 bg-white/5 p-4 text-center">
							<div class="text-2xl font-bold text-green-400">{event.stats.going_count}</div>
							<div class="text-muted-foreground text-sm">Going</div>
						</div>
						<div class="rounded-xl border border-white/10 bg-white/5 p-4 text-center">
							<div class="text-2xl font-bold text-yellow-400">{event.stats.interested_count}</div>
							<div class="text-muted-foreground text-sm">Interested</div>
						</div>
						<div class="rounded-xl border border-white/10 bg-white/5 p-4 text-center">
							<div class="text-2xl font-bold text-blue-400">{event.stats.invited_count}</div>
							<div class="text-muted-foreground text-sm">Invited</div>
						</div>
					</div>
				</div>

				<!-- Full Guest List -->
				<div class="glass-card bg-card rounded-xl border border-white/5 p-6">
					<EventGuestList eventId={event.id} isHost={event.is_host} />
				</div>
			{/if}
		</div>
	</div>
</div>
