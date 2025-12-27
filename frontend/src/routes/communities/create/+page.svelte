<script lang="ts">
	import { createCommunity, type CreateCommunityRequest } from '$lib/api';
	import { goto } from '$app/navigation';
	import { ArrowLeft, Upload, Shield, Globe, Lock, CheckCircle } from '@lucide/svelte';

	let loading = false;
	let error = '';

	let formData: CreateCommunityRequest = {
		name: '',
		description: '',
		category: 'general',
		privacy: 'public',
		require_post_approval: false,
		require_join_approval: false
	};

	async function handleSubmit() {
		loading = true;
		error = '';
		try {
			const res = await createCommunity(formData);
			goto(`/communities/${res.id}`);
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	}
</script>

<div class="container mx-auto max-w-2xl px-4 py-8">
	<button
		on:click={() => history.back()}
		class="mb-6 flex items-center gap-2 text-gray-500 transition-colors hover:text-gray-900 dark:hover:text-white"
	>
		<ArrowLeft class="h-5 w-5" />
		Back
	</button>

	<div class="glass-panel rounded-3xl p-8">
		<h1
			class="mb-2 bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-3xl font-bold text-transparent"
		>
			Create Community
		</h1>
		<p class="mb-8 text-gray-500 dark:text-gray-400">
			Build a space for people to connect and share.
		</p>

		{#if error}
			<div class="mb-6 rounded-xl bg-red-50 p-4 text-red-600 dark:bg-red-900/20 dark:text-red-400">
				{error}
			</div>
		{/if}

		<form on:submit|preventDefault={handleSubmit} class="space-y-6">
			<!-- Name -->
			<div>
				<label for="name" class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
					>Community Name</label
				>
				<input
					bind:value={formData.name}
					type="text"
					id="name"
					required
					placeholder="e.g. Hiking Enthusiasts"
					class="w-full rounded-xl border-transparent bg-gray-50 px-4 py-3 font-medium text-gray-900 placeholder-gray-400 transition-all focus:border-blue-500 focus:bg-white focus:ring-0 dark:bg-gray-800/50 dark:text-white dark:focus:bg-gray-800"
				/>
			</div>

			<!-- Description -->
			<div>
				<label
					for="description"
					class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">Description</label
				>
				<textarea
					bind:value={formData.description}
					id="description"
					rows="3"
					required
					placeholder="What is this community about?"
					class="w-full resize-none rounded-xl border-transparent bg-gray-50 px-4 py-3 font-medium text-gray-900 placeholder-gray-400 transition-all focus:border-blue-500 focus:bg-white focus:ring-0 dark:bg-gray-800/50 dark:text-white dark:focus:bg-gray-800"
				></textarea>
			</div>

			<!-- Category -->
			<div>
				<label
					for="category"
					class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">Category</label
				>
				<select
					bind:value={formData.category}
					id="category"
					class="w-full rounded-xl border-transparent bg-gray-50 px-4 py-3 font-medium text-gray-900 transition-all focus:border-blue-500 focus:bg-white focus:ring-0 dark:bg-gray-800/50 dark:text-white dark:focus:bg-gray-800"
				>
					<option value="general">General</option>
					<option value="technology">Technology</option>
					<option value="hobbies">Hobbies</option>
					<option value="sports">Sports</option>
					<option value="music">Music</option>
					<option value="education">Education</option>
				</select>
			</div>

			<!-- Privacy -->
			<div class="space-y-3">
				<label class="block text-sm font-medium text-gray-700 dark:text-gray-300">Privacy</label>

				<label
					class="flex cursor-pointer items-start gap-3 rounded-xl border-2 p-4 transition-all {formData.privacy ===
					'public'
						? 'border-blue-500 bg-blue-50/50 dark:bg-blue-900/10'
						: 'border-transparent bg-gray-50 hover:bg-gray-100 dark:bg-gray-800/50 dark:hover:bg-gray-800'}"
				>
					<input
						type="radio"
						value="public"
						bind:group={formData.privacy}
						class="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500"
					/>
					<div class="flex-1">
						<div class="flex items-center gap-2 font-medium text-gray-900 dark:text-white">
							<Globe class="h-4 w-4" /> Public
						</div>
						<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
							Anyone can see who's in the community and what they post.
						</p>
					</div>
				</label>

				<label
					class="flex cursor-pointer items-start gap-3 rounded-xl border-2 p-4 transition-all {formData.privacy ===
					'closed'
						? 'border-blue-500 bg-blue-50/50 dark:bg-blue-900/10'
						: 'border-transparent bg-gray-50 hover:bg-gray-100 dark:bg-gray-800/50 dark:hover:bg-gray-800'}"
				>
					<input
						type="radio"
						value="closed"
						bind:group={formData.privacy}
						class="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500"
					/>
					<div class="flex-1">
						<div class="flex items-center gap-2 font-medium text-gray-900 dark:text-white">
							<Lock class="h-4 w-4" /> Closed
						</div>
						<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
							Anyone can find the community, but only members can see posts.
						</p>
					</div>
				</label>

				<label
					class="flex cursor-pointer items-start gap-3 rounded-xl border-2 p-4 transition-all {formData.privacy ===
					'secret'
						? 'border-blue-500 bg-blue-50/50 dark:bg-blue-900/10'
						: 'border-transparent bg-gray-50 hover:bg-gray-100 dark:bg-gray-800/50 dark:hover:bg-gray-800'}"
				>
					<input
						type="radio"
						value="secret"
						bind:group={formData.privacy}
						class="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500"
					/>
					<div class="flex-1">
						<div class="flex items-center gap-2 font-medium text-gray-900 dark:text-white">
							<Shield class="h-4 w-4" /> Secret
						</div>
						<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
							Only members can find this community and see posts.
						</p>
					</div>
				</label>
			</div>

			<hr class="border-gray-200 dark:border-gray-700/50" />

			<!-- Settings -->
			<div class="space-y-4">
				<label class="flex cursor-pointer items-center gap-3">
					<input
						type="checkbox"
						bind:checked={formData.require_join_approval}
						class="h-5 w-5 rounded border-gray-300 bg-gray-50 text-blue-600 transition focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-800"
					/>
					<span class="select-none text-gray-700 dark:text-gray-300">Require approval to join</span>
				</label>

				<label class="flex cursor-pointer items-center gap-3">
					<input
						type="checkbox"
						bind:checked={formData.require_post_approval}
						class="h-5 w-5 rounded border-gray-300 bg-gray-50 text-blue-600 transition focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-800"
					/>
					<span class="select-none text-gray-700 dark:text-gray-300"
						>Require approval for posts</span
					>
				</label>
			</div>

			<div class="pt-4">
				<button
					type="submit"
					disabled={loading}
					class="flex w-full items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-blue-600 to-blue-700 py-4 text-lg font-bold text-white shadow-lg transition-all hover:from-blue-700 hover:to-blue-800 hover:shadow-blue-500/30 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50"
				>
					{#if loading}
						Creating...
					{:else}
						Create Community
					{/if}
				</button>
			</div>
		</form>
	</div>
</div>
