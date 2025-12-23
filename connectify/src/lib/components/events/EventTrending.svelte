<script lang="ts">
	import { onMount } from 'svelte';
	import { getTrendingEvents, type TrendingEvent } from '$lib/api';
	import EventCard from './EventCard.svelte';
	import { Loader2, Flame, TrendingUp } from '@lucide/svelte';

	let trending = $state<TrendingEvent[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		try {
			trending = await getTrendingEvents(6);
		} catch (e) {
			console.error('Failed to load trending events:', e);
			error = 'Failed to load trending events';
		} finally {
			loading = false;
		}
	});
</script>

<div class="space-y-6">
	<div class="flex items-center gap-3">
		<div class="rounded-full bg-orange-500/20 p-2 text-orange-400 ring-1 ring-orange-500/50">
			<Flame size={20} />
		</div>
		<h2 class="text-2xl font-bold text-white">Trending Now</h2>
	</div>

	{#if loading}
		<div class="flex h-64 items-center justify-center rounded-2xl border border-white/5 bg-white/5">
			<Loader2 class="h-8 w-8 animate-spin text-orange-400" />
		</div>
	{:else if error}
		<div class="rounded-2xl border border-red-500/20 bg-red-500/10 p-8 text-center text-red-400">
			{error}
		</div>
	{:else if trending.length === 0}
		<div class="rounded-2xl border border-white/5 bg-white/5 p-12 text-center text-gray-400">
			No trending events right now. Be the first to start a trend!
		</div>
	{:else}
		<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
			{#each trending as item}
				{#if item.event}
					<div class="group relative">
						<EventCard event={item.event} />
						<!-- Position Badge -->
						<div class="pointer-events-none absolute left-3 top-3 z-20">
							<div
								class="flex items-center gap-1.5 rounded-full border border-orange-400/30 bg-orange-600/90 px-3 py-1 text-xs font-bold text-white shadow-lg backdrop-blur-md"
							>
								<TrendingUp size={10} />
								Hot
							</div>
						</div>
					</div>
				{/if}
			{/each}
		</div>
	{/if}
</div>
