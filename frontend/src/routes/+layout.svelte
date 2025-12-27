<script>
	import '../app.css';
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth.svelte.js';
	import Toast from '$lib/components/ui/toast/Toast.svelte';
	import CallContainer from '$lib/components/messages/CallContainer.svelte';
	import AppHeader from '$lib/components/layout/AppHeader.svelte';
	import { page } from '$app/stores';
	import { getUnreadNotificationCount } from '$lib/api';
	import { setUnreadCount } from '$lib/stores/notifications';
	import { connectWebSocket, disconnectWebSocket } from '$lib/websocket';

onMount(() => {
	auth.initialize();
	if (!auth.state.accessToken) {
		auth.refresh().catch(() => {});
	}
});

	// Reactive effect to manage WebSocket and notifications
	$effect(() => {
		if (auth.state.user) {
			connectWebSocket();
			(async () => {
				try {
					const response = await getUnreadNotificationCount();
					setUnreadCount(response.count);
				} catch (error) {
					console.error('Failed to fetch initial unread notification count:', error);
				}
			})();
		} else {
			disconnectWebSocket();
		}
	});

	// Determine if we should show the AppHeader
	// Hiding on public routes: login (root?), register, forgot-password, and potentially landing page if different
	let showHeader = $derived(
		!!auth.state.user &&
			!['/login', '/register', '/forgot-password'].includes($page.url.pathname) &&
			$page.url.pathname !== '/'
	);
</script>

<div class="text-foreground selection:bg-primary/30 relative min-h-screen antialiased">
	<div class="bg-background/50 fixed inset-0 -z-10 backdrop-blur-[1px]"></div>

	{#if showHeader}
		<AppHeader />
	{/if}

	<Toast />
	<CallContainer />

	<main class:pt-16={showHeader}>
		<slot />
	</main>
</div>
