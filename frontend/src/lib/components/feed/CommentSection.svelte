<script lang="ts">
	import { onMount } from 'svelte';
	import { apiRequest } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import { websocketMessages } from '$lib/websocket';
	import Comment from './Comment.svelte';
	import UserMentionDropdown from './UserMentionDropdown.svelte';

	export let postId: string;
	let comments: any[] = [];
	let newCommentContent = '';

	// Mention-related state
	let showMentions = false;
	let mentionQuery = '';
	let mentionStartPos = -1;
	let mentionedUsers: any[] = [];

	async function fetchComments() {
		try {
			comments = await apiRequest('GET', `/posts/${postId}/comments`);
			if (!Array.isArray(comments)) {
				comments = [];
			}
		} catch (error) {
			console.error('Failed to fetch comments:', error);
		}
	}

	async function handlePostComment() {
		if (!newCommentContent.trim()) return;
		try {
			const newComment = await apiRequest('POST', '/comments', {
				post_id: postId,
				content: newCommentContent,
				mentions: mentionedUsers.map((u) => u.id)
			});
			console.log('New comment created:', newComment);
			// Ensure replies field is an array for optimistic update
			if (!newComment?.replies) {
				newComment.replies = [];
			}

			const newCommentWithAuthor = { ...newComment, author: auth.state.user };
			console.log('New comment created 2:', newComment);
			comments = [newCommentWithAuthor, ...comments];
			console.log('Comment posted successfully:', newCommentWithAuthor);
			newCommentContent = '';
			mentionedUsers = [];
		} catch (error) {
			console.log('Failed to post comment:', error);
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
		const before = newCommentContent.substring(0, mentionStartPos);
		const after = newCommentContent.substring(mentionStartPos + 1 + mentionQuery.length);

		newCommentContent = `${before}@${user.username} ${after}`;

		if (!mentionedUsers.some((u) => u.id === user.id)) {
			mentionedUsers.push(user);
		}

		showMentions = false;
	}

	onMount(() => {
		fetchComments();

		const unsubscribe = websocketMessages.subscribe((event) => {
			if (event && event.type === 'CommentCreated' && event.data.post_id === postId) {
				// Avoid duplicate comments if we just posted it ourselves
				// (Although handlePostComment adds it to the list, the WS event might come too)
				// A simple check is if it's already in the list
				if (!comments.some((c) => c.id === event.data.id)) {
					comments = [event.data, ...comments];
				}
			}
		});

		return () => {
			unsubscribe();
		};
	});
</script>

<div class="mt-3 border-t border-white/10 pt-3">
	<!-- Form to add a new comment -->
	<div class="relative">
		<div class="flex items-start space-x-2">
			<img
				src={auth.state.user?.avatar || 'https://github.com/shadcn.png'}
				alt="Your avatar"
				class="h-8 w-8 rounded-full"
			/>
			<div class="relative flex-1">
				<textarea
					bind:value={newCommentContent}
					oninput={handleInput}
					onblur={() => setTimeout(() => (showMentions = false), 150)}
					class="text-foreground placeholder:text-muted-foreground block w-full rounded-2xl border-none bg-black/5 px-4 py-2 text-sm focus:ring-0"
					placeholder="Write a comment..."
					onkeydown={(e) => {
						if (e.key === 'Enter' && !e.shiftKey && !showMentions) {
							e.preventDefault();
							handlePostComment();
						}
					}}
				></textarea>
				{#if showMentions}
					<div class="glass-card absolute bottom-full z-10 mb-1 mt-1 w-full">
						<UserMentionDropdown query={mentionQuery} onSelection={handleMentionSelection} />
					</div>
				{/if}
			</div>
		</div>
	</div>

	<!-- List of comments -->
	<div class="mt-4 space-y-2">
		{#each comments as comment (comment.id)}
			<Comment {comment} />
		{/each}
	</div>
</div>
