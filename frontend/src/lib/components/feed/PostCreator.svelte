<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import { apiRequest, uploadFiles } from '$lib/api';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Button } from '$lib/components/ui/button';
	import { CustomSelect } from '$lib/components/ui/custom-select';
	import { X, Image as ImageIcon, Smile, MapPin, Tag, UserPlus, Video } from '@lucide/svelte';
	import UserMentionDropdown from './UserMentionDropdown.svelte';
	import { auth } from '$lib/stores/auth.svelte';

	const dispatch = createEventDispatcher();

	type MediaItem = {
		file: File;
		previewUrl: string;
		type: 'image' | 'video';
	};

	type User = {
		id: string;
		username: string;
		avatar?: string;
		first_name?: string;
		full_name?: string;
	};

	let { communityId = undefined } = $props<{ communityId?: string }>();

	let postContent = $state('');
	let mediaItems = $state<MediaItem[]>([]);
	let privacy: 'PUBLIC' | 'FRIENDS' | 'ONLY_ME' = $state('PUBLIC');
	let submitting = $state(false);
	let fileInput: HTMLInputElement;
	let isExpanded = $state(false);

	// Rich Features State
	let location = $state('');
	let showLocationInput = $state(false);
	let showEmojiPicker = $state(false);
	let showUserTagger = $state(false);
	let taggedUsers = $state<User[]>([]);
	let userSearchQuery = $state('');

	let emojiPickerContainer: HTMLElement;
	let emojiToggleButton: HTMLElement;

	// Computed or derived if needed (none really)

	onMount(async () => {
		if (typeof window !== 'undefined') {
			await import('emoji-picker-element');
		}
	});

	function handleFileSelect(event: Event) {
		const input = event.target as HTMLInputElement;
		if (input.files) {
			addFiles(Array.from(input.files));
		}
		input.value = '';
	}

	function addFiles(files: File[]) {
		files.forEach((file) => {
			const match = file.type.match(/^(image|video)\//);
			if (match) {
				const type = match[1] as 'image' | 'video';
				const previewUrl = URL.createObjectURL(file);
				mediaItems = [...mediaItems, { file, previewUrl, type }];
			} else {
				alert(`File type ${file.type} not supported`);
			}
		});
	}

	function removeMedia(index: number) {
		URL.revokeObjectURL(mediaItems[index].previewUrl);
		mediaItems = mediaItems.filter((_, i) => i !== index);
	}

	function onEmojiSelect(event: any) {
		// emoji-picker-element emits 'emoji-click' with detail: { unicode: '...', ... }
		if (event.detail && event.detail.unicode) {
			postContent += event.detail.unicode;
		}
		showEmojiPicker = false;
	}

	function toggleEmojiPicker() {
		showEmojiPicker = !showEmojiPicker;
	}

	function setupEmojiPicker(node: HTMLElement) {
		const handleEmojiClick = (event: any) => onEmojiSelect(event);
		node.addEventListener('emoji-click', handleEmojiClick);
		return {
			destroy() {
				node.removeEventListener('emoji-click', handleEmojiClick);
			}
		};
	}

	function handleWindowClick(event: MouseEvent) {
		if (!showEmojiPicker) return;

		if (
			emojiPickerContainer &&
			!emojiPickerContainer.contains(event.target as Node) &&
			emojiToggleButton &&
			!emojiToggleButton.contains(event.target as Node)
		) {
			showEmojiPicker = false;
		}
	}

	function toggleLocation() {
		showLocationInput = !showLocationInput;
	}

	function toggleUserTagger() {
		showUserTagger = !showUserTagger;
		if (!showUserTagger) {
			userSearchQuery = '';
		}
	}

	function addUserTag(user: User) {
		if (!taggedUsers.find((u) => u.id === user.id)) {
			taggedUsers = [...taggedUsers, user];
		}
		userSearchQuery = '';
		showUserTagger = false;
	}

	function removeUserTag(userId: string) {
		taggedUsers = taggedUsers.filter((u) => u.id !== userId);
	}

	// Inline Mention Logic
	let showInlineMentions = $state(false);
	let mentionQuery = $state('');
	let mentionStartPos = -1;

	function handleTextareaInput(event: Event) {
		const textarea = event.target as HTMLTextAreaElement;
		const cursorPos = textarea.selectionStart;
		const textUpToCursor = textarea.value.substring(0, cursorPos);

		const lastAtPos = textUpToCursor.lastIndexOf('@');

		if (lastAtPos === -1) {
			showInlineMentions = false;
			return;
		}

		const textAfterAt = textUpToCursor.substring(lastAtPos + 1);
		if (/\s/.test(textAfterAt)) {
			showInlineMentions = false;
			return;
		}

		mentionStartPos = lastAtPos;
		mentionQuery = textAfterAt;
		showInlineMentions = true;
	}

	function handleInlineMentionSelection(user: User) {
		const before = postContent.substring(0, mentionStartPos);
		const after = postContent.substring(mentionStartPos + 1 + mentionQuery.length);

		postContent = `${before}@${user.username} ${after}`;

		if (!taggedUsers.find((u) => u.id === user.id)) {
			taggedUsers = [...taggedUsers, user];
		}

		showInlineMentions = false;
	}

	function resetForm() {
		mediaItems.forEach((item) => URL.revokeObjectURL(item.previewUrl));
		postContent = '';
		mediaItems = [];
		location = '';
		taggedUsers = [];
		showLocationInput = false;
		showUserTagger = false;
		privacy = 'PUBLIC';
		isExpanded = false;
	}

	async function handleSubmit() {
		if (!postContent.trim() && mediaItems.length === 0) {
			alert('Post content or media cannot be empty.');
			return;
		}

		submitting = true;

		try {
			let uploadedMedia: { url: string; type: string }[] = [];
			if (mediaItems.length > 0) {
				uploadedMedia = await uploadFiles(mediaItems.map((item) => item.file));
			}

			const payload: any = {
				content: postContent.trim(),
				privacy: privacy,
				location: location,
				community_id: communityId,
				media: uploadedMedia,
				mentions: taggedUsers.map((u) => u.id)
			};

			const newPost = await apiRequest('POST', '/posts', payload);

			if (!newPost.comments) {
				newPost.comments = [];
			}

			dispatch('postCreated', newPost);
			resetForm();
		} catch (err: any) {
			console.error('Create post error:', err);
			alert(err.message || 'Failed to create post.');
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:window on:click={handleWindowClick} />

<div class="glass-card bg-card mb-4 rounded-xl border border-white/5 px-4 pb-2 pt-4 shadow-sm">
	{#if !isExpanded}
		<!-- Collapsed State (Facebook Like) -->
		<div class="mb-3 flex items-center gap-3">
			<a href={`/profile/${auth.state.user?.id}`}>
				{#if auth.state.user?.avatar}
					<img
						src={auth.state.user?.avatar}
						alt="Avatar"
						class="h-10 w-10 rounded-full object-cover"
					/>
				{:else}
					<div class="bg-secondary flex h-10 w-10 items-center justify-center rounded-full">👤</div>
				{/if}
			</a>
			<button
				class="bg-secondary/50 hover:bg-secondary/70 text-muted-foreground h-10 flex-grow rounded-full px-4 text-left transition-colors"
				onclick={() => (isExpanded = true)}
			>
				What's on your mind, {(auth.state.user as any)?.first_name || auth.state.user?.username}?
			</button>
		</div>

		<hr class="my-2 border-white/10" />

		<div class="flex items-center justify-between px-2">
			<!-- Mock Live Video -->
			<button
				class="hover:bg-secondary/50 text-muted-foreground flex flex-1 items-center justify-center gap-2 rounded-lg p-2 font-medium transition-colors"
			>
				<Video size={24} class="text-red-500" />
				<span>Live Video</span>
			</button>

			<button
				class="hover:bg-secondary/50 text-muted-foreground flex flex-1 items-center justify-center gap-2 rounded-lg p-2 font-medium transition-colors"
				onclick={() => {
					isExpanded = true;
					setTimeout(() => fileInput.click(), 100);
				}}
			>
				<ImageIcon size={24} class="text-green-500" />
				<span>Photo/video</span>
			</button>

			<button
				class="hover:bg-secondary/50 text-muted-foreground mobile-hidden flex flex-1 items-center justify-center gap-2 rounded-lg p-2 font-medium transition-colors"
				onclick={() => {
					isExpanded = true;
					setTimeout(() => toggleEmojiPicker(), 100);
				}}
			>
				<Smile size={24} class="text-yellow-500" />
				<span>Feeling/activity</span>
			</button>
		</div>
	{:else}
		<!-- Expanded State -->
		<div class="relative">
			<!-- Header -->
			<div class="mb-4 flex items-center justify-between border-b border-white/10 pb-2">
				<h3 class="w-full text-center text-lg font-semibold">Create Post</h3>
				<button
					class="hover:bg-secondary/50 text-muted-foreground absolute right-0 top-0 rounded-full p-2"
					onclick={resetForm}
				>
					<X size={20} />
				</button>
			</div>

			<!-- User Info -->
			<div class="mb-2 flex items-center gap-3">
				{#if auth.state.user?.avatar}
					<img
						src={auth.state.user?.avatar}
						alt="Avatar"
						class="h-10 w-10 rounded-full object-cover"
					/>
				{:else}
					<div class="bg-secondary flex h-10 w-10 items-center justify-center rounded-full">👤</div>
				{/if}
				<div>
					<p class="font-semibold">
						{(auth.state.user as any)?.full_name || auth.state.user?.username}
					</p>
					<!-- Privacy & Submit -->
					<div class="flex items-center space-x-3">
						<CustomSelect
							bind:value={privacy}
							options={[
								{ value: 'PUBLIC', label: 'Public' },
								{ value: 'FRIENDS', label: 'Friends' },
								{ value: 'ONLY_ME', label: 'Only Me' }
							]}
							placeholder="Privacy"
							disabled={submitting}
							style="w-[90px]"
							triggerClass="bg-secondary/40 hover:bg-secondary/60 h-6 px-2 rounded-md text-xs border-none"
						/>
					</div>
				</div>
			</div>

			<!-- Input -->
			<div class="relative mb-2 min-h-[80px]">
				<Textarea
					placeholder={`What's on your mind, ${(auth.state.user as any)?.first_name || auth.state.user?.username}?`}
					bind:value={postContent}
					oninput={handleTextareaInput}
					rows={3}
					class="placeholder:text-muted-foreground/50 w-full resize-none border-none bg-transparent p-0 text-xl focus-visible:ring-0"
					disabled={submitting}
					autofocus
				/>

				<!-- Emoji Picker -->
				{#if showEmojiPicker}
					<div
						class="glass-card absolute right-0 top-10 z-50 overflow-hidden rounded-lg border border-white/10"
						bind:this={emojiPickerContainer}
					>
						<emoji-picker use:setupEmojiPicker class="light"></emoji-picker>
					</div>
				{/if}

				<!-- Mentions -->
				{#if showInlineMentions}
					<div class="absolute left-0 top-full z-50 mt-1 w-full" style="max-width: 300px;">
						<UserMentionDropdown query={mentionQuery} onSelection={handleInlineMentionSelection} />
					</div>
				{/if}
			</div>

			<!-- Media Previews (Reused) -->
			{#if mediaItems.length > 0}
				<div class="mb-4 overflow-x-auto rounded-lg border border-white/10 bg-black/20 p-2">
					<div class="flex w-max space-x-3">
						{#each mediaItems as item, index}
							<div class="relative h-32 w-32 flex-shrink-0 overflow-hidden rounded-lg">
								{#if item.type === 'image'}
									<img src={item.previewUrl} alt="Preview" class="h-full w-full object-cover" />
								{:else}
									<video src={item.previewUrl} class="h-full w-full object-cover" controls></video>
								{/if}
								<button
									onclick={() => removeMedia(index)}
									class="absolute right-1 top-1 rounded-full bg-black/70 p-1 text-white hover:bg-black"
								>
									<X size={14} />
								</button>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Feature Inputs (Location / User Search) -->
			{#if showLocationInput}
				<div
					class="glass-panel animate-in fade-in slide-in-from-top-1 mb-2 flex items-center space-x-2 rounded-lg p-2"
				>
					<MapPin size={18} class="text-red-500" />
					<input
						type="text"
						placeholder="Where are you?"
						bind:value={location}
						class="text-foreground w-full border-none bg-transparent text-sm focus:ring-0"
						autofocus
					/>
					<button onclick={() => (showLocationInput = false)}><X size={16} /></button>
				</div>
			{/if}

			{#if showUserTagger}
				<div
					class="glass-panel animate-in fade-in slide-in-from-top-1 relative mb-2 rounded-lg p-2"
				>
					<div class="flex items-center space-x-2">
						<UserPlus size={18} class="text-blue-500" />
						<input
							type="text"
							placeholder="Search for friends to tag..."
							bind:value={userSearchQuery}
							class="text-foreground w-full border-none bg-transparent text-sm focus:ring-0"
							autofocus
						/>
						<button onclick={() => (showUserTagger = false)}><X size={16} /></button>
					</div>
					{#if userSearchQuery}
						<div class="absolute left-0 top-full z-10 mt-1 w-full">
							<UserMentionDropdown query={userSearchQuery} onSelection={addUserTag} />
						</div>
					{/if}
				</div>
			{/if}

			<!-- Add to Your Post -->
			<div
				class="bg-background/50 mb-4 flex items-center justify-between rounded-lg border border-white/10 p-3 shadow-sm"
			>
				<p class="text-foreground text-sm font-semibold">Add to your post</p>

				<input
					type="file"
					multiple
					accept="image/*,video/*"
					class="hidden"
					bind:this={fileInput}
					onchange={handleFileSelect}
				/>

				<div class="flex items-center gap-1">
					<Button
						variant="ghost"
						size="icon"
						class="rounded-full text-green-400 transition-all hover:scale-110 hover:bg-green-500/10"
						title="Photo/Video"
						onclick={() => fileInput.click()}
					>
						<ImageIcon size={22} />
					</Button>
					<Button
						variant="ghost"
						size="icon"
						class="rounded-full text-blue-400 transition-all hover:scale-110 hover:bg-blue-500/10"
						title="Tag Friends"
						onclick={toggleUserTagger}
					>
						<Tag size={22} />
					</Button>
					<span bind:this={emojiToggleButton}>
						<Button
							variant="ghost"
							size="icon"
							class="rounded-full text-yellow-400 transition-all hover:scale-110 hover:bg-yellow-500/10"
							title="Feeling/Activity"
							onclick={toggleEmojiPicker}
						>
							<Smile size={22} />
						</Button>
					</span>
					<Button
						variant="ghost"
						size="icon"
						class="rounded-full text-red-400 transition-all hover:scale-110 hover:bg-red-500/10"
						title="Check in"
						onclick={toggleLocation}
					>
						<MapPin size={22} />
					</Button>
				</div>
			</div>

			<!-- Submit Button -->
			<Button
				onclick={handleSubmit}
				disabled={submitting || (!postContent.trim() && mediaItems.length === 0)}
				class="mt-4 w-full font-semibold"
			>
				{submitting ? 'Posting...' : 'Post'}
			</Button>
		</div>
	{/if}
</div>
