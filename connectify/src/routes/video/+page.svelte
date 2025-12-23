<script lang="ts">
	import { Search, MonitorPlay, Clapperboard, Radio, Bookmark, Settings, Tv } from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button';
	import VideoCard from '$lib/components/video/VideoCard.svelte';

	let activeTab = $state('home');
	let searchQuery = $state('');

	const sidebarItems = [
		{ id: 'home', label: 'Home', icon: MonitorPlay },
		{ id: 'live', label: 'Live', icon: Radio },
		{ id: 'reels', label: 'Reels', icon: Clapperboard },
		{ id: 'shows', label: 'Shows', icon: Tv },
		{ id: 'saved', label: 'Saved Video', icon: Bookmark }
	];

	// Mock Data
	const mockVideos = [
		{
			id: '1',
			title: 'Exploring the hidden gems of the Swiss Alps 🏔️ | Travel Vlog',
			video_url: 'https://cdn.pixabay.com/video/2023/10/26/186638-878456361_large.mp4',
			thumbnail:
				'https://images.pexels.com/photos/1761279/pexels-photo-1761279.jpeg?auto=compress&cs=tinysrgb&w=1260&h=750&dpr=2', // Placeholder
			views: '1.2M',
			date: '2 days ago',
			author: {
				name: 'Wanderlust Travels',
				avatar:
					'https://images.pexels.com/photos/220453/pexels-photo-220453.jpeg?auto=compress&cs=tinysrgb&w=150'
			},
			stats: { likes: '45K', comments: '1.2K', shares: '5K' }
		},
		{
			id: '2',
			title: 'Best Street Food in Tokyo! 🍜🇯🇵',
			video_url: 'https://cdn.pixabay.com/video/2024/05/24/213564_large.mp4',
			thumbnail:
				'https://images.pexels.com/photos/2664417/pexels-photo-2664417.jpeg?auto=compress&cs=tinysrgb&w=1260&h=750&dpr=2',
			views: '890K',
			date: '5 hours ago',
			author: {
				name: 'Foodie Adventures',
				avatar:
					'https://images.pexels.com/photos/415829/pexels-photo-415829.jpeg?auto=compress&cs=tinysrgb&w=150'
			},
			stats: { likes: '32K', comments: '800', shares: '2.1K' }
		},
		{
			id: '3',
			title: 'Satisfying Kinetic Sand Mixing #shorts',
			video_url: 'https://cdn.pixabay.com/video/2023/10/22/186007-876939943_large.mp4',
			thumbnail:
				'https://images.pexels.com/photos/1108099/pexels-photo-1108099.jpeg?auto=compress&cs=tinysrgb&w=1260&h=750&dpr=2',
			views: '5M',
			date: '1 week ago',
			author: {
				name: 'Oddly Satisfying',
				avatar:
					'https://images.pexels.com/photos/774909/pexels-photo-774909.jpeg?auto=compress&cs=tinysrgb&w=150'
			},
			stats: { likes: '200K', comments: '5K', shares: '20K' }
		}
	];
</script>

<div class="flex min-h-screen bg-transparent pt-14 font-sans">
	<!-- Left Sidebar (Sticky) -->
	<aside
		class="bg-background/50 fixed left-0 top-14 hidden h-[calc(100vh-56px)] w-[360px] overflow-y-auto border-r border-white/10 p-4 backdrop-blur-xl lg:block"
	>
		<div class="mb-6 flex items-center justify-between">
			<h1 class="text-2xl font-bold">Video</h1>
			<Button variant="ghost" size="icon" class="rounded-full bg-white/5 hover:bg-white/10">
				<Settings size={20} />
			</Button>
		</div>

		<!-- Search Input -->
		<div class="relative mb-6">
			<Search class="text-muted-foreground absolute left-3 top-1/2 -translate-y-1/2" size={18} />
			<input
				type="text"
				placeholder="Search videos"
				class="focus:ring-primary/50 w-full rounded-full bg-white/10 py-2 pl-10 pr-4 text-sm outline-none focus:ring-2"
				bind:value={searchQuery}
			/>
		</div>

		<nav class="space-y-1">
			{#each sidebarItems as item}
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
	</aside>

	<!-- Main Content Area -->
	<main class="flex flex-1 justify-center p-4 md:p-8 lg:pl-[360px]">
		<div class="w-full max-w-2xl space-y-6">
			<!-- Featured / New for you -->
			<div class="mb-4">
				<h2 class="mb-2 text-xl font-bold">For You</h2>
			</div>

			{#each mockVideos as video (video.id)}
				<VideoCard {video} />
			{/each}

			<div class="text-muted-foreground py-8 text-center">
				<p>You're all caught up!</p>
			</div>
		</div>
	</main>
</div>
