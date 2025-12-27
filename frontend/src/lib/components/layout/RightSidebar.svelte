<script lang="ts">
	import { getFriendships } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import Skeleton from '$lib/components/ui/skeleton/Skeleton.svelte';
	import { Gift, ExternalLink } from '@lucide/svelte';

	let currentUser = $derived(auth.state.user);
	let contacts = $state<any[]>([]);
	let loading = $state(true);

	// Mock data
	let birthdays = [{ id: 1, name: 'Alex Johnson', date: 'Today' }];

	let suggestedGroups = [
		{
			id: 1,
			name: 'Svelte Lovers',
			members: '12k members',
			image: 'https://images.unsplash.com/photo-1555099962-4199c345e5dd?w=200&h=200&fit=crop'
		},
		{
			id: 2,
			name: 'Nature Photography',
			members: '8.5k members',
			image: 'https://images.unsplash.com/photo-1470071459604-3b5ec3a7fe05?w=200&h=200&fit=crop'
		}
	];

	async function fetchContacts() {
		try {
			// Fetch accepted friends for the "Contacts" list
			const response = await getFriendships('accepted', 1, 7);
			contacts = response.data || [];
		} catch (err) {
			console.error('Failed to load contacts:', err);
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (currentUser) {
			fetchContacts();
		}
	});
</script>

<div class="space-y-6">
	<!-- Sponsored -->
	<div>
		<h3 class="text-muted-foreground px-2 text-xs font-semibold uppercase">Sponsored</h3>
		<div class="mt-3 grid gap-4">
			<div
				class="group flex cursor-pointer items-center space-x-3 rounded-xl p-2 transition-all hover:bg-white/5"
			>
				<div class="aspect-square h-24 w-24 overflow-hidden rounded-lg bg-white/5">
					<img
						src="https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=400&q=80"
						alt="Product"
						class="h-full w-full object-cover transition-transform duration-500 group-hover:scale-110"
					/>
				</div>
				<div class="flex-1 space-y-1">
					<p class="text-foreground font-semibold">Premium Sneakers</p>
					<p class="text-muted-foreground text-xs">stepup.com</p>
				</div>
			</div>
		</div>
	</div>

	<hr class="border-white/10" />

	<!-- Birthdays -->
	<div>
		<h3 class="text-muted-foreground mb-3 px-2 text-xs font-semibold uppercase">Birthdays</h3>
		{#each birthdays as birthday}
			<div
				class="flex cursor-pointer items-center space-x-3 rounded-xl p-2 transition-all hover:bg-white/5"
			>
				<Gift class="text-primary h-8 w-8" />
				<div>
					<p class="text-foreground text-sm font-medium">
						<span class="font-bold">{birthday.name}'s</span> birthday is {birthday.date}
					</p>
				</div>
			</div>
		{/each}
	</div>

	<hr class="border-white/10" />

	<!-- Contacts -->
	<div>
		<div class="mb-2 flex items-center justify-between px-2">
			<h3 class="text-muted-foreground text-xs font-semibold uppercase">Contacts</h3>
		</div>

		{#if loading}
			<div class="space-y-3 px-2">
				{#each Array(3) as _}
					<div class="flex items-center space-x-3">
						<Skeleton class="bg-primary/10 h-8 w-8 rounded-full" />
						<Skeleton class="bg-primary/10 h-4 w-24" />
					</div>
				{/each}
			</div>
		{:else if contacts.length === 0}
			<p class="text-muted-foreground px-2 text-sm">No online contacts.</p>
		{:else}
			<div class="space-y-1">
				{#each contacts as contact (contact.id)}
					{@const friend =
						contact.receiver_id === currentUser?.id
							? contact.requester_info
							: contact.receiver_info}
					<a
						href={`/profile/${friend.id}`}
						class="hover:bg-primary/10 flex items-center space-x-3 rounded-xl p-2 transition-all"
					>
						<div class="relative">
							<Avatar class="h-8 w-8">
								<AvatarImage src={friend.avatar} alt={friend.username} />
								<AvatarFallback>{friend.username.charAt(0).toUpperCase()}</AvatarFallback>
							</Avatar>
							<!-- Simple green dot for now, presence logic can be expanded -->
							<span
								class="border-background absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border-2 bg-green-500"
							></span>
						</div>
						<span class="text-foreground text-sm font-medium"
							>{friend.full_name || friend.username}</span
						>
					</a>
				{/each}
			</div>
		{/if}
	</div>

	<hr class="border-white/10" />

	<!-- Suggested Pages/Groups -->
	<div>
		<h3 class="text-muted-foreground mb-3 px-2 text-xs font-semibold uppercase">
			Suggested for you
		</h3>
		<div class="space-y-3">
			{#each suggestedGroups as group}
				<div
					class="flex cursor-pointer items-center justify-between rounded-xl p-2 transition-all hover:bg-white/5"
				>
					<div class="flex items-center space-x-3">
						<Avatar class="h-9 w-9 rounded-lg">
							<AvatarImage src={group.image} />
							<AvatarFallback>{group.name[0]}</AvatarFallback>
						</Avatar>
						<div>
							<p class="text-foreground text-sm font-semibold">{group.name}</p>
							<p class="text-muted-foreground text-xs">{group.members}</p>
						</div>
					</div>
					<button class="text-muted-foreground hover:text-primary transition-colors">
						<ExternalLink size={16} />
					</button>
				</div>
			{/each}
		</div>
	</div>
</div>
