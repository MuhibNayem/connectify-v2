<script lang="ts">
	import { getContext } from 'svelte';
	import type { Writable } from 'svelte/store';
	import type { Community } from '$lib/api';
	import { Shield, Info, Globe, Lock, Calendar, Eye, EyeOff } from '@lucide/svelte';

	const communityStore = getContext<Writable<Community>>('community');
	let community = $derived($communityStore);
</script>

<div class="grid grid-cols-1 gap-6 md:grid-cols-3">
	<div class="col-span-2 space-y-6">
		<!-- About Section -->
		<div class="rounded-2xl bg-white p-6 shadow-sm ring-1 ring-gray-200">
			<h2 class="mb-4 text-2xl font-bold text-gray-900">About this group</h2>
			<p class="whitespace-pre-line leading-relaxed text-gray-600">
				{community?.description || 'No description provided.'}
			</p>

			<div class="mt-6 grid grid-cols-1 gap-4 border-t border-gray-100 pt-6 sm:grid-cols-2">
				<div class="flex items-start gap-3">
					{#if community?.privacy === 'public'}
						<Globe class="mt-1 text-gray-700" size={20} />
						<div>
							<h3 class="font-semibold text-gray-900">Public</h3>
							<p class="text-sm text-gray-500">
								Anyone can see who's in the group and what they post.
							</p>
						</div>
					{:else}
						<Lock class="mt-1 text-gray-700" size={20} />
						<div>
							<h3 class="font-semibold text-gray-900">Private</h3>
							<p class="text-sm text-gray-500">
								Only members can see who's in the group and what they post.
							</p>
						</div>
					{/if}
				</div>

				<div class="flex items-start gap-3">
					{#if community?.visibility === 'visible'}
						<Eye class="mt-1 text-gray-700" size={20} />
						<div>
							<h3 class="font-semibold text-gray-900">Visible</h3>
							<p class="text-sm text-gray-500">Anyone can find this group.</p>
						</div>
					{:else}
						<EyeOff class="mt-1 text-gray-700" size={20} />
						<div>
							<h3 class="font-semibold text-gray-900">Hidden</h3>
							<p class="text-sm text-gray-500">Only members can find this group.</p>
						</div>
					{/if}
				</div>

				<div class="flex items-start gap-3">
					<Shield class="mt-1 text-gray-700" size={20} />
					<div>
						<h3 class="font-semibold text-gray-900">History</h3>
						<p class="text-sm text-gray-500">
							Created {new Date(community?.created_at).toLocaleDateString(undefined, {
								year: 'numeric',
								month: 'long',
								day: 'numeric'
							})}
						</p>
					</div>
				</div>

				<!-- Category/General -->
				<div class="flex items-start gap-3">
					<Info class="mt-1 text-gray-700" size={20} />
					<div>
						<h3 class="font-semibold text-gray-900">General</h3>
						<p class="text-sm text-gray-500">
							Category: {community?.category || 'General'}
						</p>
					</div>
				</div>
			</div>
		</div>

		<!-- Rules Section -->
		{#if community?.rules && community.rules.length > 0}
			<div class="rounded-2xl bg-white p-6 shadow-sm ring-1 ring-gray-200">
				<h2 class="mb-6 text-xl font-bold text-gray-900">Group Rules</h2>
				<ul class="space-y-6">
					{#each community.rules as rule, i}
						<li class="flex gap-4">
							<span
								class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-gray-100 font-bold text-gray-900"
							>
								{i + 1}
							</span>
							<div>
								<h3 class="mb-1 font-semibold text-gray-900">{rule.title}</h3>
								<p class="text-sm text-gray-500">{rule.description}</p>
							</div>
						</li>
					{/each}
				</ul>
			</div>
		{/if}
	</div>

	<!-- Sidebar / Stats -->
	<div class="col-span-1 space-y-6">
		<div class="rounded-2xl bg-white p-6 shadow-sm ring-1 ring-gray-200">
			<h3 class="mb-4 font-bold text-gray-900">Activity</h3>
			<div class="space-y-4">
				<div class="flex items-center gap-3 text-gray-600">
					<Calendar size={18} />
					<span>Created {new Date(community?.created_at).toLocaleDateString()}</span>
				</div>
				<div class="flex items-center gap-3 text-gray-600">
					<Info size={18} />
					<span>{community?.stats?.post_count || 0} posts today</span>
				</div>
				<div class="flex items-center gap-3 text-gray-600">
					<Info size={18} />
					<span>{community?.stats?.member_count || 0} total members</span>
				</div>
			</div>
		</div>
	</div>
</div>
