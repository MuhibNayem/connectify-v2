<script lang="ts">
	import { onMount } from 'svelte';
	import {
		getGroupDetails,
		inviteMemberToGroup,
		approveGroupMember,
		rejectGroupMember,
		removeMemberFromGroup,
		addAdminToGroup,
		removeAdminFromGroup,
		updateGroupSettings,
		updateGroup,
		uploadFiles,
		searchFriends
	} from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import type { GroupResponse, UserShortResponse } from '$lib/api'; // Correct import source
	import { fade, fly } from 'svelte/transition';
	import { Camera } from '@lucide/svelte';
	import { websocketMessages } from '$lib/websocket';

	export let showModal: boolean;
	export let groupId: string;
	export let onClose: () => void;

	let group: GroupResponse | null = null;
	let isLoading = false;
	let error: string | null = null;
	let activeTab: 'members' | 'admins' | 'pending' | 'settings' = 'members';

	// Search State for adding members
	let searchTerm = '';
	let searchResults: any[] = [];
	let isSearching = false;
	let searchTimeout: any;

	// Avatar Update State
	let isUpdatingAvatar = false;

	import { onDestroy } from 'svelte';

	$: if (showModal && groupId) {
		fetchGroupDetails();
	}

	// Subscribe to WebSocket messages for real-time updates
	const unsubscribe = websocketMessages.subscribe((event) => {
		if (event?.type === 'GROUP_UPDATED' && group && event.data.id === group.id) {
			console.log('GroupInfoModal received real-time update:', event.data);
			group = event.data;
		}
	});

	onDestroy(() => {
		unsubscribe();
	});

	$: isTriggeredByMe = (userId: string) => userId === auth.state.user?.id;
	$: amIAdmin = group?.admins.some((a) => a.id === auth.state.user?.id) ?? false;

	async function fetchGroupDetails() {
		isLoading = true;
		error = null;
		try {
			group = await getGroupDetails(groupId);
		} catch (e: any) {
			error = e.message;
		} finally {
			isLoading = false;
		}
	}

	async function handleAvatarChange(event: Event) {
		if (!amIAdmin) return;
		const input = event.target as HTMLInputElement;
		if (!input.files || !input.files[0]) return;

		const file = input.files[0];
		isUpdatingAvatar = true;
		try {
			const uploads = await uploadFiles([file]);
			if (uploads && uploads.length > 0) {
				const newAvatar = uploads[0].url;

				// Optimistic Update: Update API first
				await updateGroup(groupId, { avatar: newAvatar });

				// Update Local State immediately
				if (group) {
					group.avatar = newAvatar;
				}

				// Broadcast to frontend components via WebSocket store (acts as event bus)
				if (group) {
					websocketMessages.set({
						type: 'GROUP_UPDATED',
						data: { ...group, avatar: newAvatar }
					});
				}

				// Fetch fresh details in background to be sure
				fetchGroupDetails();
			}
		} catch (e: any) {
			alert(e.message);
		} finally {
			isUpdatingAvatar = false;
		}
	}

	async function handleInvite(userId: string) {
		try {
			await inviteMemberToGroup(groupId, userId);
			searchTerm = '';
			searchResults = [];
			alert('Invitation sent / Member added!'); // Simple feedback
			fetchGroupDetails();
		} catch (e: any) {
			alert(e.message);
		}
	}

	async function handleApprove(userId: string) {
		try {
			await approveGroupMember(groupId, userId);
			fetchGroupDetails();
		} catch (e: any) {
			alert(e.message);
		}
	}

	async function handleReject(userId: string) {
		try {
			await rejectGroupMember(groupId, userId);
			fetchGroupDetails();
		} catch (e: any) {
			alert(e.message);
		}
	}

	async function handleRemoveMember(userId: string) {
		if (!confirm('Are you sure you want to remove this member?')) return;
		try {
			await removeMemberFromGroup(groupId, userId);
			fetchGroupDetails();
		} catch (e: any) {
			alert(e.message);
		}
	}

	async function handleMakeAdmin(userId: string) {
		try {
			await addAdminToGroup(groupId, userId);
			fetchGroupDetails();
		} catch (e: any) {
			alert(e.message);
		}
	}

	async function handleRemoveAdmin(userId: string) {
		if (!confirm('Demote this admin to member?')) return;
		try {
			await removeAdminFromGroup(groupId, userId);
			fetchGroupDetails();
		} catch (e: any) {
			alert(e.message);
		}
	}

	async function toggleApproval(currentVal: boolean) {
		try {
			await updateGroupSettings(groupId, { requires_approval: !currentVal });
			fetchGroupDetails();
		} catch (e: any) {
			alert(e.message);
		}
	}

	function handleSearchInput() {
		clearTimeout(searchTimeout);
		if (!searchTerm.trim()) {
			searchResults = [];
			return;
		}
		isSearching = true;
		searchTimeout = setTimeout(async () => {
			try {
				searchResults = await searchFriends(searchTerm);
			} catch (e: any) {
				console.error(e);
			} finally {
				isSearching = false;
			}
		}, 300);
	}
</script>

{#if showModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" transition:fade>
		<div
			class="flex h-[80vh] w-full max-w-2xl flex-col rounded-xl bg-white shadow-2xl"
			transition:fly={{ y: 20 }}
		>
			<!-- Header -->
			<div class="flex items-center justify-between border-b p-6">
				<div class="flex items-center gap-4">
					<!-- Avatar -->
					<div class="group relative h-16 w-16 flex-shrink-0">
						{#if group?.avatar}
							<img
								src={group.avatar}
								alt={group.name}
								class="h-16 w-16 rounded-full object-cover"
							/>
						{:else}
							<div
								class="flex h-16 w-16 items-center justify-center rounded-full bg-blue-100 text-2xl font-bold text-blue-600"
							>
								{group?.name && group.name[0] ? group.name[0].toUpperCase() : 'G'}
							</div>
						{/if}

						{#if amIAdmin}
							<label
								class="absolute inset-0 flex cursor-pointer items-center justify-center rounded-full bg-black/30 opacity-0 transition-opacity group-hover:opacity-100"
							>
								{#if isUpdatingAvatar}
									<div
										class="spinner h-6 w-6 animate-spin rounded-full border-2 border-white border-t-transparent"
									></div>
								{:else}
									<Camera size={24} color="white" />
								{/if}
								<input
									type="file"
									accept="image/*"
									class="hidden"
									on:change={handleAvatarChange}
									disabled={isUpdatingAvatar}
								/>
							</label>
						{/if}
					</div>

					<h2 class="text-2xl font-bold text-gray-800">
						{group?.name || 'Group Info'}
					</h2>
				</div>
				<button on:click={onClose} class="text-gray-500 hover:text-gray-700">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="1.5"
						stroke="currentColor"
						class="h-6 w-6"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<div class="flex flex-1 overflow-hidden">
				<!-- Sidebar / Tabs -->
				<div class="w-48 overflow-y-auto border-r bg-gray-50 p-4">
					<div class="flex flex-col space-y-2">
						{#each ['members', 'admins'] as tab}
							<button
								class="rounded-lg px-4 py-2 text-left text-sm font-medium transition-colors {activeTab ===
								tab
									? 'bg-blue-100 text-blue-700'
									: 'text-gray-600 hover:bg-gray-100'}"
								on:click={() => (activeTab = tab as any)}
							>
								{tab.charAt(0).toUpperCase() + tab.slice(1)}
							</button>
						{/each}
						{#if amIAdmin}
							<button
								class="rounded-lg px-4 py-2 text-left text-sm font-medium transition-colors {activeTab ===
								'pending'
									? 'bg-blue-100 text-blue-700'
									: 'text-gray-600 hover:bg-gray-100'}"
								on:click={() => (activeTab = 'pending')}
							>
								Pending ({group?.pending_members?.length || 0})
							</button>
							<button
								class="rounded-lg px-4 py-2 text-left text-sm font-medium transition-colors {activeTab ===
								'settings'
									? 'bg-blue-100 text-blue-700'
									: 'text-gray-600 hover:bg-gray-100'}"
								on:click={() => (activeTab = 'settings')}
							>
								Settings
							</button>
						{/if}
					</div>
				</div>

				<!-- Content -->
				<div class="flex-1 overflow-y-auto p-6">
					{#if isLoading && !group}
						<div class="flex h-full items-center justify-center">Loading...</div>
					{:else if group}
						{#if activeTab === 'members'}
							<div class="space-y-6">
								<div>
									<h3 class="font-semibold text-gray-900">Add Members</h3>
									<div class="relative mt-2">
										<input
											type="text"
											placeholder="Search friends to add..."
											bind:value={searchTerm}
											on:input={handleSearchInput}
											class="w-full rounded-lg border border-gray-300 px-4 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
										/>
										{#if searchResults.length > 0}
											<div class="absolute z-10 mt-1 w-full rounded-lg border bg-white shadow-lg">
												{#each searchResults as user}
													<button
														class="flex w-full items-center justify-between px-4 py-2 hover:bg-gray-50"
														on:click={() => handleInvite(user.id)}
													>
														<div class="flex items-center gap-2">
															<span
																class="flex h-8 w-8 items-center justify-center rounded-full bg-blue-100 text-xs font-bold text-blue-600"
															>
																{(user.username && user.username[0]
																	? user.username[0]
																	: 'U'
																).toUpperCase()}
															</span>
															<span>{user.username}</span>
														</div>
														<span class="text-xs text-blue-600">Add</span>
													</button>
												{/each}
											</div>
										{/if}
									</div>
								</div>

								<div class="space-y-4">
									<h3 class="font-semibold text-gray-900">Members ({group.members.length})</h3>
									{#each group.members as member}
										<div class="flex items-center justify-between rounded-lg border p-3">
											<div class="flex items-center gap-3">
												<div
													class="flex h-10 w-10 items-center justify-center rounded-full bg-gray-200"
												>
													{#if member.avatar}
														<img
															src={member.avatar}
															alt={member.username}
															class="h-10 w-10 rounded-full"
														/>
													{:else}
														<span class="text-lg font-bold text-gray-500"
															>{(member.username && member.username[0]
																? member.username[0]
																: 'M'
															).toUpperCase()}</span
														>
													{/if}
												</div>
												<div>
													<p class="font-medium text-gray-900">{member.username}</p>
													<p class="text-xs text-gray-500">{member.email}</p>
												</div>
											</div>
											{#if amIAdmin && member.id !== auth.state.user?.id}
												<div class="flex gap-2">
													<button
														disabled={group.admins.some((a) => a.id === member.id)}
														on:click={() => {
															if (!group?.admins.some((a) => a.id === member.id))
																handleMakeAdmin(member.id);
														}}
														class="text-xs text-blue-600 hover:underline disabled:text-gray-400"
													>
														{group.admins.some((a) => a.id === member.id) ? 'Admin' : 'Make Admin'}
													</button>
													<button
														on:click={() => handleRemoveMember(member.id)}
														class="text-xs text-red-600 hover:underline">Remove</button
													>
												</div>
											{/if}
										</div>
									{/each}
								</div>
							</div>
						{/if}

						{#if activeTab === 'admins'}
							<div class="space-y-4">
								<h3 class="font-semibold text-gray-900">Admins ({group.admins.length})</h3>
								{#each group.admins as admin}
									<div class="flex items-center justify-between rounded-lg border p-3">
										<div class="flex items-center gap-3">
											<div
												class="flex h-10 w-10 items-center justify-center rounded-full bg-purple-100"
											>
												{#if admin.avatar}
													<img
														src={admin.avatar}
														alt={admin.username}
														class="h-10 w-10 rounded-full"
													/>
												{:else}
													<span class="text-lg font-bold text-purple-600"
														>{(admin.username && admin.username[0]
															? admin.username[0]
															: 'A'
														).toUpperCase()}</span
													>
												{/if}
											</div>
											<div>
												<p class="font-medium text-gray-900">{admin.username}</p>
												<p class="text-xs text-gray-500">Admin</p>
											</div>
										</div>
										{#if amIAdmin && admin.id !== auth.state.user?.id}
											<button
												on:click={() => handleRemoveAdmin(admin.id)}
												class="text-xs text-red-600 hover:underline"
											>
												Demote
											</button>
										{/if}
									</div>
								{/each}
							</div>
						{/if}

						{#if activeTab === 'pending' && amIAdmin}
							<div class="space-y-4">
								<h3 class="font-semibold text-gray-900">
									Pending Requests ({group.pending_members?.length || 0})
								</h3>
								{#if !group.pending_members || group.pending_members.length === 0}
									<p class="text-gray-500">No pending requests.</p>
								{:else}
									{#each group.pending_members as pending}
										<div class="flex items-center justify-between rounded-lg border p-3">
											<div class="flex items-center gap-3">
												<div
													class="flex h-10 w-10 items-center justify-center rounded-full bg-yellow-100"
												>
													<span class="font-bold text-yellow-600"
														>{(pending.username && pending.username[0]
															? pending.username[0]
															: 'P'
														).toUpperCase()}</span
													>
												</div>
												<p class="font-medium text-gray-900">{pending.username}</p>
											</div>
											<div class="flex gap-2">
												<button
													on:click={() => handleApprove(pending.id)}
													class="rounded bg-green-50 px-3 py-1 text-xs font-semibold text-green-700 hover:bg-green-100"
												>
													Approve
												</button>
												<button
													on:click={() => handleReject(pending.id)}
													class="rounded bg-red-50 px-3 py-1 text-xs font-semibold text-red-700 hover:bg-red-100"
												>
													Reject
												</button>
											</div>
										</div>
									{/each}
								{/if}
							</div>
						{/if}

						{#if activeTab === 'settings' && amIAdmin}
							<div class="space-y-6">
								<h3 class="font-semibold text-gray-900">Group Settings</h3>
								<div class="flex items-center justify-between rounded-lg border p-4">
									<div>
										<p class="font-medium text-gray-900">Require Admin Approval</p>
										<p class="text-sm text-gray-500">
											New members must be approved by an admin before joining.
										</p>
									</div>
									<button
										on:click={() => toggleApproval(group?.settings?.requires_approval || false)}
										class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 {group
											?.settings?.requires_approval
											? 'bg-blue-600'
											: 'bg-gray-200'}"
									>
										<span
											class="pc pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {group
												?.settings?.requires_approval
												? 'translate-x-5'
												: 'translate-x-0'}"
										></span>
									</button>
								</div>
							</div>
						{/if}
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}
