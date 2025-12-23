<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import SearchInput from '$lib/components/ui/search/SearchInput.svelte';
	import NotificationList from '../notifications/NotificationList.svelte';
	import { notifications } from '../../stores/notifications';
	import { auth } from '$lib/stores/auth.svelte';
	import { onMount } from 'svelte';

	import { page } from '$app/stores';
	import {
		Search,
		Home,
		Users,
		MonitorPlay,
		Store,
		UsersRound,
		Grid,
		MessageCircle,
		Bell,
		LogOut,
		Menu
	} from '@lucide/svelte';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';

	let showNotifications = $state(false);
	let notificationButton: HTMLElement;
	let notificationList: HTMLElement;
	let mobileMenuOpen = $state(false);

	function toggleMobileMenu() {
		mobileMenuOpen = !mobileMenuOpen;
	}

	function handleSearchSubmit(event: CustomEvent<string>) {
		const query = event.detail;
		if (query) {
			goto(`/search?query=${encodeURIComponent(query)}`);
		}
	}

	function toggleNotifications() {
		showNotifications = !showNotifications;
	}

	async function handleLogout() {
		await auth.logout();
		goto('/');
	}

	function handleClickOutside(event: MouseEvent) {
		if (
			showNotifications &&
			!notificationButton.contains(event.target as Node) &&
			!notificationList.contains(event.target as Node)
		) {
			showNotifications = false;
		}
	}

	onMount(() => {
		window.addEventListener('click', handleClickOutside);
		return () => {
			window.removeEventListener('click', handleClickOutside);
		};
	});
</script>

<header
	class="glass-panel fixed left-0 right-0 top-0 z-50 flex h-14 items-center justify-between border-b-0 px-4"
>
	<!-- Left: Logo & Search -->
	<div class="flex w-auto items-center space-x-2 md:w-[260px] lg:w-[300px]">
		<a href="/dashboard" class="flex-shrink-0">
			<div
				class="h-10 w-10 overflow-hidden rounded-full shadow-lg transition-transform hover:scale-105"
			>
				<img src="/logo.png" alt="Connectify Logo" class="h-full w-full object-cover" />
			</div>
		</a>
		<div class="relative hidden lg:block">
			<SearchInput
				on:search={handleSearchSubmit}
				class="glass-input h-10 w-[240px] rounded-full bg-black/5 px-4 pl-10 focus:bg-white/10"
				placeholder="Search Connectify"
			/>
		</div>
		<Button
			variant="ghost"
			size="icon"
			class="rounded-full bg-black/5 hover:bg-black/10 lg:hidden dark:bg-white/10 dark:hover:bg-white/20"
		>
			<Search size={22} class="text-foreground" />
		</Button>
	</div>

	<!-- Center: Navigation Tabs -->
	<nav class="hidden max-w-xl flex-1 justify-center space-x-1 md:flex">
		<a
			href="/dashboard"
			class="group relative flex h-12 w-28 items-center justify-center rounded-lg hover:bg-black/5 dark:hover:bg-white/10 {$page
				.url.pathname === '/dashboard'
				? 'text-primary'
				: 'text-muted-foreground'}"
		>
			<Home
				size={26}
				class="transition-transform group-hover:scale-110 {$page.url.pathname === '/dashboard'
					? 'fill-current'
					: ''}"
			/>
			{#if $page.url.pathname === '/dashboard'}
				<span class="bg-primary absolute bottom-0 h-1 w-full rounded-t-md"></span>
			{/if}
		</a>
		<a
			href="/friends"
			class="group relative flex h-12 w-28 items-center justify-center rounded-lg hover:bg-black/5 dark:hover:bg-white/10 {$page
				.url.pathname === '/friends'
				? 'text-primary'
				: 'text-muted-foreground'}"
		>
			<Users
				size={26}
				class="transition-transform group-hover:scale-110 {$page.url.pathname === '/friends'
					? 'fill-current'
					: ''}"
			/>
			{#if $page.url.pathname === '/friends'}
				<span class="bg-primary absolute bottom-0 h-1 w-full rounded-t-md"></span>
			{/if}
		</a>
		<a
			href="/video"
			class="group relative flex h-12 w-28 items-center justify-center rounded-lg hover:bg-black/5 dark:hover:bg-white/10 {$page
				.url.pathname === '/video'
				? 'text-primary'
				: 'text-muted-foreground'}"
		>
			<MonitorPlay
				size={26}
				class="transition-transform group-hover:scale-110 {$page.url.pathname === '/video'
					? 'fill-current'
					: ''}"
			/>
			{#if $page.url.pathname === '/video'}
				<span class="bg-primary absolute bottom-0 h-1 w-full rounded-t-md"></span>
			{/if}
		</a>
		<a
			href="/marketplace"
			class="group relative flex h-12 w-28 items-center justify-center rounded-lg hover:bg-black/5 dark:hover:bg-white/10 {$page
				.url.pathname === '/marketplace'
				? 'text-primary'
				: 'text-muted-foreground'}"
		>
			<Store
				size={26}
				class="transition-transform group-hover:scale-110 {$page.url.pathname === '/marketplace'
					? 'fill-current'
					: ''}"
			/>
			{#if $page.url.pathname === '/marketplace'}
				<span class="bg-primary absolute bottom-0 h-1 w-full rounded-t-md"></span>
			{/if}
		</a>
		<a
			href="/communities"
			class="group relative flex h-12 w-28 items-center justify-center rounded-lg hover:bg-black/5 dark:hover:bg-white/10 {$page
				.url.pathname === '/communities'
				? 'text-primary'
				: 'text-muted-foreground'}"
		>
			<UsersRound
				size={26}
				class="transition-transform group-hover:scale-110 {$page.url.pathname === '/communities'
					? 'fill-current'
					: ''}"
			/>
			{#if $page.url.pathname === '/communities'}
				<span class="bg-primary absolute bottom-0 h-1 w-full rounded-t-md"></span>
			{/if}
		</a>
	</nav>

	<!-- Right: Utility Icons -->
	<div class="flex w-auto items-center justify-end space-x-2 md:w-[260px] lg:w-[300px]">
		<!-- Mobile Menu Button -->
		<Button
			variant="ghost"
			size="icon"
			class="h-10 w-10 rounded-full bg-black/5 hover:bg-black/10 md:hidden dark:bg-white/10 dark:hover:bg-white/20"
			onclick={toggleMobileMenu}
		>
			<Menu size={20} class="text-foreground" />
		</Button>

		<!-- Menu Grid (Desktop only) -->
		<Button
			variant="ghost"
			size="icon"
			class="hidden h-10 w-10 rounded-full bg-black/5 hover:bg-black/10 md:flex dark:bg-white/10 dark:hover:bg-white/20"
		>
			<Grid size={20} class="text-foreground" />
		</Button>

		<!-- Messenger -->
		<Button
			variant="ghost"
			size="icon"
			class="hidden h-10 w-10 rounded-full bg-black/5 hover:bg-black/10 sm:flex dark:bg-white/10 dark:hover:bg-white/20"
			onclick={() => goto('/messages')}
		>
			<MessageCircle size={20} class="text-foreground" />
		</Button>

		<!-- Notifications -->
		<div class="relative">
			<div bind:this={notificationButton}>
				<Button
					variant="ghost"
					size="icon"
					class="h-10 w-10 rounded-full bg-black/5 hover:bg-black/10 dark:bg-white/10 dark:hover:bg-white/20"
					onclick={toggleNotifications}
				>
					<Bell size={20} class="text-foreground" />
					{#if $notifications.unreadCount > 0}
						<span
							class="ring-background absolute -right-1 -top-1 inline-flex h-5 w-5 items-center justify-center rounded-full bg-red-500 text-xs font-bold text-white shadow-sm ring-2"
						>
							{$notifications.unreadCount}
						</span>
					{/if}
				</Button>
			</div>
			{#if showNotifications}
				<div
					bind:this={notificationList}
					class="border-border/40 bg-background absolute right-0 z-50 mt-2 w-96 overflow-hidden rounded-xl border shadow-2xl"
				>
					<NotificationList mode="dropdown" onClose={() => (showNotifications = false)} />
				</div>
			{/if}
		</div>

		<!-- Profile Dropdown Trigger (Simply Avatar for now) -->
		<div class="relative">
			<Button
				variant="ghost"
				class="hover:ring-primary/20 h-10 w-10 overflow-hidden rounded-full p-0 ring-2 ring-transparent"
				onclick={() => (mobileMenuOpen = !mobileMenuOpen)}
			>
				<Avatar class="h-10 w-10">
					<AvatarImage src={auth.state.user?.avatar} />
					<AvatarFallback>{auth.state.user?.username?.charAt(0).toUpperCase()}</AvatarFallback>
				</Avatar>
			</Button>

			<!-- Dropdown Menu (Reusing logic for desktop profile menu if needed, simplified here as user menu) -->
			{#if mobileMenuOpen}
				<div
					class="bg-popover border-border/40 absolute right-0 z-50 mt-2 flex w-60 flex-col space-y-1 rounded-xl border p-2 shadow-xl"
				>
					<a
						href={`/profile/${auth.state.user?.id}`}
						class="flex items-center space-x-3 rounded-lg p-2 hover:bg-white/10"
					>
						<Avatar class="h-9 w-9">
							<AvatarImage src={auth.state.user?.avatar} />
							<AvatarFallback>{auth.state.user?.username?.charAt(0).toUpperCase()}</AvatarFallback>
						</Avatar>
						<span class="font-semibold"
							>{auth.state.user?.full_name || auth.state.user?.username}</span
						>
					</a>
					<hr class="my-1 border-white/10" />
					<button
						class="flex w-full items-center space-x-2 rounded-lg p-2 text-left hover:bg-white/10"
						onclick={handleLogout}
					>
						<LogOut size={20} />
						<span>Log Out</span>
					</button>
				</div>
			{/if}
		</div>
	</div>

	{#if mobileMenuOpen}
		<div
			class="glass-panel fixed bottom-0 left-0 right-0 top-14 z-40 overflow-y-auto p-4 md:hidden"
		>
			<nav class="space-y-6">
				<!-- Main Nav Grid -->
				<div>
					<h3 class="text-muted-foreground mb-2 px-2 text-xs font-semibold uppercase">Menu</h3>
					<div class="grid grid-cols-2 gap-3">
						<a
							href="/dashboard"
							class="flex items-center space-x-3 rounded-xl bg-white/5 p-3 transition-all active:scale-95"
							onclick={() => (mobileMenuOpen = false)}
						>
							<Home size={20} class="text-primary" />
							<span class="font-medium">Home</span>
						</a>
						<a
							href="/friends"
							class="flex items-center space-x-3 rounded-xl bg-white/5 p-3 transition-all active:scale-95"
							onclick={() => (mobileMenuOpen = false)}
						>
							<Users size={20} class="text-blue-500" />
							<span class="font-medium">Friends</span>
						</a>
						<a
							href="/messages"
							class="flex items-center space-x-3 rounded-xl bg-white/5 p-3 transition-all active:scale-95"
							onclick={() => (mobileMenuOpen = false)}
						>
							<MessageCircle size={20} class="text-green-500" />
							<span class="font-medium">Messages</span>
						</a>
						<a
							href="/video"
							class="flex items-center space-x-3 rounded-xl bg-white/5 p-3 transition-all active:scale-95"
							onclick={() => (mobileMenuOpen = false)}
						>
							<MonitorPlay size={20} class="text-blue-400" />
							<span class="font-medium">Video</span>
						</a>
						<a
							href="/marketplace"
							class="flex items-center space-x-3 rounded-xl bg-white/5 p-3 transition-all active:scale-95"
							onclick={() => (mobileMenuOpen = false)}
						>
							<Store size={20} class="text-blue-500" />
							<span class="font-medium">Marketplace</span>
						</a>
						<a
							href="/communities"
							class="flex items-center space-x-3 rounded-xl bg-white/5 p-3 transition-all active:scale-95"
							onclick={() => (mobileMenuOpen = false)}
						>
							<UsersRound size={20} class="text-blue-500" />
							<span class="font-medium">Groups</span>
						</a>
					</div>
				</div>

				<!-- Account Section -->
				<div>
					<h3 class="text-muted-foreground mb-2 px-2 text-xs font-semibold uppercase">Account</h3>
					<div class="space-y-2">
						<a
							href="/profile/{auth.state.user?.id}"
							class="flex items-center space-x-3 rounded-xl bg-white/5 p-3 transition-all active:scale-95"
							onclick={() => (mobileMenuOpen = false)}
						>
							<Avatar class="h-8 w-8">
								<AvatarImage src={auth.state.user?.avatar} />
								<AvatarFallback>{auth.state.user?.username?.charAt(0).toUpperCase()}</AvatarFallback
								>
							</Avatar>
							<span class="font-medium">Profile</span>
						</a>
						<Button
							variant="ghost"
							class="w-full justify-start space-x-3 rounded-xl bg-white/5 p-6 hover:bg-red-500/10 hover:text-red-500"
							onclick={handleLogout}
						>
							<LogOut size={20} />
							<span>Log Out</span>
						</Button>
					</div>
				</div>
			</nav>
		</div>
	{/if}
</header>
