<script lang="ts">
	import { onMount } from 'svelte';
	import EventsSidebar from '$lib/components/events/EventsSidebar.svelte';
	import EventCard from '$lib/components/events/EventCard.svelte';
	import { MapPin, Loader2 } from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button';
	import { getEvents, type Event } from '$lib/api';

	let events: Event[] = $state([]);
	let loading = $state(true);

	onMount(async () => {
		try {
			// For now, fetching all events. Later pass location filter.
			const res = await getEvents(1, 20);
			events = res.events;
		} catch (err) {
			console.error('Failed to load local events:', err);
		} finally {
			loading = false;
		}
	});
</script>

<div class="bg-background text-foreground flex h-[calc(100vh-4rem)] w-full overflow-hidden">
	<EventsSidebar />

	<div class="flex-1 overflow-y-auto p-4 md:p-8">
		<div class="mx-auto max-w-5xl pb-20">
			<div class="mb-8 flex items-center justify-between">
				<h1 class="text-3xl font-bold">Local Events</h1>
				<Button variant="outline" class="gap-2">
					<MapPin size={16} /> Change Location
				</Button>
			</div>

			<!-- Map Placeholder -->
			<div
				class="bg-secondary/10 mb-8 flex h-64 w-full items-center justify-center rounded-xl border border-white/5"
			>
				<div class="text-muted-foreground text-center">
					<MapPin size={48} class="mx-auto mb-2 opacity-50" />
					<p>Map View Coming Soon</p>
				</div>
			</div>

			<!-- Events Grid -->
			<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
				<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
					{#if loading}
						<div class="col-span-full flex h-40 items-center justify-center">
							<Loader2 class="animate-spin text-white" size={32} />
						</div>
					{:else if events.length === 0}
						<div class="col-span-full py-10 text-center text-gray-400">No local events found.</div>
					{:else}
						{#each events as event}
							<EventCard {event} />
						{/each}
					{/if}
				</div>
			</div>
		</div>
	</div>
</div>
