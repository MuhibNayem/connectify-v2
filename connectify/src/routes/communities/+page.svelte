<script lang="ts">
	import { onMount } from 'svelte';
	import { getCommunities, getUserCommunities, type Community } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import { Plus, Users, Search, Compass, Settings, ChevronRight, Newspaper } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import Skeleton from '$lib/components/ui/skeleton/Skeleton.svelte';

	let activeTab = $state('discover');
	let communities = $state<Community[]>([]);
	let userCommunities = $state<Community[]>([]);
	let loading = $state(true);
	let searchQuery = $state('');
	let searchTimeout: any;

	// Pagination
	let currentPage = $state(1);
	let totalCommunities = $state(0);
	const limit = 9;

	async function loadCommunities(page: number = 1, query: string = '') {
		try {
			loading = true;
			const res = await getCommunities(page, limit, query);
			communities = res.communities;
			totalCommunities = res.total;
			currentPage = res.page;
		} catch (error) {
			console.error('Failed to load communities:', error);
		} finally {
			loading = false;
		}
	}

	function handleSearch() {
		clearTimeout(searchTimeout);
		searchTimeout = setTimeout(() => {
			currentPage = 1;
			loadCommunities(1, searchQuery);
		}, 300);
	}

	function handlePageChange(newPage: number) {
		if (newPage < 1 || newPage > Math.ceil(totalCommunities / limit)) return;
		loadCommunities(newPage, searchQuery);
	}

	onMount(async () => {
		try {
			const [_, userRes] = await Promise.all([loadCommunities(1), getUserCommunities()]);
			userCommunities = userRes;
			if (userCommunities.length > 0) {
				activeTab = 'feed'; // Default to feed if user has groups (Feed logic to be implemented, showing Your Groups for now)
				activeTab = 'your_groups';
			}
		} catch (error) {
			console.error('Failed to load initial data:', error);
		}
	});

	const menuItems = [
		{ id: 'discover', label: 'Discover', icon: Compass },
		{ id: 'your_groups', label: 'Your Groups', icon: Users }
		// { id: 'feed', label: 'Feed', icon: Newspaper } // Future implementation
	];
</script>

<div class="flex min-h-screen bg-transparent pt-14 font-sans">
	<!-- Left Sidebar -->
	<aside
		class="bg-background/50 fixed left-0 top-14 hidden h-[calc(100vh-56px)] w-[360px] overflow-y-auto border-r border-white/10 p-4 backdrop-blur-xl lg:block"
	>
		<div class="mb-6 flex items-center justify-between">
			<h1 class="text-2xl font-bold">Groups</h1>
			<Button variant="ghost" size="icon" class="rounded-full bg-white/5 hover:bg-white/10">
				<Settings size={20} />
			</Button>
		</div>

		<!-- Search Sidebar Input -->
		<div class="relative mb-6">
			<Search class="text-muted-foreground absolute left-3 top-1/2 -translate-y-1/2" size={18} />
			<input
				type="text"
				placeholder="Search groups"
				class="focus:ring-primary/50 w-full rounded-full bg-white/10 py-2 pl-10 pr-4 text-sm outline-none focus:ring-2"
				bind:value={searchQuery}
				oninput={handleSearch}
				onclick={() => (activeTab = 'discover')}
			/>
		</div>

		<nav class="mb-6 space-y-1">
			{#each menuItems as item}
				<button
					class="flex w-full items-center space-x-3 rounded-lg p-3 transition-colors {activeTab ===
					item.id
						? 'bg-primary/10 text-primary'
						: 'hover:bg-white/5'}"
					onclick={() => (activeTab = item.id)}
				>
					<div
						class="rounded-full bg-white/10 p-2 {activeTab === item.id
							? 'bg-primary text-white'
							: ''}"
					>
						<item.icon size={20} />
					</div>
					<span class="text-lg font-medium">{item.label}</span>
				</button>
			{/each}
		</nav>

		<Button class="mb-6 w-full gap-2" onclick={() => goto('/communities/create')}>
			<Plus size={20} />
			Create New Group
		</Button>

		<hr class="mb-4 border-white/10" />

		<!-- Your Groups Snippet List -->
		<h3 class="mb-2 px-2 text-lg font-semibold">Groups You've Joined</h3>
		<div class="space-y-1">
			{#each userCommunities.slice(0, 5) as group}
				<a
					href="/communities/{group.id}"
					class="flex items-center space-x-3 rounded-lg p-2 hover:bg-white/5"
				>
					<div class="h-10 w-10 overflow-hidden rounded-lg bg-gray-700">
						{#if group.avatar}
							<img src={group.avatar} alt={group.name} class="h-full w-full object-cover" />
						{:else}
							<div class="flex h-full w-full items-center justify-center font-bold text-white/50">
								{group.name[0]}
							</div>
						{/if}
					</div>
					<div class="min-w-0 flex-1">
						<p class="truncate font-medium">{group.name}</p>
						<p class="text-muted-foreground truncate text-xs">Last active recently</p>
					</div>
				</a>
			{/each}
			{#if userCommunities.length > 5}
				<button
					class="text-primary w-full p-2 text-left text-sm hover:underline"
					onclick={() => (activeTab = 'your_groups')}
				>
					See more
				</button>
			{/if}
		</div>
	</aside>

	<!-- Main Content -->
	<main class="flex-1 p-4 md:p-8 lg:pl-[360px]">
		<div class="mx-auto max-w-6xl">
			{#if activeTab === 'discover'}
				<div class="mb-4">
					<h2 class="mb-1 text-2xl font-bold">Discover</h2>
					<p class="text-muted-foreground">Suggested groups for you</p>
				</div>

				{#if loading}
					<div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
						{#each Array(6) as _}
							<div class="glass-card h-64 animate-pulse rounded-xl"></div>
						{/each}
					</div>
				{:else if communities.length === 0}
					<div class="glass-panel text-muted-foreground p-12 text-center">
						No communities found matching "{searchQuery}"
					</div>
				{:else}
					<div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
						{#each communities as group}
							<a
								href="/communities/{group.id}"
								class="glass-card group flex flex-col overflow-hidden rounded-xl transition-transform hover:scale-[1.01]"
							>
								<!-- Cover -->
								<div class="relative h-32 bg-gray-800">
									{#if group.cover_image}
										<img
											src={group.cover_image}
											alt={group.name}
											class="h-full w-full object-cover"
										/>
									{:else}
										<div class="h-full w-full bg-gradient-to-r from-blue-500 to-purple-600"></div>
									{/if}
								</div>

								<!-- Info -->
								<div class="flex flex-1 flex-col p-4">
									<h3 class="group-hover:text-primary mb-1 text-lg font-bold transition-colors">
										{group.name}
									</h3>
									<div class="text-muted-foreground mb-3 flex items-center gap-2 text-xs">
										<span>{group.stats.member_count} members</span>
										<span>•</span>
										<span>{group.privacy || 'Public'}</span>
									</div>
									<p class="text-muted-foreground mb-4 line-clamp-2 text-sm">{group.description}</p>
									<Button variant="secondary" class="mt-auto w-full">View Group</Button>
								</div>
							</a>
						{/each}
					</div>
				{/if}
			{:else if activeTab === 'your_groups'}
				<div class="mb-4">
					<h2 class="mb-1 text-2xl font-bold">Your Groups</h2>
					<p class="text-muted-foreground">Groups you've joined or manage</p>
				</div>

				<div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
					{#each userCommunities as group}
						<a
							href="/communities/{group.id}"
							class="glass-card group flex flex-col overflow-hidden rounded-xl transition-transform hover:scale-[1.01]"
						>
							<div class="relative h-32 bg-gray-800">
								{#if group.cover_image}
									<img
										src={group.cover_image}
										alt={group.name}
										class="h-full w-full object-cover"
									/>
								{:else}
									<div class="h-full w-full bg-gradient-to-r from-emerald-500 to-teal-600"></div>
								{/if}
							</div>
							<div class="flex flex-1 flex-col p-4">
								<h3 class="group-hover:text-primary mb-1 text-lg font-bold transition-colors">
									{group.name}
								</h3>
								<div class="text-muted-foreground mb-3 text-xs">
									{group.stats.member_count} members
								</div>
								<Button variant="secondary" class="mt-auto w-full">Visit Group</Button>
							</div>
						</a>
					{/each}
				</div>
			{/if}
		</div>
	</main>
</div>
