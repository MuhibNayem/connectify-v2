<script lang="ts">
	import { page } from '$app/stores';
	import type { Community } from '$lib/types/community';
	import {
		Users,
		Lock,
		Globe,
		Settings,
		Image,
		FileText,
		UserPlus,
		LogOut,
		CheckCircle,
		ShieldCheck
	} from '@lucide/svelte';

	export let community: Community;
	export let isMember = false;
	export let isAdmin = false;
	export let isPending = false;

	// Handlers for join/leave/invite would be passed down or handled via dispatch/store
	import { createEventDispatcher } from 'svelte';
	const dispatch = createEventDispatcher();

	function handleJoin() {
		dispatch('join');
	}

	function handleLeave() {
		dispatch('leave');
	}

	function handleInvite() {
		dispatch('invite');
	}

	$: activeTab = $page.url.pathname.split('/').pop() || 'discussion';
	// If path is just /communities/[id], activeTab is id, but effectively 'discussion'
	$: if (activeTab === community.id) activeTab = 'discussion';
</script>

<div class="relative mb-6 w-full">
	<!-- Cover Image -->
	<div class="group relative h-[350px] w-full overflow-hidden rounded-b-3xl">
		{#if community.cover_image}
			<img
				src={community.cover_image}
				alt="Cover"
				class="h-full w-full object-cover transition-transform duration-700 group-hover:scale-105"
			/>
		{:else}
			<div class="h-full w-full bg-gradient-to-br from-blue-100 to-purple-100"></div>
		{/if}
		<div class="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent"></div>
	</div>

	<div class="relative mx-auto max-w-7xl px-4 pb-4 sm:px-6 lg:px-8">
		<div class="flex flex-col items-end gap-6 md:flex-row md:items-end">
			<!-- Avatar -->
			<div class="relative -mt-[84px] shrink-0">
				<div class="h-[168px] w-[168px] rounded-full border-4 border-white bg-white p-1 shadow-lg">
					<img
						src={community.avatar ||
							`https://ui-avatars.com/api/?name=${community.name}&background=random`}
						alt={community.name}
						class="h-full w-full rounded-full object-cover"
					/>
				</div>
			</div>

			<!-- Info -->
			<div class="mb-2 flex-1 pt-4 md:mb-0 md:pt-0">
				<h1 class="mb-2 text-4xl font-bold tracking-tight text-gray-900">{community.name}</h1>
				<div class="flex items-center gap-4 text-sm font-medium text-gray-600">
					{#if community.privacy === 'public'}
						<div class="flex items-center gap-1.5 rounded-full bg-gray-100 px-3 py-1 text-gray-700">
							<Globe size={14} />
							<span>Public Group</span>
						</div>
					{:else}
						<div class="flex items-center gap-1.5 rounded-full bg-gray-100 px-3 py-1 text-gray-700">
							<Lock size={14} />
							<span>Private Group</span>
						</div>
					{/if}
					<span>•</span>
					<span class="font-semibold">{community.stats.member_count} members</span>
					{#if community.stats.post_count}
						<span>•</span>
						<span>{community.stats.post_count} posts</span>
					{/if}
				</div>
			</div>

			<!-- Actions -->
			<div class="mb-4 flex items-center gap-3 md:mb-2">
				{#if isMember}
					<button
						class="flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-2 text-gray-700 transition-colors hover:bg-gray-50"
						on:click={handleInvite}
					>
						<UserPlus size={18} />
						<span>Invite</span>
					</button>
					<button
						class="group flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-2 text-gray-700 transition-colors hover:border-red-200 hover:bg-red-50 hover:text-red-600"
						on:click={handleLeave}
					>
						<LogOut size={18} class="group-hover:text-red-600" />
						<span class="group-hover:text-red-600">Joined</span>
					</button>
				{:else if isPending}
					<button
						class="flex cursor-not-allowed items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 px-4 py-2 text-gray-500"
					>
						<CheckCircle size={18} />
						<span>Pending</span>
					</button>
				{:else}
					<button
						class="flex items-center gap-2 rounded-lg bg-blue-600 px-6 py-2.5 font-semibold text-white shadow-lg shadow-blue-900/20 hover:bg-blue-700"
						on:click={handleJoin}
					>
						<UserPlus size={18} />
						<span>Join Group</span>
					</button>
				{/if}
			</div>
		</div>

		<!-- Navigation Tabs -->
		<div class="mt-8 border-t border-gray-200 pt-1">
			<nav class="scrollbar-hide flex gap-1 overflow-x-auto pb-2">
				<a
					href="/communities/{community.id}"
					class="flex items-center gap-2 rounded-lg px-4 py-3 text-sm font-medium transition-all
					{activeTab === 'discussion'
						? 'bg-blue-50 text-blue-600'
						: 'text-gray-500 hover:bg-gray-100 hover:text-gray-900'}"
				>
					<FileText size={18} />
					Discussion
				</a>
				<a
					href="/communities/{community.id}/members"
					class="flex items-center gap-2 rounded-lg px-4 py-3 text-sm font-medium transition-all
					{activeTab === 'members'
						? 'bg-blue-50 text-blue-600'
						: 'text-gray-500 hover:bg-gray-100 hover:text-gray-900'}"
				>
					<Users size={18} />
					Members
				</a>
				<a
					href="/communities/{community.id}/media"
					class="flex items-center gap-2 rounded-lg px-4 py-3 text-sm font-medium transition-all
					{activeTab === 'media'
						? 'bg-blue-50 text-blue-600'
						: 'text-gray-500 hover:bg-gray-100 hover:text-gray-900'}"
				>
					<Image size={18} />
					Media
				</a>
				<a
					href="/communities/{community.id}/about"
					class="flex items-center gap-2 rounded-lg px-4 py-3 text-sm font-medium transition-all
					{activeTab === 'about'
						? 'bg-blue-50 text-blue-600'
						: 'text-gray-500 hover:bg-gray-100 hover:text-gray-900'}"
				>
					<ShieldCheck size={18} />
					About
				</a>
				{#if isAdmin}
					<a
						href="/communities/{community.id}/admin"
						class="ml-auto flex items-center gap-2 rounded-lg px-4 py-3 text-sm font-medium transition-all
						{activeTab === 'admin'
							? 'bg-blue-50 text-blue-600'
							: 'text-gray-500 hover:bg-gray-100 hover:text-gray-900'}"
					>
						<Settings size={18} />
						Manage
					</a>
				{/if}
			</nav>
		</div>
	</div>
</div>

<style>
	/* Custom scrollbar hiding if needed */
	.scrollbar-hide::-webkit-scrollbar {
		display: none;
	}
	.scrollbar-hide {
		-ms-overflow-style: none;
		scrollbar-width: none;
	}
</style>
