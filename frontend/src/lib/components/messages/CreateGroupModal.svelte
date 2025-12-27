<script lang="ts">
	import { createGroup, searchFriends, uploadFiles } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import type { User } from '$lib/types';
	import { Camera } from '@lucide/svelte';

	export let showModal: boolean;
	export let onGroupCreated: (group: any) => void;

	let groupName = '';
	let selectedParticipants: string[] = [];
	let selectedFriends: User[] = []; // Store full user objects for chips
	let searchResults: User[] = [];
	let isLoadingFriends = false;
	let error: string | null = null;
	let isCreatingGroup = false;
	let searchTerm = '';
	let searchTimeout: any;

	// Avatar State
	let avatarFile: File | null = null;
	let avatarPreview: string | null = null;
	let avatarInput: HTMLInputElement;

	$: if (showModal) {
		groupName = '';
		selectedParticipants = [];
		selectedFriends = [];
		searchTerm = '';
		searchResults = [];
		error = null;
		avatarFile = null;
		avatarPreview = null;
	}

	function handleFileChange(event: Event) {
		const input = event.target as HTMLInputElement;
		if (input.files && input.files[0]) {
			avatarFile = input.files[0];
			avatarPreview = URL.createObjectURL(avatarFile);
		}
	}

	function handleSearchInput() {
		clearTimeout(searchTimeout);
		if (!searchTerm.trim()) {
			searchResults = [];
			return;
		}
		isLoadingFriends = true;
		searchTimeout = setTimeout(async () => {
			try {
				searchResults = await searchFriends(searchTerm);
				error = null;
			} catch (e: any) {
				error = e.message || 'Failed to search friends.';
			} finally {
				isLoadingFriends = false;
			}
		}, 300); // 300ms debounce
	}

	function addParticipant(user: User) {
		if (!selectedParticipants.includes(user.id)) {
			selectedParticipants = [...selectedParticipants, user.id];
			selectedFriends = [...selectedFriends, user];
			searchTerm = '';
			searchResults = [];
		}
	}

	function removeParticipant(userId: string) {
		selectedParticipants = selectedParticipants.filter((id) => id !== userId);
	}

	async function handleSubmit(e?: Event) {
		if (e) e.preventDefault();
		if (!groupName.trim() || selectedParticipants.length === 0 || isCreatingGroup) return;

		isCreatingGroup = true;
		try {
			let avatarUrl = '';
			if (avatarFile) {
				const uploads = await uploadFiles([avatarFile]);
				if (uploads.length > 0) {
					avatarUrl = uploads[0].url;
				}
			}

			// Add current user to participants if not already there
			const allParticipants = auth.state.user
				? [...new Set([...selectedParticipants, auth.state.user.id])]
				: selectedParticipants;

			const result = await createGroup({
				name: groupName,
				member_ids: allParticipants,
				avatar: avatarUrl
			});

			// Optimistic update: pass group data to callback
			onGroupCreated(result);
			showModal = false;
		} catch (e: any) {
			error = e.message || 'Failed to create group.';
		} finally {
			isCreatingGroup = false;
		}
	}
</script>

{#if showModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto bg-gray-900 bg-opacity-50"
	>
		<div class="relative w-full max-w-md rounded-lg bg-white p-6 shadow-lg">
			<h3 class="mb-4 text-xl font-semibold">Create New Group</h3>
			<form onsubmit={handleSubmit}>
				<!-- Avatar Upload -->
				<div class="mb-6 flex justify-center">
					<button
						type="button"
						class="group relative h-24 w-24 overflow-hidden rounded-full bg-gray-100 transition hover:opacity-90"
						onclick={() => avatarInput.click()}
					>
						{#if avatarPreview}
							<img src={avatarPreview} alt="Group Avatar" class="h-full w-full object-cover" />
						{:else}
							<div class="flex h-full items-center justify-center text-gray-400">
								<Camera size={32} />
							</div>
						{/if}
						<div
							class="absolute inset-0 flex items-center justify-center bg-black/30 opacity-0 transition-opacity group-hover:opacity-100"
						>
							<span class="text-xs font-medium text-white">Change</span>
						</div>
					</button>
					<input
						type="file"
						accept="image/*"
						bind:this={avatarInput}
						onchange={handleFileChange}
						hidden
					/>
				</div>

				<div class="mb-4">
					<label for="groupName" class="mb-2 block text-sm font-medium text-gray-700"
						>Group Name</label
					>
					<input
						type="text"
						id="groupName"
						bind:value={groupName}
						class="w-full rounded-lg border border-gray-300 p-2.5 text-sm focus:border-blue-500 focus:ring-blue-500"
						placeholder="Enter group name"
						required
					/>
				</div>

				<div class="mb-4">
					<label class="mb-2 block text-sm font-medium text-gray-700">Select Participants</label>

					<!-- Selected Chips -->
					<div class="mb-2 flex flex-wrap gap-2">
						{#each selectedFriends as participant (participant.id)}
							<div
								class="flex items-center rounded-full bg-blue-100 px-3 py-1 text-sm text-blue-800"
							>
								<span>{participant.username}</span>
								<button
									type="button"
									class="ml-2 text-blue-600 hover:text-blue-800 focus:outline-none"
									onclick={() => removeParticipant(participant.id)}
								>
									&times;
								</button>
							</div>
						{/each}
					</div>

					<!-- Search and List -->
					<div class="relative">
						<input
							type="text"
							bind:value={searchTerm}
							oninput={handleSearchInput}
							placeholder="Search friends..."
							class="w-full rounded-lg border border-gray-300 p-2.5 text-sm focus:border-blue-500 focus:ring-blue-500"
						/>

						{#if isLoadingFriends}
							<p class="mt-2 text-sm text-gray-500">Searching...</p>
						{:else if error}
							<p class="mt-2 text-sm text-red-500">{error}</p>
						{:else if searchResults.length > 0}
							{@const visibleResults = searchResults.filter(
								(u) => !selectedParticipants.includes(u.id)
							)}
							{#if visibleResults.length > 0}
								<div
									class="mt-1 max-h-48 overflow-y-auto rounded-lg border border-gray-200 bg-white shadow-sm"
								>
									{#each visibleResults as friend (friend.id)}
										<button
											type="button"
											class="flex w-full items-center px-4 py-2 text-left text-sm hover:bg-gray-100"
											onclick={() => addParticipant(friend)}
										>
											<div
												class="flex h-8 w-8 items-center justify-center rounded-full bg-gray-200 text-xs font-bold text-gray-600"
											>
												{friend.username.charAt(0).toUpperCase()}
											</div>
											<span class="ml-3 text-gray-900">{friend.username}</span>
											{#if friend.full_name}
												<span class="ml-2 text-gray-500">({friend.full_name})</span>
											{/if}
										</button>
									{/each}
								</div>
							{:else}
								<p class="mt-2 text-sm text-gray-500">All matching friends selected.</p>
							{/if}
						{:else if searchTerm && !isLoadingFriends}
							<p class="mt-2 text-sm text-gray-500">No matching friends found.</p>
						{/if}
					</div>
				</div>

				{#if error}
					<p class="mb-4 text-sm text-red-500">{error}</p>
				{/if}

				<div class="flex justify-end space-x-2">
					<button
						type="button"
						onclick={() => (showModal = false)}
						class="rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
					>
						Cancel
					</button>
					<button
						type="submit"
						disabled={isCreatingGroup || !groupName.trim() || selectedParticipants.length === 0}
						class="inline-flex justify-center rounded-lg border border-transparent bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50"
					>
						{#if isCreatingGroup}Creating...{:else}Create Group{/if}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
