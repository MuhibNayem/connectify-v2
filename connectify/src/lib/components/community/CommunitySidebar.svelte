<script lang="ts">
	import type { Community } from '$lib/types/community';
	import { Shield, Info, FileText, ChevronRight, Settings, Users } from '@lucide/svelte';

	export let community: Community;
	export let isAdmin = false;
</script>

<div class="flex flex-col gap-6">
	<!-- About Card -->
	<div class="rounded-2xl bg-white p-5 shadow-sm ring-1 ring-gray-200">
		<h3 class="mb-4 text-lg font-semibold text-gray-900">About</h3>
		<p class="mb-4 text-sm leading-relaxed text-gray-600">
			{community.description}
		</p>
		<div class="flex flex-col gap-3">
			<div class="flex items-center gap-3 text-sm text-gray-500">
				<Shield size={16} />
				<span>
					{community.privacy === 'public' ? 'Public' : 'Private'} Group
					<span class="mt-0.5 block text-xs text-gray-400">
						{community.privacy === 'public'
							? "Anyone can see who's in the group and what they post."
							: "Only members can see who's in the group and what they post."}
					</span>
				</span>
			</div>
			<!-- Add location or created date if available -->
			<div class="flex items-center gap-3 text-sm text-gray-500">
				<Info size={16} />
				<span>Created {new Date(community.created_at).toLocaleDateString()}</span>
			</div>
		</div>
	</div>

	<!-- Admin Tools (Only for Admin) -->
	{#if isAdmin}
		<div class="rounded-2xl border-l-4 border-blue-500 bg-white p-5 shadow-sm ring-1 ring-gray-200">
			<div class="mb-4 flex items-center justify-between">
				<h3 class="text-lg font-semibold text-gray-900">Admin Tools</h3>
				<Shield size={18} class="text-blue-500" />
			</div>
			<div class="flex flex-col gap-2">
				<a
					href="/communities/{community.id}/admin?tab=settings"
					class="group flex items-center justify-between rounded-xl p-3 transition-colors hover:bg-gray-50"
				>
					<div class="flex items-center gap-3 text-gray-600 group-hover:text-gray-900">
						<Settings size={18} />
						<span class="text-sm font-medium">Settings</span>
					</div>
					<ChevronRight size={16} class="text-gray-400 group-hover:text-gray-600" />
				</a>

				<a
					href="/communities/{community.id}/admin?tab=member-requests"
					class="group flex items-center justify-between rounded-xl p-3 transition-colors hover:bg-gray-50"
				>
					<div class="flex items-center gap-3 text-gray-600 group-hover:text-gray-900">
						<Users size={18} />
						<!-- Changed from Shield to Users for clarity -->
						<span class="text-sm font-medium">Member Requests</span>
					</div>
					<div class="flex items-center gap-2">
						{#if (community.stats?.pending_count || 0) > 0}
							<span class="rounded-full bg-red-100 px-2 py-0.5 text-xs font-bold text-red-600">
								{community.stats.pending_count}
							</span>
						{/if}
						<ChevronRight size={16} class="text-gray-400 group-hover:text-gray-600" />
					</div>
				</a>

				<a
					href="/communities/{community.id}/admin?tab=pending-posts"
					class="group flex items-center justify-between rounded-xl p-3 transition-colors hover:bg-gray-50"
				>
					<div class="flex items-center gap-3 text-gray-600 group-hover:text-gray-900">
						<FileText size={18} />
						<span class="text-sm font-medium">Pending Posts</span>
					</div>
					<ChevronRight size={16} class="text-gray-400 group-hover:text-gray-600" />
				</a>
			</div>
		</div>
	{/if}

	<!-- Rules Snippet (if any) -->
	{#if community.rules && community.rules.length > 0}
		<div class="rounded-2xl bg-white p-5 shadow-sm ring-1 ring-gray-200">
			<div class="mb-4 flex items-center justify-between">
				<h3 class="text-lg font-semibold text-gray-900">Group Rules</h3>
				<a
					href="/communities/{community.id}/about"
					class="text-xs text-blue-600 hover:text-blue-700">See All</a
				>
			</div>
			<ul class="space-y-3">
				{#each community.rules.slice(0, 3) as rule, i}
					<li class="flex gap-3 text-sm text-gray-500">
						<span class="font-bold text-gray-400">{i + 1}</span>
						<span class="line-clamp-2">{rule.title}</span>
					</li>
				{/each}
			</ul>
		</div>
	{/if}
</div>
