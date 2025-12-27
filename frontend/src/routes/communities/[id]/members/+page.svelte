<script lang="ts">
	import { getContext } from 'svelte';
	import type { Writable } from 'svelte/store';
	import { getCommunityMembers, getCommunityAdmins, type Community } from '$lib/api';
	import type { User } from '$lib/types';
	import { Search, Settings } from '@lucide/svelte';

	const communityStore = getContext<Writable<Community>>('community');
	let community = $derived($communityStore);
	let members: User[] = $state([]);
	let loading = $state(true);
	let activeTab = $state('all'); // all, admins

	async function loadMembers() {
		loading = true;
		try {
			if (activeTab === 'admins') {
				const res = await getCommunityAdmins(community.id);
				members = res || [];
			} else {
				const res = await getCommunityMembers(community.id);
				members = res.users || [];
			}
		} catch (e) {
			console.error(e);
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (activeTab && community) {
			loadMembers();
		}
	});
</script>

<div class="min-h-[500px] overflow-hidden rounded-2xl bg-white shadow-sm ring-1 ring-gray-200">
	<!-- Header / Tabs -->
	<div class="border-b border-gray-100 p-4">
		<div class="mb-4 flex items-center justify-between">
			<h2 class="text-xl font-bold text-gray-900">
				Members · <span class="text-gray-500">{community?.stats.member_count}</span>
			</h2>
			<div class="relative w-64">
				<Search class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={16} />
				<input
					type="text"
					placeholder="Find a member"
					class="w-full rounded-full border border-gray-200 bg-gray-50 py-2 pl-9 pr-4 text-sm text-gray-900 transition-colors focus:border-blue-500 focus:outline-none"
				/>
			</div>
		</div>

		<div class="flex gap-1">
			<button
				class="rounded-lg px-4 py-2 text-sm font-medium transition-colors {activeTab === 'all'
					? 'bg-gray-100 text-gray-900'
					: 'text-gray-500 hover:bg-gray-50'}"
				on:click={() => (activeTab = 'all')}
			>
				All Members
			</button>
			<button
				class="rounded-lg px-4 py-2 text-sm font-medium transition-colors {activeTab === 'admins'
					? 'bg-gray-100 text-gray-900'
					: 'text-gray-500 hover:bg-gray-50'}"
				on:click={() => (activeTab = 'admins')}
			>
				Admins
			</button>
		</div>
	</div>

	<!-- List -->
	<div class="grid grid-cols-1 gap-4 p-4 md:grid-cols-2">
		{#if loading}
			{#each Array(6) as _}
				<div
					class="flex animate-pulse items-center gap-4 rounded-xl border border-gray-100 bg-white p-4"
				>
					<div class="h-12 w-12 rounded-full bg-gray-200"></div>
					<div class="flex-1 space-y-2">
						<div class="h-4 w-1/2 rounded bg-gray-200"></div>
						<div class="h-3 w-1/3 rounded bg-gray-100"></div>
					</div>
				</div>
			{/each}
		{:else if members.length > 0}
			{#each members as member}
				<div
					class="group flex items-center gap-4 rounded-xl border border-gray-100 bg-white p-4 transition-colors hover:border-gray-200 hover:bg-gray-50"
				>
					<img
						src={member.avatar ||
							`https://ui-avatars.com/api/?name=${member.full_name || member.username}`}
						alt={member.username}
						class="h-14 w-14 rounded-full object-cover"
					/>
					<div class="min-w-0 flex-1">
						<h3 class="truncate font-semibold text-gray-900">
							{member.full_name || member.username}
						</h3>
						<p class="truncate text-xs text-gray-500">@{member.username}</p>
						<!-- Add Joined date if available -->
					</div>

					<!-- Actions (if admin viewing) -->
					{#if community.is_admin && member.id !== community.creator_id}
						<!-- Simplistic admin actions -->
						<button
							class="rounded-lg p-2 text-gray-400 opacity-0 transition-opacity hover:bg-gray-200 hover:text-gray-700 group-hover:opacity-100"
						>
							<Settings size={18} />
						</button>
					{/if}
				</div>
			{/each}
		{:else}
			<div class="col-span-full py-12 text-center text-gray-500">No members found.</div>
		{/if}
	</div>
</div>
