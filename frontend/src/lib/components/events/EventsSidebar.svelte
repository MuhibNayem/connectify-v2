<script lang="ts">
	import { onMount } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import {
		Search,
		Compass,
		Calendar,
		MapPin,
		Gift,
		Bell,
		Plus,
		Settings,
		User
	} from '@lucide/svelte';
	import { page } from '$app/stores';
	import { getEventInvitations } from '$lib/api';

	let activePath = $derived($page.url.pathname);
	let invitationCount = $state(0);

	// Fetch pending invitations count
	onMount(async () => {
		try {
			const res = await getEventInvitations(1, 1); // Just get count
			invitationCount = res.total || 0;
		} catch (err) {
			console.error('Failed to load invitation count:', err);
		}
	});

	const links = [
		{ label: 'Home', icon: Compass, href: '/events' },
		{ label: 'Your Events', icon: User, href: '/events/your-events' },
		{ label: 'Local', icon: MapPin, href: '/events/local' },
		{ label: 'Birthdays', icon: Gift, href: '/events/birthdays' },
		{ label: 'Notifications', icon: Bell, href: '/events/notifications', showBadge: true }
	];

	const categories = ['Music', 'Nightlife', 'Arts', 'Food', 'Technology', 'Sports', 'Wellness'];
</script>

<div
	class="bg-background hidden h-full w-80 flex-shrink-0 flex-col border-r border-white/5 p-4 md:flex"
>
	<h1 class="mb-6 text-2xl font-bold">Events</h1>

	<!-- Search -->
	<div class="relative mb-6">
		<Search class="text-muted-foreground absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2" />
		<Input placeholder="Search events..." class="bg-secondary/50 border-none pl-9" />
	</div>

	<!-- Main Nav -->
	<nav class="space-y-1">
		{#each links as link}
			<a href={link.href} class="block">
				<Button
					variant="ghost"
					class="text-muted-foreground hover:bg-secondary/20 w-full justify-start gap-3 {activePath ===
					link.href
						? 'bg-secondary/20 text-primary font-bold'
						: ''}"
				>
					<link.icon size={20} />
					{link.label}
					{#if link.showBadge && invitationCount > 0}
						<span
							class="ml-auto flex h-5 min-w-5 items-center justify-center rounded-full bg-red-500 px-1.5 text-xs font-bold text-white"
						>
							{invitationCount > 99 ? '99+' : invitationCount}
						</span>
					{/if}
				</Button>
			</a>
		{/each}
	</nav>

	<div class="my-4 border-t border-white/5"></div>

	<!-- Categories -->
	<div class="flex-1 overflow-y-auto">
		<h3 class="text-muted-foreground mb-2 px-4 text-xs font-bold uppercase tracking-wider">
			Categories
		</h3>
		<div class="space-y-1">
			{#each categories as cat}
				<Button
					variant="ghost"
					size="sm"
					class="text-muted-foreground hover:text-foreground w-full justify-start px-4"
				>
					{cat}
				</Button>
			{/each}
		</div>
	</div>

	<!-- Footer Actions -->
	<div class="mt-4 flex flex-col gap-2">
		<a href="/events/create" class="w-full">
			<Button class="bg-primary text-primary-foreground w-full gap-2 font-semibold">
				<Plus size={18} /> Create New Event
			</Button>
		</a>
		<Button variant="ghost" class="text-muted-foreground w-full justify-start gap-3">
			<Settings size={20} /> Settings
		</Button>
	</div>
</div>
