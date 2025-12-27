<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		notifications,
		setLoading,
		setError,
		markNotificationAsRead,
		setNotifications,
		appendNotifications
	} from '../../stores/notifications';
	import {
		fetchNotifications,
		getUnreadNotificationCount,
		markNotificationAsRead as apiMarkAsRead
	} from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import { formatDistanceToNow } from 'date-fns';
	import { Button } from '$lib/components/ui/button';
	import { MoreHorizontal } from '@lucide/svelte';

	let { mode = 'full', onClose } = $props<{ mode?: 'dropdown' | 'full'; onClose?: () => void }>();

	let currentPage = $state(1);
	let limit = 10;
	let totalNotifications = $state(0);
	let activeFilter = $state('all'); // 'all' | 'unread'
	let observer: IntersectionObserver;
	let loadMoreTrigger: HTMLElement;

	async function loadNotifications(reset = false) {
		if ($notifications.isLoading) return;

		setLoading(true);
		setError(null);
		try {
			if (!auth.state.user) {
				setError('User not authenticated.');
				return;
			}

			const pageToFetch = reset ? 1 : currentPage;
			const response = await fetchNotifications(
				pageToFetch,
				limit,
				activeFilter === 'unread' ? false : undefined
			);
			const unreadCountResponse = await getUnreadNotificationCount();

			if (reset) {
				setNotifications(response.notifications, unreadCountResponse.count);
				currentPage = 1;
			} else {
				appendNotifications(response.notifications); // Helper to append in store
			}
			totalNotifications = response.total;

			if (response.notifications.length > 0 && !reset) {
				currentPage++;
			} else if (reset && response.notifications.length > 0) {
				currentPage = 2; // Next page will be 2
			}
		} catch (err: any) {
			setError(err.message || 'Failed to load notifications.');
		} finally {
			setLoading(false);
		}
	}

	// We need to implement appendNotifications in the store or simulate it here.
	// Since I cannot edit the store file in this turn easily without seeing it, I'll assume setNotifications overwrites.
	// To support infinite scroll, I need to modify how I use the store or manually manage the list if the store doesn't support append.
	// The previous code used `$notifications.notifications`.
	// I will assume for now I need to fetch and set, but wait, `loadNotifications` usually overwrites.
	// I will modify the logic to strictly Append locally if the store doesn't support it,
	// OR just use `setNotifications` with a concatenated list if I had access to the current list.
	// Accessing `$notifications` value: `notifications.subscribe` ... or just `$notifications`.

	// Actually, to limit scope hazard, let's just create a `localNotifications` state array if the store is strict.
	// But `setNotifications` updates the specialized store.
	// Let's assumme I can't change the store right now.
	// I will handle the "Append" by getting the current list from `$notifications` and adding to it.

	async function loadMore() {
		if ($notifications.notifications.length >= totalNotifications) return;

		setLoading(true);
		try {
			const response = await fetchNotifications(
				currentPage,
				limit,
				activeFilter === 'unread' ? false : undefined
			);

			// Manual append logic if store is simple
			const currentList = $notifications.notifications;
			const newList = [...currentList, ...response.notifications];
			// Remove duplicates just in case
			const uniqueList = Array.from(new Map(newList.map((item) => [item.id, item])).values());

			const unreadCountResponse = await getUnreadNotificationCount();
			setNotifications(uniqueList, unreadCountResponse.count); // Updating store with full list

			totalNotifications = response.total;
			currentPage++;
		} catch (e) {
			console.error(e);
		} finally {
			setLoading(false);
		}
	}

	async function handleMarkAsRead(notificationId: string) {
		try {
			await apiMarkAsRead(notificationId);
			markNotificationAsRead(notificationId);
		} catch (err: any) {
			setError(err.message || 'Failed to mark notification as read.');
		}
	}

	function toggleFilter(filter: 'all' | 'unread') {
		activeFilter = filter;
		currentPage = 1;
		loadNotifications(true); // Reset
	}

	onMount(() => {
		loadNotifications(true);

		if (mode === 'dropdown' || mode === 'full') {
			observer = new IntersectionObserver(
				(entries) => {
					const first = entries[0];
					if (first.isIntersecting) {
						loadMore();
					}
				},
				{ threshold: 0.1 }
			);
		}
	});

	$effect(() => {
		if (loadMoreTrigger && observer) {
			observer.observe(loadMoreTrigger);
		}
		return () => {
			if (loadMoreTrigger && observer) observer.unobserve(loadMoreTrigger);
		};
	});

	onDestroy(() => {
		if (observer) observer.disconnect();
	});

	function getNotificationMessage(notification: any): string {
		// Use data if available for rich formatting
		if (notification.data?.sender_username) {
			const senderUsername = notification.data.sender_username;
			const targetType = notification.data.target_type || 'content';
			const reactionType = notification.data.reaction_type || 'reacted';
			const eventTitle = notification.data.event_title || 'an event';

			switch (notification.type) {
				case 'FRIEND_REQUEST':
					return `<span class="font-bold">${senderUsername}</span> sent you a friend request.`;
				case 'FRIEND_ACCEPT':
					return `<span class="font-bold">${senderUsername}</span> accepted your friend request.`;
				case 'LIKE':
					return `<span class="font-bold">${senderUsername}</span> ${reactionType.toLowerCase()} to your ${targetType}.`;
				case 'COMMENT':
					return `<span class="font-bold">${senderUsername}</span> commented on your ${targetType}.`;
				case 'REPLY':
					return `<span class="font-bold">${senderUsername}</span> replied to your ${targetType}.`;
				case 'MENTION':
					return `<span class="font-bold">${senderUsername}</span> mentioned you in a ${targetType}.`;
				case 'EVENT_INVITE':
					return `<span class="font-bold">${senderUsername}</span> invited you to <span class="font-bold">${eventTitle}</span>.`;
				case 'EVENT_REMINDER':
					return `Reminder: <span class="font-bold">${eventTitle}</span> is starting soon!`;
				case 'EVENT_INVITE_ACCEPTED':
					return `<span class="font-bold">${senderUsername}</span> accepted your invitation to <span class="font-bold">${eventTitle}</span>.`;
				case 'EVENT_INVITE_DECLINED':
					return `<span class="font-bold">${senderUsername}</span> declined your invitation to <span class="font-bold">${eventTitle}</span>.`;
			}
		}

		// Fallback to content field
		return notification.content || 'New notification';
	}
</script>

<div
	class="flex w-full flex-col {mode === 'dropdown'
		? 'h-[500px] max-h-[500px]'
		: 'bg-background min-h-screen'}"
>
	<!-- Header -->
	<div class="flex-shrink-0 px-4 pb-2 pt-4">
		<div class="mb-2 flex items-center justify-between">
			<h2 class="text-2xl font-bold">Notifications</h2>
			<Button variant="ghost" size="icon" class="rounded-full hover:bg-white/10">
				<MoreHorizontal size={20} />
			</Button>
		</div>
		<div class="flex gap-2">
			<button
				class="rounded-full px-3 py-1.5 text-sm font-semibold transition-colors {activeFilter ===
				'all'
					? 'bg-primary/20 text-primary'
					: 'text-muted-foreground hover:bg-white/5'}"
				onclick={() => toggleFilter('all')}
			>
				All
			</button>
			<button
				class="rounded-full px-3 py-1.5 text-sm font-semibold transition-colors {activeFilter ===
				'unread'
					? 'bg-primary/20 text-primary'
					: 'text-muted-foreground hover:bg-white/5'}"
				onclick={() => toggleFilter('unread')}
			>
				Unread
			</button>
		</div>
	</div>

	<!-- List -->
	<div class="flex-1 px-2 {mode === 'dropdown' ? 'overflow-y-auto' : ''}">
		{#if $notifications.isLoading && currentPage === 1}
			<div class="text-muted-foreground p-4 text-center">Loading...</div>
		{:else if $notifications.error}
			<div class="p-4 text-center text-red-500">{$notifications.error}</div>
		{:else if $notifications?.notifications?.length === 0}
			<div class="text-muted-foreground p-8 text-center">
				<p>No notifications.</p>
			</div>
		{:else}
			{#each $notifications.notifications as notification (notification.id)}
				<button
					class="hover:bg-secondary/50 group relative flex w-full items-start gap-3 rounded-lg p-2 text-left transition-colors {notification.read
						? ''
						: 'bg-blue-50/10'}"
					onclick={() => handleMarkAsRead(notification.id)}
				>
					<!-- Avatar -->
					<div class="relative flex-shrink-0">
						{#if notification.data?.sender_avatar}
							<img
								src={notification.data.sender_avatar}
								alt="Sender"
								class="h-14 w-14 rounded-full border border-white/10 object-cover"
							/>
						{:else}
							<div
								class="bg-secondary flex h-14 w-14 items-center justify-center rounded-full text-lg"
							>
								👤
							</div>
						{/if}
						<!-- Icon Badge -->
						<div
							class="ring-background absolute -bottom-1 -right-1 flex h-7 w-7 items-center justify-center rounded-full bg-blue-500 text-[14px] text-white shadow-sm ring-2"
						>
							{#if notification.type === 'LIKE'}👍{:else if notification.type === 'COMMENT'}💬{:else if notification.type === 'MENTION'}@{:else if notification.type === 'EVENT_INVITE'}📅{:else if notification.type === 'EVENT_REMINDER'}⏰{:else if notification.type === 'FRIEND_REQUEST'}👥{:else if notification.type === 'EVENT_INVITE_ACCEPTED'}✅{:else if notification.type === 'EVENT_INVITE_DECLINED'}❌{:else}🔔{/if}
						</div>
					</div>

					<!-- Content -->
					<div class="min-w-0 flex-1 py-1">
						<p class="text-foreground text-[15px] leading-snug">
							{@html getNotificationMessage(notification)}
						</p>
						<p class="text-primary mt-1 text-xs font-semibold">
							{formatDistanceToNow(new Date(notification.created_at), { addSuffix: true })}
						</p>
					</div>

					<!-- Unread Indicator -->
					{#if !notification.read}
						<div
							class="absolute right-2 top-1/2 h-3 w-3 -translate-y-1/2 rounded-full bg-blue-500 shadow-sm"
						></div>
					{/if}
				</button>
			{/each}

			<!-- Sentinel / Footer -->
			<div class="p-2" bind:this={loadMoreTrigger}>
				{#if $notifications.isLoading}
					<div class="text-muted-foreground text-center text-xs">Loading more...</div>
				{/if}
			</div>

			<!-- Bottom Action for Dropdown -->
			{#if mode === 'dropdown'}
				<div
					class="bg-background/95 border-border/10 sticky bottom-0 border-t pb-1 pt-2 backdrop-blur"
				>
					<Button
						href="/notifications"
						variant="ghost"
						class="text-primary w-full text-sm"
						onclick={() => onClose?.()}
					>
						See all notifications
					</Button>
				</div>
			{/if}
		{/if}
	</div>
</div>
