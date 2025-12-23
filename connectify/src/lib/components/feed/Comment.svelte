<script lang="ts">
	import { onMount } from 'svelte';
	import { formatDistanceToNow } from 'date-fns';
	import { apiRequest } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import UserMentionDropdown from './UserMentionDropdown.svelte';

	export let comment: any;
	export let isReply: boolean = false;

	let replies: any[] = [];
	let showReplyForm = false;
	let newReplyContent = '';

	// Mention-related state
	let showMentions = false;
	let mentionQuery = '';
	let mentionStartPos = -1;
	let mentionedUsers: any[] = [];

	async function fetchReplies() {
		if (isReply) return;
		try {
			replies = (await apiRequest('GET', `/comments/${comment.id}/replies`)) || [];
		} catch (error) {
			console.error('Failed to fetch replies:', error);
		}
	}

	async function handlePostReply(parentReplyId: string | null = null) {
		if (!newReplyContent.trim()) return;
		try {
			const newReply = await apiRequest('POST', `/comments/${comment.id}/replies`, {
				comment_id: comment.id, // Add comment_id to the request body
				content: newReplyContent,
				parent_reply_id: parentReplyId,
				mentions: mentionedUsers.map((u) => u.id)
			});
			const newReplyWithAuthor = { ...newReply, author: auth.state.user };
			replies = [...replies, newReplyWithAuthor];
			newReplyContent = '';
			mentionedUsers = [];
			showReplyForm = false;
		} catch (error) {
			console.error('Failed to post reply:', error);
			alert('Failed to post reply.');
		}
	}

	function handleInput(event: Event) {
		const textarea = event.target as HTMLTextAreaElement;
		const cursorPos = textarea.selectionStart;
		const textUpToCursor = textarea.value.substring(0, cursorPos);

		const lastAtPos = textUpToCursor.lastIndexOf('@');

		if (lastAtPos === -1) {
			showMentions = false;
			return;
		}

		const textAfterAt = textUpToCursor.substring(lastAtPos + 1);
		if (/\s/.test(textAfterAt)) {
			showMentions = false;
			return;
		}

		mentionStartPos = lastAtPos;
		mentionQuery = textAfterAt;
		showMentions = true;
	}

	function handleMentionSelection(user: any) {
		const before = newReplyContent.substring(0, mentionStartPos);
		const after = newReplyContent.substring(mentionStartPos + 1 + mentionQuery.length);

		newReplyContent = `${before}@${user.username} ${after}`;

		if (!mentionedUsers.some((u) => u.id === user.id)) {
			mentionedUsers.push(user);
		}

		showMentions = false;
	}

	onMount(() => {
		fetchReplies();
	});
</script>

<div class="flex items-start space-x-2">
	<img
		src={comment.author?.avatar || 'https://github.com/shadcn.png'}
		alt={comment.author?.username}
		class="rounded-full {isReply ? 'h-6 w-6' : 'h-8 w-8'}"
	/>
	<div class="flex-1">
		<div class="rounded-xl bg-gray-100 px-3 py-2">
			<a href={`/profile/${comment.author?.id}`} class="text-sm font-semibold hover:underline"
				>{comment.author?.username || '[Deleted User]'}</a
			>
			<p class="text-sm text-gray-800">{comment.content}</p>
		</div>
		<div class="flex items-center space-x-2 px-3 text-xs text-gray-500">
			{#if !isReply}
				<button
					class="font-semibold hover:underline"
					onclick={() => (showReplyForm = !showReplyForm)}>Reply</button
				>
				<span>·</span>
			{/if}
			<span>{formatDistanceToNow(new Date(comment.created_at), { addSuffix: true })}</span>
		</div>

		{#if showReplyForm}
			<div class="relative mt-2 flex items-start space-x-2">
				<img
					src={auth.state.user?.avatar || 'https://github.com/shadcn.png'}
					alt="Your avatar"
					class="h-6 w-6 rounded-full"
				/>
				<div class="relative flex-1">
					<textarea
						bind:value={newReplyContent}
						class="block w-full rounded-2xl border-transparent bg-gray-100 px-4 py-2 text-sm focus:border-transparent focus:ring-0"
						placeholder="Write a reply..."
						oninput={handleInput}
						onblur={() => setTimeout(() => (showMentions = false), 150)}
						onkeydown={(e) => {
							if (e.key === 'Enter' && !e.shiftKey && !showMentions) {
								e.preventDefault();
								handlePostReply();
							}
						}}
					></textarea>
					{#if showMentions}
						<div class="absolute bottom-full z-10 mb-1 mt-1 w-full">
							<UserMentionDropdown query={mentionQuery} onSelection={handleMentionSelection} />
						</div>
					{/if}
				</div>
			</div>
		{/if}

		<!-- Replies -->
		<div class="mt-2 space-y-2">
			{#each replies as reply (reply.id)}
				<svelte:self comment={reply} isReply={true} />
			{/each}
		</div>
	</div>
</div>
