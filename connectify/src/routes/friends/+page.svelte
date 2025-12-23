<script lang="ts">
	import { Users, UserPlus, Sparkles, Gift, List, Settings, ChevronRight } from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button';
	import FriendListTab from '$lib/components/friends/FriendListTab.svelte';
	import FriendRequestsTab from '$lib/components/friends/FriendRequestsTab.svelte';
	import Skeleton from '$lib/components/ui/skeleton/Skeleton.svelte';

	let activeTab = $state('home');

	const menuItems = [
		{ id: 'home', label: 'Home', icon: Users },
		{ id: 'requests', label: 'Friend Requests', icon: UserPlus },
		{ id: 'suggestions', label: 'Suggestions', icon: Sparkles },
		{ id: 'all', label: 'All Friends', icon: Users },
		{ id: 'birthdays', label: 'Birthdays', icon: Gift },
		{ id: 'lists', label: 'Custom Lists', icon: List }
	];
</script>

<div class="flex min-h-screen bg-transparent pt-14 font-sans">
	<!-- Left Sidebar -->
	<aside
		class="bg-background/50 fixed left-0 top-14 hidden h-[calc(100vh-56px)] w-[360px] overflow-y-auto border-r border-white/10 p-4 backdrop-blur-xl lg:block"
	>
		<div class="mb-6 flex items-center justify-between">
			<h1 class="text-2xl font-bold">Friends</h1>
			<Button variant="ghost" size="icon" class="rounded-full bg-white/5 hover:bg-white/10">
				<Settings size={20} />
			</Button>
		</div>

		<nav class="space-y-2">
			{#each menuItems as item}
				<button
					class="flex w-full items-center justify-between rounded-lg p-3 transition-colors {activeTab ===
					item.id
						? 'bg-primary/10 text-primary'
						: 'hover:bg-white/5'}"
					onclick={() => (activeTab = item.id)}
				>
					<div class="flex items-center space-x-3">
						<div
							class="rounded-full bg-white/10 p-2 {activeTab === item.id
								? 'bg-primary text-white'
								: ''}"
						>
							<item.icon size={20} />
						</div>
						<span class="text-lg font-medium">{item.label}</span>
					</div>
					<ChevronRight size={20} class="opacity-50" />
				</button>
			{/each}
		</nav>
	</aside>

	<!-- Main Content -->
	<main class="flex-1 p-4 md:p-8 lg:pl-[360px]">
		<div class="mx-auto max-w-5xl">
			{#if activeTab === 'home'}
				<div class="space-y-8">
					<!-- Home View: Aggregated content -->
					<div>
						<div class="mb-4 flex items-center justify-between">
							<h2 class="text-xl font-bold">Friend Requests</h2>
							<button class="text-primary hover:underline" onclick={() => (activeTab = 'requests')}
								>See all</button
							>
						</div>
						<FriendRequestsTab limit={5} />
					</div>
					<hr class="border-white/10" />
					<div>
						<div class="mb-4 flex items-center justify-between">
							<h2 class="text-xl font-bold">All Friends</h2>
							<button class="text-primary hover:underline" onclick={() => (activeTab = 'all')}
								>See all</button
							>
						</div>
						<FriendListTab limit={5} />
					</div>
				</div>
			{:else if activeTab === 'requests'}
				<div>
					<h2 class="mb-6 text-2xl font-bold">Friend Requests</h2>
					<FriendRequestsTab />
				</div>
			{:else if activeTab === 'all'}
				<div>
					<h2 class="mb-6 text-2xl font-bold">All Friends</h2>
					<FriendListTab />
				</div>
			{:else}
				<div class="flex flex-col items-center justify-center py-20 text-center">
					<div class="bg-primary/10 mb-4 rounded-full p-6">
						<Sparkles size={48} class="text-primary" />
					</div>
					<h3 class="mb-2 text-2xl font-bold">Coming Soon</h3>
					<p class="text-muted-foreground">This feature is actively being developed.</p>
					<Button variant="outline" class="mt-6" onclick={() => (activeTab = 'home')}
						>Go Back Home</Button
					>
				</div>
			{/if}
		</div>
	</main>
</div>
