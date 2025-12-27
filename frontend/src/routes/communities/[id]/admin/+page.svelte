<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { getContext } from 'svelte';
	import type { Writable } from 'svelte/store';
	import {
		getPosts,
		updatePostStatus,
		updateCommunitySettings,
		getCommunityPendingMembers,
		approveCommunityMember,
		rejectCommunityMember
	} from '$lib/api';
	import type { Community } from '$lib/api';
	import type { Post, User } from '$lib/types';
	import {
		Settings,
		CheckCircle,
		Users,
		FileText,
		AlertTriangle,
		Trash2,
		Check,
		X
	} from '@lucide/svelte';

	const communityStore = getContext<Writable<Community>>('community');
	let community = $derived($communityStore);

	// Drive state from URL
	let activeTab = $derived($page.url.searchParams.get('tab') || 'pending-posts');

	let pendingPosts: Post[] = $state([]);
	let pendingMembers: User[] = $state([]);
	let loading = $state(true);

	// Settings Form State
	let settingsForm = $state({
		name: '',
		description: '',
		privacy: '',
		visibility: '',
		require_post_approval: false,
		require_join_approval: false,
		allow_member_posts: true // Default to true for UI, but will overwrite from community
	});

	$effect(() => {
		if (community) {
			settingsForm = {
				name: community.name,
				description: community.description,
				privacy: community.privacy,
				visibility: community.visibility || 'visible',
				require_post_approval: community.settings?.require_post_approval || false,
				require_join_approval: community.settings?.require_join_approval || false,
				allow_member_posts:
					community.settings?.allow_member_posts !== undefined
						? community.settings.allow_member_posts
						: true
			};
		}
	});

	async function handleDelete() {
		if (
			!confirm(
				'Are you ABSOLUTELY SURE? This action cannot be undone. This will permanently delete the community and all its contents.'
			)
		)
			return;

		// TODO: Implement deleteCommunity API
		alert('Delete functionality not yet implemented in backend.');
	}

	async function loadPendingPosts() {
		loading = true;
		try {
			const res = await getPosts({
				community_id: community.id,
				status: 'pending',
				limit: 50
			});
			pendingPosts = res.posts || [];
		} catch (e) {
			console.error('Failed to load pending posts:', e);
		} finally {
			loading = false;
		}
	}

	async function handleApprovePost(post: Post) {
		try {
			// Optimistic update
			pendingPosts = pendingPosts.filter((p) => p.id !== post.id);
			await updatePostStatus(post.id, 'active');
		} catch (e) {
			console.error('Failed to approve post:', e);
			loadPendingPosts();
		}
	}

	async function handleDeclinePost(post: Post) {
		try {
			pendingPosts = pendingPosts.filter((p) => p.id !== post.id);
			await updatePostStatus(post.id, 'declined');
		} catch (e) {
			console.error('Failed to decline post:', e);
			loadPendingPosts();
		}
	}

	async function loadMemberRequests() {
		loading = true;
		try {
			const res = await getCommunityPendingMembers(community.id);
			pendingMembers = res.users || [];
		} catch (e) {
			console.error('Failed to load member requests:', e);
		} finally {
			loading = false;
		}
	}

	async function handleApproveMember(user: User) {
		try {
			pendingMembers = pendingMembers.filter((u) => u.id !== user.id);
			await approveCommunityMember(community.id, user.id);
		} catch (e) {
			console.error('Failed to approve member:', e);
			loadMemberRequests();
		}
	}

	async function handleRejectMember(user: User) {
		try {
			pendingMembers = pendingMembers.filter((u) => u.id !== user.id);
			await rejectCommunityMember(community.id, user.id);
		} catch (e) {
			console.error('Failed to reject member:', e);
			loadMemberRequests();
		}
	}

	async function handleSaveSettings() {
		try {
			// Cast privacy to correct string literal type or standard string from form
			const payload = {
				...settingsForm,
				privacy: settingsForm.privacy as 'public' | 'closed' | 'secret',
				visibility: settingsForm.visibility as 'visible' | 'hidden'
			};
			await updateCommunitySettings(community.id, payload);
			alert('Settings saved successfully');
		} catch (e) {
			console.error('Failed to update settings:', e);
			alert('Failed to update settings');
		}
	}

	$effect(() => {
		if (community) {
			if (activeTab === 'pending-posts') {
				loadPendingPosts();
			} else if (activeTab === 'member-requests') {
				loadMemberRequests();
			}
		}
	});
</script>

<div class="grid grid-cols-1 gap-6 md:grid-cols-4">
	<!-- Sidebar Nav -->
	<div class="col-span-1 h-fit rounded-xl bg-white p-2 shadow-sm ring-1 ring-gray-200">
		<nav class="flex flex-col gap-1">
			<a
				href="?tab=pending-posts"
				class="flex items-center gap-3 rounded-lg px-4 py-3 text-left text-sm font-medium transition-colors
                {activeTab === 'pending-posts'
					? 'bg-blue-50 text-blue-600'
					: 'text-gray-500 hover:bg-gray-50 hover:text-gray-900'}"
			>
				<FileText size={18} />
				Pending Posts
			</a>
			<a
				href="?tab=member-requests"
				class="flex items-center gap-3 rounded-lg px-4 py-3 text-left text-sm font-medium transition-colors
                {activeTab === 'member-requests'
					? 'bg-blue-50 text-blue-600'
					: 'text-gray-500 hover:bg-gray-50 hover:text-gray-900'}"
			>
				<Users size={18} />
				Member Requests
			</a>
			<a
				href="?tab=settings"
				class="flex items-center gap-3 rounded-lg px-4 py-3 text-left text-sm font-medium transition-colors
                {activeTab === 'settings'
					? 'bg-blue-50 text-blue-600'
					: 'text-gray-500 hover:bg-gray-50 hover:text-gray-900'}"
			>
				<Settings size={18} />
				Group Settings
			</a>
		</nav>
	</div>

	<!-- Content Area -->
	<div class="col-span-3">
		{#if activeTab === 'pending-posts'}
			<div class="min-h-[500px] rounded-2xl bg-white p-6 shadow-sm ring-1 ring-gray-200">
				<h2 class="mb-6 text-xl font-bold text-gray-900">Pending Posts</h2>

				{#if loading}
					<p class="text-gray-500">Loading...</p>
				{:else if pendingPosts.length > 0}
					<div class="space-y-6">
						{#each pendingPosts as post (post.id)}
							<div class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
								<!-- Post Preview (Simplified) -->
								<div class="mb-3 flex items-start gap-3">
									<img src={post.author.avatar} alt="avatar" class="h-10 w-10 rounded-full" />
									<div>
										<p class="font-semibold text-gray-900">{post.author.full_name}</p>
										<p class="text-xs text-gray-500">
											{new Date(post.created_at).toLocaleString()}
										</p>
									</div>
								</div>
								<p class="mb-4 text-gray-700">{post.content}</p>

								<!-- Actions -->
								<div class="flex gap-3 border-t border-gray-100 pt-3">
									<button
										class="flex-1 rounded-lg bg-blue-600 py-2 text-sm font-medium text-white hover:bg-blue-700"
										on:click={() => handleApprovePost(post)}
									>
										Approve
									</button>
									<button
										class="flex-1 rounded-lg bg-gray-100 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200"
										on:click={() => handleDeclinePost(post)}
									>
										Decline
									</button>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<div class="flex flex-col items-center justify-center py-12 text-gray-500">
						<CheckCircle size={48} class="mb-4 text-gray-300" />
						<p>No pending posts.</p>
					</div>
				{/if}
			</div>
		{:else if activeTab === 'member-requests'}
			<div class="min-h-[500px] rounded-2xl bg-white p-6 shadow-sm ring-1 ring-gray-200">
				<h2 class="mb-6 text-xl font-bold text-gray-900">Member Requests</h2>

				{#if loading}
					<p class="text-gray-500">Loading...</p>
				{:else if pendingMembers.length > 0}
					<div class="space-y-4">
						{#each pendingMembers as member}
							<div
								class="flex items-center justify-between rounded-xl border border-gray-100 bg-white p-4 shadow-sm"
							>
								<div class="flex items-center gap-3">
									<img
										src={member.avatar ||
											`https://ui-avatars.com/api/?name=${member.full_name || member.username}`}
										alt={member.username}
										class="h-12 w-12 rounded-full object-cover"
									/>
									<div>
										<h3 class="font-semibold text-gray-900">
											{member.full_name || member.username}
										</h3>
										<p class="text-xs text-gray-500">@{member.username}</p>
									</div>
								</div>
								<div class="flex gap-2">
									<button
										class="flex items-center gap-1 rounded-lg bg-green-50 px-3 py-1.5 text-sm font-medium text-green-600 transition hover:bg-green-100"
										on:click={() => handleApproveMember(member)}
									>
										<Check size={16} />
										Approve
									</button>
									<button
										class="flex items-center gap-1 rounded-lg bg-red-50 px-3 py-1.5 text-sm font-medium text-red-600 transition hover:bg-red-100"
										on:click={() => handleRejectMember(member)}
									>
										<X size={16} />
										Reject
									</button>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<div class="flex flex-col items-center justify-center py-12 text-gray-500">
						<Users size={48} class="mb-4 text-gray-300" />
						<p>No new member requests.</p>
					</div>
				{/if}
			</div>
		{:else if activeTab === 'settings'}
			<div class="rounded-2xl bg-white p-6 shadow-sm ring-1 ring-gray-200">
				<h2 class="mb-6 text-xl font-bold text-gray-900">Group Settings</h2>
				<div class="max-w-xl space-y-6">
					<div>
						<label class="mb-1 block text-sm font-medium text-gray-700">Name</label>
						<input
							type="text"
							bind:value={settingsForm.name}
							class="w-full rounded-lg border border-gray-300 bg-white px-4 py-2 text-gray-900 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
						/>
					</div>
					<div>
						<label class="mb-1 block text-sm font-medium text-gray-700">Description</label>
						<textarea
							bind:value={settingsForm.description}
							class="h-24 w-full rounded-lg border border-gray-300 bg-white px-4 py-2 text-gray-900 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
						/>
					</div>

					<div class="grid grid-cols-2 gap-4">
						<div>
							<label class="mb-1 block text-sm font-medium text-gray-700">Privacy</label>
							<select
								bind:value={settingsForm.privacy}
								class="w-full rounded-lg border border-gray-300 bg-white px-4 py-2 text-gray-900 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
							>
								<option value="public">Public</option>
								<option value="private">Private</option>
							</select>
						</div>
						<div>
							<label class="mb-1 block text-sm font-medium text-gray-700">Visibility</label>
							<select
								bind:value={settingsForm.visibility}
								class="w-full rounded-lg border border-gray-300 bg-white px-4 py-2 text-gray-900 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
							>
								<option value="visible">Visible</option>
								<option value="hidden">Hidden</option>
							</select>
						</div>
					</div>

					<div class="border-t border-gray-100 pt-4">
						<h3 class="mb-4 text-lg font-semibold text-gray-900">Privacy & Permissions</h3>

						<div class="mb-4 flex items-center justify-between">
							<div>
								<p class="font-medium text-gray-900">Allow Member Posts</p>
								<p class="text-xs text-gray-500">If disabled, only admins can create posts.</p>
							</div>
							<input
								type="checkbox"
								bind:checked={settingsForm.allow_member_posts}
								class="toggle"
							/>
						</div>

						<div class="mb-4 flex items-center justify-between">
							<div>
								<p class="font-medium text-gray-900">Require Post Approval</p>
								<p class="text-xs text-gray-500">
									Admins must approve posts before they are visible.
								</p>
							</div>
							<input
								type="checkbox"
								bind:checked={settingsForm.require_post_approval}
								class="toggle"
							/>
						</div>
						<div class="flex items-center justify-between">
							<div>
								<p class="font-medium text-gray-900">Require Join Approval</p>
								<p class="text-xs text-gray-500">Admins must approve new members.</p>
							</div>
							<input
								type="checkbox"
								bind:checked={settingsForm.require_join_approval}
								class="toggle"
							/>
						</div>
					</div>

					<div class="pt-6">
						<button
							class="w-full rounded-lg bg-blue-600 py-3 font-bold text-white shadow-lg shadow-blue-900/10 hover:bg-blue-700"
							on:click={handleSaveSettings}
						>
							Save Changes
						</button>
					</div>
				</div>
			</div>

			<!-- Danger Zone -->
			<div class="mt-8 overflow-hidden rounded-2xl border border-red-100 bg-white shadow-sm">
				<div class="border-b border-red-100 bg-red-50/50 p-6">
					<h2 class="flex items-center gap-2 text-xl font-bold text-red-600">
						<AlertTriangle class="h-5 w-5" /> Danger Zone
					</h2>
				</div>
				<div class="flex items-center justify-between p-6">
					<div>
						<h3 class="font-bold text-gray-900">Delete Community</h3>
						<p class="text-sm text-gray-500">
							Once you delete a community, there is no going back. Please be certain.
						</p>
					</div>
					<button
						on:click={handleDelete}
						class="flex items-center gap-2 rounded-xl bg-red-100 px-6 py-2.5 font-bold text-red-700 transition hover:bg-red-200"
					>
						<Trash2 class="h-4 w-4" />
						Delete Community
					</button>
				</div>
			</div>
		{/if}
	</div>
</div>
