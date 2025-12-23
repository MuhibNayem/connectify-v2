<script lang="ts">
	import { onMount } from 'svelte';
	import { getEventRecommendations, type EventRecommendation } from '$lib/api';
	import EventCard from './EventCard.svelte';
	import { Loader2, Sparkles } from '@lucide/svelte';

	let recommendations = $state<EventRecommendation[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		try {
			recommendations = await getEventRecommendations(6);
		} catch (e) {
			console.error('Failed to load recommendations:', e);
			error = 'Failed to load recommendations';
		} finally {
			loading = false;
		}
	});
</script>

<div class="space-y-6">
	<div class="flex items-center gap-3">
		<div class="rounded-full bg-purple-500/20 p-2 text-purple-400 ring-1 ring-purple-500/50">
			<Sparkles size={20} />
		</div>
		<h2 class="text-2xl font-bold text-white">Recommended for You</h2>
	</div>

	{#if loading}
		<div class="flex h-64 items-center justify-center rounded-2xl border border-white/5 bg-white/5">
			<Loader2 class="h-8 w-8 animate-spin text-purple-400" />
		</div>
	{:else if error}
		<div class="rounded-2xl border border-red-500/20 bg-red-500/10 p-8 text-center text-red-400">
			{error}
		</div>
	{:else if recommendations.length === 0}
		<div class="rounded-2xl border border-white/5 bg-white/5 p-12 text-center text-gray-400">
			<p class="text-lg font-medium">No recommendations found yet.</p>
			<p class="text-sm opacity-60">Connect with more friends to see personalized events.</p>
		</div>
	{:else}
		<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
			{#each recommendations as rec}
				{#if rec.event}
					<div class="group relative">
						<EventCard event={rec.event} />
						<!-- Reason Badge -->
						{#if rec.reason}
							<div class="pointer-events-none absolute left-3 top-3 z-20">
								<div
									class="flex items-center gap-1.5 rounded-full border border-purple-400/30 bg-purple-600/90 px-3 py-1 text-xs font-bold text-white shadow-lg backdrop-blur-md"
								>
									<Sparkles size={10} />
									{rec.reason}
								</div>
							</div>
						{/if}
					</div>
				{/if}
			{/each}
		</div>
	{/if}
</div>
