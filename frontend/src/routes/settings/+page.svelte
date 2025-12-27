<script lang="ts">
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth.svelte';
	import { updateUserProfile, updatePrivacySettings, updateNotificationSettings } from '$lib/api';
	import { fade } from 'svelte/transition';
	import { goto } from '$app/navigation';
	import AppHeader from '$lib/components/layout/AppHeader.svelte';

	// Tabs
	const tabs = [
		{ id: 'general', label: 'General' },
		{ id: 'privacy', label: 'Privacy' },
		{ id: 'notifications', label: 'Notifications' },
		{ id: 'security', label: 'Security' }
	];
	let activeTab = 'general';

	// Form Data
	let formData = {
		fullName: '',
		bio: '',
		location: '',
		phoneNumber: '',
		dateOfBirth: '',
		gender: ''
	};

	let privacyData: any = {
		default_post_privacy: 'PUBLIC',
		can_see_my_friends_list: 'FRIENDS',
		can_send_me_friend_requests: 'EVERYONE',
		can_tag_me_in_posts: 'FRIENDS'
	};

	let notificationData: any = {
		email_notifications: true,
		push_notifications: true,
		notify_on_friend_request: true,
		notify_on_comment: true,
		notify_on_like: true,
		notify_on_tag: true,
		notify_on_message: true
	};

	let securityData = {
		is_encryption_enabled: true
	};

	let isLoading = false;
	let message = { type: '', text: '' };

	onMount(() => {
		loadUserData();
	});

	function loadUserData() {
		const user = auth.state.user;
		if (!user) return;

		formData = {
			fullName: user.full_name || '',
			bio: user.bio || '',
			location: user.location || '',
			phoneNumber: user.phone_number || '',
			dateOfBirth: user.date_of_birth ? user.date_of_birth.split('T')[0] : '', // Format for input type=date
			gender: user.gender || ''
		};

		if (user.privacy_settings) {
			privacyData = { ...user.privacy_settings };
		}

		if (user.notification_settings) {
			notificationData = { ...user.notification_settings };
		}

		// Security settings
		securityData = {
			is_encryption_enabled: user.is_encryption_enabled !== false // Default to true if undefined
		};
	}

	async function handleGeneralSubmit() {
		isLoading = true;
		message = { type: '', text: '' };
		try {
			const payload: any = {
				full_name: formData.fullName,
				bio: formData.bio,
				location: formData.location,
				phone_number: formData.phoneNumber,
				gender: formData.gender
			};
			if (formData.dateOfBirth) {
				payload.date_of_birth = new Date(formData.dateOfBirth).toISOString();
			}

			const updatedUser = await updateUserProfile(payload);
			auth.updateUser(updatedUser); // Update local store
			message = { type: 'success', text: 'Profile updated successfully!' };
		} catch (e: any) {
			message = { type: 'error', text: e.message || 'Failed to update profile.' };
		} finally {
			isLoading = false;
		}
	}

	async function handlePrivacySubmit() {
		isLoading = true;
		message = { type: '', text: '' };
		try {
			await updatePrivacySettings(privacyData);
			// Manually update store for nested objects if backend doesn't return full user
			// Re-fetch or manually patch for now:
			if (auth.state.user) {
				auth.updateUser({ ...auth.state.user, privacy_settings: privacyData as any });
			}

			message = { type: 'success', text: 'Privacy settings updated!' };
		} catch (e: any) {
			message = { type: 'error', text: e.message || 'Failed to update settings.' };
		} finally {
			isLoading = false;
		}
	}

	async function handleNotificationSubmit() {
		isLoading = true;
		message = { type: '', text: '' };
		try {
			// Convert to pointers logic happens in backend, we send full object here
			await updateNotificationSettings(notificationData);

			if (auth.state.user) {
				auth.updateUser({ ...auth.state.user, notification_settings: notificationData });
			}

			message = { type: 'success', text: 'Notification preferences saved!' };
		} catch (e: any) {
			message = { type: 'error', text: e.message || 'Failed to save preferences.' };
		} finally {
			isLoading = false;
		}
	}

	async function handleSecuritySubmit() {
		isLoading = true;
		message = { type: '', text: '' };
		try {
			const updatedUser = await updateUserProfile({
				is_encryption_enabled: securityData.is_encryption_enabled
			});
			auth.updateUser(updatedUser);
			message = { type: 'success', text: 'Security settings updated!' };
		} catch (e: any) {
			message = { type: 'error', text: e.message || 'Failed to save security settings.' };
		} finally {
			isLoading = false;
		}
	}
</script>

<AppHeader />

<div class="min-h-screen bg-gray-50 pb-12 pt-20">
	<div class="mx-auto max-w-5xl px-4 sm:px-6 lg:px-8">
		<h1 class="mb-8 text-3xl font-bold text-gray-900">Settings</h1>

		<div class="flex flex-col gap-8 lg:flex-row">
			<!-- Sidebar Navigation -->
			<nav class="w-full flex-shrink-0 lg:w-64">
				<div class="overflow-hidden rounded-lg bg-white shadow">
					{#each tabs as tab}
						<button
							class="w-full border-l-4 px-6 py-4 text-left text-sm font-medium transition-colors
                            {activeTab === tab.id
								? 'border-blue-500 bg-blue-50 text-blue-700'
								: 'border-transparent text-gray-600 hover:bg-gray-50 hover:text-gray-900'}"
							on:click={() => {
								activeTab = tab.id;
								message = { type: '', text: '' };
							}}
						>
							{tab.label}
						</button>
					{/each}
				</div>
				<div class="mt-4 text-center">
					<a href={`/profile/${auth.state.user?.id}`} class="text-sm text-blue-600 hover:underline"
						>View my Profile</a
					>
				</div>
			</nav>

			<!-- Content Area -->
			<main class="min-h-[500px] flex-1 rounded-lg bg-white p-6 shadow lg:p-8">
				{#if message.text}
					<div
						class="mb-6 rounded-md p-4 {message.type === 'success'
							? 'bg-green-50 text-green-700'
							: 'bg-red-50 text-red-700'}"
						transition:fade
					>
						{message.text}
					</div>
				{/if}

				{#if activeTab === 'general'}
					<form on:submit|preventDefault={handleGeneralSubmit} class="space-y-6">
						<h2 class="border-b pb-2 text-xl font-semibold text-gray-800">Profile Information</h2>

						<div class="grid grid-cols-1 gap-6 sm:grid-cols-2">
							<div>
								<label for="fullName" class="block text-sm font-medium text-gray-700"
									>Full Name</label
								>
								<input
									type="text"
									id="fullName"
									bind:value={formData.fullName}
									class="mt-1 block w-full rounded-md border border-gray-300 p-2 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
								/>
							</div>
							<div>
								<label for="location" class="block text-sm font-medium text-gray-700"
									>Location</label
								>
								<input
									type="text"
									id="location"
									bind:value={formData.location}
									class="mt-1 block w-full rounded-md border border-gray-300 p-2 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
								/>
							</div>
							<div>
								<label for="phone" class="block text-sm font-medium text-gray-700"
									>Phone Number</label
								>
								<input
									type="tel"
									id="phone"
									bind:value={formData.phoneNumber}
									class="mt-1 block w-full rounded-md border border-gray-300 p-2 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
								/>
							</div>
							<div>
								<label for="dob" class="block text-sm font-medium text-gray-700"
									>Date of Birth</label
								>
								<input
									type="date"
									id="dob"
									bind:value={formData.dateOfBirth}
									class="mt-1 block w-full rounded-md border border-gray-300 p-2 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
								/>
							</div>
							<div>
								<label for="gender" class="block text-sm font-medium text-gray-700">Gender</label>
								<select
									id="gender"
									bind:value={formData.gender}
									class="mt-1 block w-full rounded-md border border-gray-300 p-2 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
								>
									<option value="">Select...</option>
									<option value="male">Male</option>
									<option value="female">Female</option>
									<option value="other">Other</option>
								</select>
							</div>
						</div>

						<div>
							<label for="bio" class="block text-sm font-medium text-gray-700">Bio</label>
							<textarea
								id="bio"
								rows="3"
								bind:value={formData.bio}
								class="mt-1 block w-full rounded-md border border-gray-300 p-2 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
							></textarea>
						</div>

						<div class="flex justify-end">
							<button
								type="submit"
								disabled={isLoading}
								class="rounded-md bg-blue-600 px-4 py-2 text-white transition hover:bg-blue-700 disabled:opacity-50"
							>
								{isLoading ? 'Saving...' : 'Save Changes'}
							</button>
						</div>
					</form>
				{/if}

				{#if activeTab === 'privacy'}
					<form on:submit|preventDefault={handlePrivacySubmit} class="space-y-6">
						<h2 class="border-b pb-2 text-xl font-semibold text-gray-800">Privacy Settings</h2>

						<div class="space-y-4">
							{#each [{ key: 'default_post_privacy', label: 'Who can see your future posts?' }, { key: 'can_see_my_friends_list', label: 'Who can see your friends list?' }, { key: 'can_send_me_friend_requests', label: 'Who can send you friend requests?' }, { key: 'can_tag_me_in_posts', label: 'Who can tag you in posts?' }] as field}
								<div
									class="flex flex-col justify-between border-b border-gray-100 py-2 last:border-0 sm:flex-row sm:items-center"
								>
									<label for={field.key} class="text-sm font-medium text-gray-700 sm:w-1/2"
										>{field.label}</label
									>
									<select
										id={field.key}
										bind:value={privacyData[field.key]}
										class="mt-1 block w-full rounded-md border border-gray-300 p-2 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:mt-0 sm:w-1/2 sm:text-sm"
									>
										<option value="PUBLIC">Public</option>
										<option value="FRIENDS">Friends Only</option>
										<option value="FRIENDS_OF_FRIENDS">Friends of Friends</option>
										<option value="ONLY_ME">Only Me</option>
										<option value="EVERYONE">Everyone</option>
									</select>
								</div>
							{/each}
						</div>

						<div class="mt-6 flex justify-end">
							<button
								type="submit"
								disabled={isLoading}
								class="rounded-md bg-blue-600 px-4 py-2 text-white transition hover:bg-blue-700 disabled:opacity-50"
							>
								{isLoading ? 'Saving...' : 'Update Privacy'}
							</button>
						</div>
					</form>
				{/if}

				{#if activeTab === 'notifications'}
					<form on:submit|preventDefault={handleNotificationSubmit} class="space-y-6">
						<h2 class="border-b pb-2 text-xl font-semibold text-gray-800">
							Notification Preferences
						</h2>

						<div class="space-y-4">
							<h3 class="font-medium text-gray-900">Channels</h3>
							<!-- Email & Push Toggles -->
							{#each [{ key: 'email_notifications', label: 'Email Notifications' }, { key: 'push_notifications', label: 'Push Notifications' }] as channel}
								<div class="flex items-center justify-between">
									<span class="text-sm text-gray-700">{channel.label}</span>
									<button
										type="button"
										on:click={() =>
											(notificationData[channel.key] = !notificationData[channel.key])}
										class="{notificationData[channel.key]
											? 'bg-blue-600'
											: 'bg-gray-200'} relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
									>
										<span
											aria-hidden="true"
											class="{notificationData[channel.key]
												? 'translate-x-5'
												: 'translate-x-0'} pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
										></span>
									</button>
								</div>
							{/each}

							<h3 class="mt-6 font-medium text-gray-900">Notify Me When...</h3>
							{#each [{ key: 'notify_on_friend_request', label: 'Someone sends me a friend request' }, { key: 'notify_on_comment', label: 'Someone comments on my post' }, { key: 'notify_on_like', label: 'Someone likes my post' }, { key: 'notify_on_tag', label: 'Someone tags me' }, { key: 'notify_on_message', label: 'Someone sends me a message' }] as setting}
								<div
									class="flex items-center justify-between border-b border-gray-100 py-2 last:border-0"
								>
									<span class="text-sm text-gray-700">{setting.label}</span>
									<button
										type="button"
										on:click={() =>
											(notificationData[setting.key] = !notificationData[setting.key])}
										class="{notificationData[setting.key]
											? 'bg-blue-600'
											: 'bg-gray-200'} relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
									>
										<span
											aria-hidden="true"
											class="{notificationData[setting.key]
												? 'translate-x-5'
												: 'translate-x-0'} pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
										></span>
									</button>
								</div>
							{/each}
						</div>

						<div class="mt-6 flex justify-end">
							<button
								type="submit"
								disabled={isLoading}
								class="rounded-md bg-blue-600 px-4 py-2 text-white transition hover:bg-blue-700 disabled:opacity-50"
							>
								{isLoading ? 'Saving...' : 'Save Preferences'}
							</button>
						</div>
					</form>
				{/if}

				{#if activeTab === 'security'}
					<form on:submit|preventDefault={handleSecuritySubmit} class="space-y-6">
						<h2 class="border-b pb-2 text-xl font-semibold text-gray-800">Security Settings</h2>

						<div class="space-y-4">
							<div class="flex items-center justify-between">
								<div>
									<h3 class="font-medium text-gray-900">End-to-End Encryption</h3>
									<p class="text-sm text-gray-500">
										Enable or disable encryption for your messages. Disabling this will prevent new
										messages from being encrypted.
									</p>
								</div>
								<button
									type="button"
									on:click={() =>
										(securityData.is_encryption_enabled = !securityData.is_encryption_enabled)}
									class="{securityData.is_encryption_enabled
										? 'bg-green-600'
										: 'bg-gray-200'} relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2"
								>
									<span
										aria-hidden="true"
										class="{securityData.is_encryption_enabled
											? 'translate-x-5'
											: 'translate-x-0'} pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
									></span>
								</button>
							</div>
						</div>

						<div class="mt-6 flex justify-end">
							<button
								type="submit"
								disabled={isLoading}
								class="rounded-md bg-blue-600 px-4 py-2 text-white transition hover:bg-blue-700 disabled:opacity-50"
							>
								{isLoading ? 'Saving...' : 'Save Security Settings'}
							</button>
						</div>
					</form>
				{/if}
			</main>
		</div>
	</div>
</div>
