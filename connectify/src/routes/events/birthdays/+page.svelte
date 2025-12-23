<script lang="ts">
	import { onMount } from 'svelte';
	import EventsSidebar from '$lib/components/events/EventsSidebar.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Gift, Send, Loader2 } from '@lucide/svelte';
	import { getBirthdays, type BirthdayUser } from '$lib/api';

	let loading = $state(true);
	let today: BirthdayUser[] = $state([]);
	let upcoming: BirthdayUser[] = $state([]);

	onMount(async () => {
		try {
			const res = await getBirthdays();
			today = res.today;
			upcoming = res.upcoming;
		} catch (err) {
			console.error('Failed to load birthdays:', err);
		} finally {
			loading = false;
		}
	});
</script>

<div class="bg-background text-foreground flex h-[calc(100vh-4rem)] w-full overflow-hidden">
	<EventsSidebar />

	<div class="flex-1 overflow-y-auto p-4 md:p-8">
		<div class="mx-auto max-w-2xl pb-20">
			<h1 class="mb-8 text-3xl font-bold">Birthdays</h1>

			{#if loading}
				<div class="flex h-40 items-center justify-center">
					<Loader2 class="animate-spin text-white" size={32} />
				</div>
			{:else}
				<!-- Today -->
				<div class="mb-10">
					<h2 class="mb-4 flex items-center gap-2 text-xl font-bold">
						<Gift class="text-red-500" /> Today's Birthdays
					</h2>
					<div class="glass-card bg-card divide-y divide-white/5 rounded-xl border border-white/5">
						{#if today.length === 0}
							<div class="p-6 text-center text-gray-400">No birthdays today.</div>
						{:else}
							{#each today as user}
								<div class="flex items-center justify-between p-4">
									<div class="flex items-center gap-4">
										<img
											src={user.avatar || 'https://github.com/shadcn.png'}
											alt={user.full_name}
											class="h-12 w-12 rounded-full object-cover"
										/>
										<div>
											<div class="font-bold">{user.full_name || user.username}</div>
											<div class="text-muted-foreground text-sm">Turning {user.age}</div>
										</div>
									</div>
									<div class="flex gap-2">
										<Button size="sm" class="gap-2">
											<Send size={16} /> Wish Happy Birthday
										</Button>
									</div>
								</div>
							{/each}
						{/if}
					</div>
				</div>

				<!-- Upcoming -->
				<div>
					<h2 class="text-muted-foreground mb-4 text-xl font-bold">Upcoming Birthdays</h2>
					<div class="glass-card bg-card divide-y divide-white/5 rounded-xl border border-white/5">
						{#if upcoming.length === 0}
							<div class="p-6 text-center text-gray-400">No upcoming birthdays soon.</div>
						{:else}
							{#each upcoming as user}
								<div class="flex items-center justify-between p-4">
									<div class="flex items-center gap-4">
										<img
											src={user.avatar || 'https://github.com/shadcn.png'}
											alt={user.full_name}
											class="h-10 w-10 rounded-full object-cover grayscale"
										/>
										<div>
											<div class="font-semibold">{user.full_name || user.username}</div>
											<div class="text-muted-foreground text-sm">{user.date}</div>
										</div>
									</div>
								</div>
							{/each}
						{/if}
					</div>
				</div>
			{/if}
		</div>
	</div>
</div>
