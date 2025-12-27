<script lang="ts">
	import { onMount } from 'svelte';
	import { sendWebSocketMessage } from '$lib/websocket';
	import { getGroupDetails } from '$lib/api';
	import { scale, fade } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';

	// Export value for binding
	export let value = '';

	export let onSend: (content: string, files: File[]) => Promise<void>;
	export let onTyping: (() => void) | undefined = undefined; // Add onTyping support if needed
	export let conversationId: string = ''; // make optional or default

	onMount(async () => {
		await import('emoji-picker-element');
	});

	let isSending = false;
	// Use exported value as local content
	$: content = value; // one way sync

	let files: File[] = [];
	let showEmojiPicker = false;
	let fileInput: HTMLInputElement;
	let textareaRef: HTMLTextAreaElement;

	// Mentions logic
	let showMentions = false;
	let mentionQuery = '';
	let mentionIndex = 0; // Index of list selection
	let validMembers: any[] = []; // Cache of group members
	let filteredMembers: any[] = [];
	let mentionCursorPos = -1; // Position where @ was typed

	// Reset cache when conversation changes
	$: if (conversationId) {
		validMembers = [];
		showMentions = false;
	}

	// Typing indicator logic
	let typingTimer: any;
	export let isMarketplace: boolean = false; // Add marketplace flag for typing context

	function handleTyping() {
		clearTimeout(typingTimer);
		sendWebSocketMessage('typing', {
			isTyping: true,
			conversation_id: conversationId,
			is_marketplace: isMarketplace
		});
		typingTimer = setTimeout(() => {
			sendWebSocketMessage('typing', {
				isTyping: false,
				conversation_id: conversationId,
				is_marketplace: isMarketplace
			});
		}, 2000); // Consider user as "stopped typing" after 2 seconds

		checkForMentions();
	}

	async function checkForMentions() {
		if (!conversationId.startsWith('group-')) return;

		const textarea = document.querySelector('textarea');
		if (!textarea) return;

		const cursorPosition = textarea.selectionStart;
		const textBeforeCursor = content.slice(0, cursorPosition);
		const lastAt = textBeforeCursor.lastIndexOf('@');

		if (lastAt !== -1 && (lastAt === 0 || /\s/.test(textBeforeCursor[lastAt - 1]))) {
			const query = textBeforeCursor.slice(lastAt + 1);
			if (!/\s/.test(query)) {
				// Only if no spaces after @
				mentionQuery = query;
				mentionCursorPos = lastAt;

				// Fetch members if not already cached
				if (validMembers.length === 0) {
					try {
						const groupId = conversationId.split('-')[1];
						const group = await getGroupDetails(groupId);
						// Combine creator, members, admins and deduplicate by ID
						const allMembers = new Map();
						if (group.creator) allMembers.set(group.creator.id, group.creator);
						if (group.members) group.members.forEach((m: any) => allMembers.set(m.id, m));
						if (group.admins) group.admins.forEach((m: any) => allMembers.set(m.id, m));
						validMembers = Array.from(allMembers.values());
					} catch (e) {
						console.error('Failed to fetch group info', e);
					}
				}

				// Filter members
				filteredMembers = validMembers.filter((m) =>
					m.username.toLowerCase().includes(query.toLowerCase())
				);
				showMentions = filteredMembers.length > 0;
				mentionIndex = 0; // Reset selection
			} else {
				showMentions = false;
			}
		} else {
			showMentions = false;
		}
	}

	function selectMention(member: any) {
		if (mentionCursorPos === -1) return;

		const beforeMention = value.slice(0, mentionCursorPos);
		const afterMention = value.slice(mentionCursorPos + mentionQuery.length + 1);

		value = `${beforeMention}@${member.username} ${afterMention}`;
		content = value;
		showMentions = false;

		// Reset cursor position (optional, but good UX)
		setTimeout(() => {
			const textarea = document.querySelector('textarea');
			if (textarea) {
				const newCursorPos = beforeMention.length + member.username.length + 2; // +2 for @ and space
				textarea.setSelectionRange(newCursorPos, newCursorPos);
				textarea.focus();
			}
		}, 0);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (showMentions) {
			if (e.key === 'ArrowDown') {
				e.preventDefault();
				mentionIndex = (mentionIndex + 1) % filteredMembers.length;
			} else if (e.key === 'ArrowUp') {
				e.preventDefault();
				mentionIndex = (mentionIndex - 1 + filteredMembers.length) % filteredMembers.length;
			} else if (e.key === 'Enter') {
				e.preventDefault();
				selectMention(filteredMembers[mentionIndex]);
			} else if (e.key === 'Escape') {
				showMentions = false;
			}
		} else {
			if (e.key === 'Enter' && !e.shiftKey) {
				e.preventDefault();
				handleSubmit();
			}
		}
	}

	async function handleSubmit() {
		if ((!value.trim() && files.length === 0) || isSending) return;

		isSending = true;
		try {
			await onSend(value, files);
			value = ''; // Clear bound value
			content = '';
			files = [];
			if (fileInput) fileInput.value = '';
			// Re-focus the input after send
			if (textareaRef) textareaRef.focus();
		} catch (error) {
			console.error('Failed to send message:', error);
		} finally {
			isSending = false;
			// Also focus on error case
			if (textareaRef) textareaRef.focus();
		}
	}

	function handleFileSelect(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files) {
			files = Array.from(input.files);
		}
	}

	function addEmoji(event: any) {
		value += event.detail.unicode;
		content = value;
		showEmojiPicker = false;
	}

	function toggleEmojiPicker() {
		showEmojiPicker = !showEmojiPicker;
	}

	// Close picker on outside click
	function handleOutsideClick(event: MouseEvent) {
		// Implementation omitted for brevity, logic handled by Svelte 'on:blur' or window click if needed
		// For simplicity, we just toggle.
	}
</script>

<div class="relative border-t border-gray-200 bg-white p-4">
	{#if files.length > 0}
		<div class="mb-2 flex flex-wrap gap-2 p-2">
			{#each files as file}
				<div
					class="relative flex h-16 w-16 items-center justify-center overflow-hidden rounded-lg border border-gray-200 bg-gray-100"
					transition:scale={{ duration: 200, easing: quintOut }}
				>
					{#if file.type.startsWith('image/')}
						<img
							src={URL.createObjectURL(file)}
							alt={file.name}
							class="h-full w-full object-cover"
						/>
					{:else if file.type.startsWith('video/')}
						<video src={URL.createObjectURL(file)} class="h-full w-full object-cover"></video>
						<div class="absolute inset-0 flex items-center justify-center bg-black/30">
							<svg
								xmlns="http://www.w3.org/2000/svg"
								class="h-6 w-6 text-white"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
								/>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
								/>
							</svg>
						</div>
					{:else}
						<div class="flex flex-col items-center justify-center p-1 text-center">
							<svg
								xmlns="http://www.w3.org/2000/svg"
								class="h-6 w-6 text-gray-400"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
								/>
							</svg>
							<span class="mt-1 w-full truncate text-[10px] leading-tight text-gray-600"
								>{file.name.split('.').pop()?.toUpperCase() || 'FILE'}</span
							>
						</div>
					{/if}

					<!-- Remove Button -->
					<button
						class="absolute right-0.5 top-0.5 rounded-full bg-black/50 p-0.5 text-white hover:bg-black/70"
						on:click={() => {
							files = files.filter((f) => f !== file);
							if (files.length === 0 && fileInput) fileInput.value = '';
						}}
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							class="h-3 w-3"
							viewBox="0 0 20 20"
							fill="currentColor"
						>
							<path
								fill-rule="evenodd"
								d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
								clip-rule="evenodd"
							/>
						</svg>
					</button>
				</div>
			{/each}
		</div>
	{/if}

	{#if showMentions}
		<div
			class="absolute bottom-full left-0 z-20 mb-2 max-h-48 w-64 overflow-hidden overflow-y-auto rounded-lg border border-gray-200 bg-white shadow-xl"
		>
			{#each filteredMembers as member, i}
				<button
					class="flex w-full items-center gap-2 px-4 py-2 text-left transition-colors hover:bg-gray-100 {i ===
					mentionIndex
						? 'bg-blue-50'
						: ''}"
					on:click={() => selectMention(member)}
				>
					<img
						src={member.avatar || `https://i.pravatar.cc/150?u=${member.id}`}
						alt={member.username}
						class="h-8 w-8 rounded-full"
					/>
					<div class="flex flex-col">
						<span class="text-sm font-medium text-gray-900">{member.username}</span>
						{#if member.full_name}
							<span class="text-xs text-gray-500">{member.full_name}</span>
						{/if}
					</div>
				</button>
			{/each}
		</div>
	{/if}

	{#if showEmojiPicker}
		<div class="absolute bottom-full left-10 z-10 mb-2 overflow-hidden rounded-lg shadow-lg">
			<emoji-picker on:emoji-click={addEmoji}></emoji-picker>
		</div>
	{/if}

	<form on:submit|preventDefault={handleSubmit}>
		<div class="flex items-center">
			<!-- File Attachment -->
			<input
				type="file"
				multiple
				class="hidden"
				bind:this={fileInput}
				on:change={handleFileSelect}
			/>
			<button
				type="button"
				on:click={() => fileInput.click()}
				class="mr-2 rounded-full p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700"
				title="Attach files"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-6 w-6"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13"
					/>
				</svg>
			</button>
			<button
				type="button"
				on:click={toggleEmojiPicker}
				class="mr-2 rounded-full p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700"
				title="Add emoji"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-6 w-6"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M14.828 14.828a4 4 0 01-5.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			</button>

			<textarea
				bind:this={textareaRef}
				bind:value
				disabled={isSending}
				on:input={handleTyping}
				on:keydown={handleKeydown}
				rows="1"
				class="mx-2 flex-1 resize-none rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
				placeholder="Type a message..."
			></textarea>

			<button
				type="submit"
				disabled={isSending || (!value.trim() && files.length === 0)}
				class="inline-flex cursor-pointer justify-center rounded-full p-2 text-blue-600 hover:bg-blue-100 disabled:cursor-not-allowed disabled:opacity-50 dark:text-blue-500 dark:hover:bg-gray-600"
			>
				<svg class="h-6 w-6 rotate-90" fill="currentColor" viewBox="0 0 20 20">
					<path
						d="M10.894 2.553a1 1 0 00-1.788 0l-7 14a1 1 0 001.169 1.409l5-1.428A1 1 0 009 15.571V11a1 1 0 112 0v4.571a1 1 0 00.725.962l5 1.428a1 1 0 001.17-1.408l-7-14z"
					></path>
				</svg>
				<span class="sr-only">Send message</span>
			</button>
		</div>
	</form>
</div>
