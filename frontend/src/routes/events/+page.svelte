<script lang="ts">
	import { onMount } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import EventCard from '$lib/components/events/EventCard.svelte';
	import EventsSidebar from '$lib/components/events/EventsSidebar.svelte';
	import EventRecommendations from '$lib/components/events/EventRecommendations.svelte';
	import EventTrending from '$lib/components/events/EventTrending.svelte';
	import { Gift, ChevronRight, Loader2, Calendar, MapPin, Search, Filter } from '@lucide/svelte';
	import {
		getEvents,
		getBirthdays,
		getEventCategories,
		type Event,
		type BirthdayUser,
		type EventCategory,
		type RSVPStatus
	} from '$lib/api';

	let events: Event[] = $state([]);
	let loading = $state(true);
	let birthdays: { today: BirthdayUser[]; upcoming: BirthdayUser[] } = $state({
		today: [],
		upcoming: []
	});
	let categories: EventCategory[] = $state([]);

	// Filters
	let activeFilter = $state<'all' | 'today' | 'week' | 'weekend'>('all');
	let activeCategory = $state('');
	let page = $state(1);
	let total = $state(0);
	let hasMore = $state(false);

	onMount(async () => {
		await Promise.all([loadEvents(), loadBirthdays(), loadCategories()]);
	});

	async function loadEvents() {
		loading = true;
		try {
			const params: { category?: string; period?: string } = {};
			if (activeCategory) params.category = activeCategory;
			if (activeFilter !== 'all') params.period = activeFilter;

			const res = await getEvents(page, 20, params);
			events = res.events || [];
			total = res.total || 0;
			hasMore = events.length < total;
		} catch (err) {
			console.error('Failed to load events:', err);
		} finally {
			loading = false;
		}
	}

	async function loadBirthdays() {
		try {
			birthdays = await getBirthdays();
		} catch (err) {
			console.error('Failed to load birthdays:', err);
		}
	}

	async function loadCategories() {
		try {
			const res = await getEventCategories();
			categories = res.categories || [];
		} catch (err) {
			console.error('Failed to load categories:', err);
		}
	}

	async function setFilter(filter: typeof activeFilter) {
		activeFilter = filter;
		page = 1;
		await loadEvents();
	}

	async function setCategory(cat: string) {
		activeCategory = cat === activeCategory ? '' : cat;
		page = 1;
		await loadEvents();
	}

	async function loadMore() {
		page++;
		try {
			const params: { category?: string; period?: string } = {};
			if (activeCategory) params.category = activeCategory;
			if (activeFilter !== 'all') params.period = activeFilter;

			const res = await getEvents(page, 20, params);
			events = [...events, ...(res.events || [])];
			hasMore = events.length < total;
		} catch (err) {
			page--;
		}
	}

	function handleStatusChange(eventId: string, status: RSVPStatus) {
		events = events.map((e) => {
			if (e.id === eventId) {
				return { ...e, my_status: status };
			}
			return e;
		});
	}

	// Featured event (first event or placeholder)
	let featuredEvent = $derived(events[0]);

	const filterButtons = [
		{ value: 'all', label: 'All' },
		{ value: 'today', label: 'Today' },
		{ value: 'week', label: 'This Week' },
		{ value: 'weekend', label: 'Weekend' }
	] as const;

	const defaultCategories = [
		{
			name: 'Music',
			img: 'https://images.unsplash.com/photo-1514525253440-b393452e3726?auto=format&fit=crop&q=80&w=500'
		},
		{
			name: 'Sports',
			img: 'https://images.unsplash.com/photo-1461896836934-ffe607ba8211?auto=format&fit=crop&q=80&w=500'
		},
		{
			name: 'Tech',
			img: 'https://images.unsplash.com/photo-1531297484001-80022131f5a1?auto=format&fit=crop&q=80&w=500'
		},
		{
			name: 'Food',
			img: 'https://images.unsplash.com/photo-1555939594-58d7cb561ad1?auto=format&fit=crop&q=80&w=500'
		}
	];
</script>

<div class="bg-background text-foreground flex h-[calc(100vh-4rem)] w-full overflow-hidden">
	<EventsSidebar />

	<!-- Main Content Feed -->
	<div class="flex-1 overflow-y-auto p-4 md:p-8">
		<div class="mx-auto max-w-5xl pb-20">
			<!-- Mobile Header -->
			<div class="mb-6 md:hidden">
				<h1 class="text-3xl font-bold">Events</h1>
				<div class="mt-4 flex gap-2 overflow-x-auto pb-2">
					<Button href="/events" variant="secondary" size="sm" class="whitespace-nowrap"
						>Discover</Button
					>
					<Button href="/events/your-events" variant="ghost" size="sm" class="whitespace-nowrap"
						>Your Events</Button
					>
					<Button href="/events/local" variant="ghost" size="sm" class="whitespace-nowrap"
						>Local</Button
					>
				</div>
			</div>

			<!-- Featured Event / Hero -->
			{#if featuredEvent}
				<div class="mb-8">
					<h2 class="mb-4 text-xl font-bold">Featured Event</h2>
					<a href="/events/{featuredEvent.id}" class="relative block overflow-hidden rounded-2xl">
						<div
							class="absolute inset-0 z-10 bg-gradient-to-t from-black/80 via-black/40 to-transparent"
						></div>
						<img
							src={featuredEvent.cover_image ||
								'https://images.unsplash.com/photo-1492684223066-81342ee5ff30?w=800'}
							alt={featuredEvent.title}
							class="h-64 w-full object-cover transition-transform duration-500 hover:scale-105 md:h-80"
						/>
						<div class="absolute bottom-0 left-0 z-20 p-6 md:p-8">
							<div
								class="mb-2 flex items-center gap-2 text-sm font-bold uppercase tracking-wider text-red-400"
							>
								<Calendar size={14} />
								{new Date(featuredEvent.start_date).toLocaleDateString(undefined, {
									weekday: 'short',
									month: 'short',
									day: 'numeric'
								})}
							</div>
							<h1 class="mb-2 text-3xl font-black text-white md:text-4xl">{featuredEvent.title}</h1>
							<div class="mb-4 flex items-center gap-4 text-gray-300">
								<span class="flex items-center gap-1">
									<MapPin size={14} />
									{featuredEvent.is_online ? 'Online' : featuredEvent.location || 'TBA'}
								</span>
								<span>{featuredEvent.stats.going_count} going</span>
							</div>
						</div>
					</a>
				</div>
			{/if}

			<!-- Trending & Recommendations -->
			<div class="mb-10 space-y-10">
				<EventTrending />
				<EventRecommendations />
			</div>

			<!-- Birthdays Widget -->
			{#if birthdays.today.length > 0}
				<a
					href="/events/birthdays"
					class="mb-8 block rounded-xl border border-pink-500/20 bg-gradient-to-r from-pink-500/10 to-purple-500/10 p-4 backdrop-blur-sm transition-colors hover:from-pink-500/15 hover:to-purple-500/15"
				>
					<div class="flex items-center gap-3">
						<div
							class="flex h-10 w-10 items-center justify-center rounded-full bg-pink-500/20 text-pink-400"
						>
							<Gift size={20} />
						</div>
						<div class="flex-1">
							<h3 class="font-bold">
								{#if birthdays.today.length === 1}
									{birthdays.today[0].full_name} has a birthday today!
								{:else}
									{birthdays.today[0].full_name} and {birthdays.today.length - 1} others have birthdays
									today.
								{/if}
							</h3>
							<p class="text-muted-foreground text-sm">Wish them a happy birthday!</p>
						</div>
						<div class="flex -space-x-2">
							{#each birthdays.today.slice(0, 3) as person}
								<img
									src={person.avatar || 'https://github.com/shadcn.png'}
									alt=""
									class="border-background h-8 w-8 rounded-full border-2"
								/>
							{/each}
						</div>
					</div>
				</a>
			{/if}

			<!-- Filter Tabs -->
			<div class="mb-6 flex flex-wrap items-center gap-2">
				{#each filterButtons as filter}
					<Button
						variant={activeFilter === filter.value ? 'secondary' : 'ghost'}
						size="sm"
						class="h-8"
						onclick={() => setFilter(filter.value)}
					>
						{filter.label}
					</Button>
				{/each}

				{#if activeCategory}
					<div
						class="bg-primary/20 text-primary flex items-center gap-1 rounded-full px-3 py-1 text-sm"
					>
						{activeCategory}
						<button onclick={() => setCategory('')} class="ml-1 hover:text-white">×</button>
					</div>
				{/if}
			</div>

			<!-- Events Grid -->
			<div class="mb-8">
				<div class="mb-4 flex items-center justify-between">
					<h2 class="text-xl font-bold">
						{activeFilter === 'all'
							? 'Upcoming Events'
							: `Events ${activeFilter === 'today' ? 'Today' : activeFilter === 'week' ? 'This Week' : 'This Weekend'}`}
						{activeCategory ? ` • ${activeCategory}` : ''}
					</h2>
					<span class="text-muted-foreground text-sm">{total} events</span>
				</div>

				<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
					{#if loading && events.length === 0}
						{#each Array(6) as _}
							<div class="h-64 w-full animate-pulse rounded-xl bg-white/5"></div>
						{/each}
					{:else if events.length === 0}
						<div
							class="col-span-full flex h-40 flex-col items-center justify-center gap-2 text-gray-400"
						>
							<Calendar size={32} class="opacity-50" />
							<p>No events found</p>
							<Button
								variant="outline"
								size="sm"
								onclick={() => {
									activeFilter = 'all';
									activeCategory = '';
									loadEvents();
								}}
							>
								Clear Filters
							</Button>
						</div>
					{:else}
						{#each events as event}
							<EventCard {event} onStatusChange={handleStatusChange} />
						{/each}
					{/if}
				</div>

				{#if hasMore && !loading}
					<div class="mt-6 text-center">
						<Button variant="outline" onclick={loadMore}>Load More</Button>
					</div>
				{/if}

				{#if loading && events.length > 0}
					<div class="mt-6 flex justify-center">
						<Loader2 class="animate-spin text-white" size={24} />
					</div>
				{/if}
			</div>

			<!-- Category Discovery -->
			<div class="mb-8">
				<h2 class="mb-4 text-xl font-bold">Explore by Category</h2>
				<div class="grid grid-cols-2 gap-4 md:grid-cols-4">
					{#each defaultCategories as cat}
						<button
							class="group relative aspect-video cursor-pointer overflow-hidden rounded-xl {activeCategory ===
							cat.name
								? 'ring-primary ring-2'
								: ''}"
							onclick={() => setCategory(cat.name)}
						>
							<div
								class="absolute inset-0 z-10 bg-black/40 transition-colors group-hover:bg-black/30"
							></div>
							<img
								src={cat.img}
								alt={cat.name}
								class="h-full w-full object-cover transition-transform duration-500 group-hover:scale-110"
							/>
							<span class="absolute bottom-3 left-4 z-20 text-lg font-bold text-white"
								>{cat.name}</span
							>
							{#if categories.find((c) => c.name === cat.name)}
								<span
									class="absolute right-3 top-3 z-20 rounded-full bg-black/50 px-2 py-0.5 text-xs text-white"
								>
									{categories.find((c) => c.name === cat.name)?.count || 0}
								</span>
							{/if}
						</button>
					{/each}
				</div>
			</div>
		</div>
	</div>
</div>
