<script lang="ts">
	import { page } from '$app/stores';
	import { auth } from '$lib/stores/auth.svelte';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import {
		Users,
		Globe,
		Calendar,
		ShoppingBag,
		Settings,
		Bookmark,
		Clock,
		ChevronDown,
		MonitorPlay
	} from '@lucide/svelte';

	let currentUser = $derived(auth.state.user);

	// Facebook-style Left Sidebar Items
	const mainNavItems = [
		{
			href: '/friends',
			label: 'Friends',
			icon: Users,
			color: 'text-blue-500',
			bg: 'bg-transparent'
		}, // FB uses specific icons, we'll simulate
		{
			href: '/communities',
			label: 'Communities',
			icon: Users,
			color: 'text-white',
			bg: 'bg-blue-500'
		}, // Groups often circular blue
		{
			href: '/marketplace',
			label: 'Marketplace',
			icon: ShoppingBag,
			color: 'text-white',
			bg: 'bg-blue-500'
		},
		{ href: '/events', label: 'Events', icon: Calendar, color: 'text-white', bg: 'bg-red-500' },
		{ href: '/memories', label: 'Memories', icon: Clock, color: 'text-white', bg: 'bg-blue-400' },
		{ href: '/saved', label: 'Saved', icon: Bookmark, color: 'text-white', bg: 'bg-purple-500' },
		{ href: '/video', label: 'Video', icon: MonitorPlay, color: 'text-white', bg: 'bg-blue-500' },
		{
			href: '/settings',
			label: 'Settings',
			icon: Settings,
			color: 'text-foreground',
			bg: 'bg-secondary'
		}
	];

	// Mock Shortcuts
	const shortcuts = [
		{ id: 1, name: 'Svelte Developers', image: 'https://github.com/sveltejs.png' },
		{ id: 2, name: 'Tailwind CSS', image: 'https://github.com/tailwindlabs.png' },
		{
			id: 3,
			name: 'Web Design Trends',
			image: 'https://images.unsplash.com/photo-1507238691740-187a5b1d37b8?w=50&h=50&fit=crop'
		}
	];

	function isActive(path: string) {
		return $page.url.pathname === path;
	}
</script>

<div class="space-y-4">
	<!-- User Profile Link -->
	{#if currentUser}
		<a
			href={`/profile/${currentUser.id}`}
			class="flex items-center space-x-3 rounded-lg border-transparent p-2 transition-all hover:bg-black/5 dark:hover:bg-white/10"
		>
			<Avatar class="h-9 w-9 border border-black/10">
				<AvatarImage src={currentUser.avatar} alt={currentUser.username} />
				<AvatarFallback>{currentUser.username.charAt(0).toUpperCase()}</AvatarFallback>
			</Avatar>
			<span class="text-foreground font-semibold"
				>{currentUser.full_name || currentUser.username}</span
			>
		</a>
	{/if}

	<!-- Main Navigation Links -->
	<nav class="space-y-1">
		{#each mainNavItems as item}
			<a
				href={item.href}
				class="group flex items-center space-x-3 rounded-lg p-2 transition-all hover:bg-black/5 dark:hover:bg-white/10 {isActive(
					item.href
				)
					? 'bg-black/5 dark:bg-white/10'
					: ''}"
			>
				{#if item.label === 'Friends'}
					<!-- Special style for Friends roughly matching generic icon if needed, or consistent circle -->
					<div class="flex h-9 w-9 items-center justify-center rounded-full bg-blue-500 text-white">
						<svelte:component this={item.icon} size={20} />
					</div>
				{:else if item.bg !== 'bg-transparent'}
					<div
						class={`flex h-9 w-9 items-center justify-center rounded-full ${item.bg} ${item.color}`}
					>
						<svelte:component this={item.icon} size={20} />
					</div>
				{:else}
					<div
						class="bg-secondary text-foreground flex h-9 w-9 items-center justify-center rounded-full"
					>
						<svelte:component this={item.icon} size={20} />
					</div>
				{/if}
				<span class="text-foreground/90 font-medium">{item.label}</span>
			</a>
		{/each}

		<button
			class="flex w-full items-center space-x-3 rounded-lg p-2 transition-all hover:bg-black/5 dark:hover:bg-white/10"
		>
			<div class="bg-secondary flex h-9 w-9 items-center justify-center rounded-full">
				<ChevronDown size={20} />
			</div>
			<span class="text-foreground/90 font-medium">See more</span>
		</button>
	</nav>

	<!-- Separator -->
	<hr class="border-border/50 mx-2" />

	<!-- Shortcuts -->
	<div class="space-y-1">
		<div class="flex items-center justify-between px-2 py-1">
			<h3 class="text-muted-foreground text-sm font-semibold">Your shortcuts</h3>
			<button
				class="text-xs text-blue-500 opacity-0 transition-opacity hover:underline group-hover:opacity-100"
				>Edit</button
			>
		</div>
		{#each shortcuts as shortcut}
			<a
				href={`/communities/${shortcut.id}`}
				class="flex items-center space-x-3 rounded-lg p-2 transition-all hover:bg-black/5 dark:hover:bg-white/10"
			>
				<img src={shortcut.image} alt={shortcut.name} class="h-9 w-9 rounded-lg object-cover" />
				<span class="text-foreground/90 font-medium">{shortcut.name}</span>
			</a>
		{/each}
	</div>
</div>
