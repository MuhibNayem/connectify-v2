<script lang="ts">
	import { onMount } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Send, Loader2, Trash2, MoreHorizontal, MessageCircle } from '@lucide/svelte';
	import {
		getEventPosts,
		createEventPost,
		deleteEventPost,
		reactToEventPost,
		type EventPost
	} from '$lib/api';
	import { formatDistanceToNow } from 'date-fns';

	let {
		eventId,
		canPost = false,
		currentUserId
	}: {
		eventId: string;
		canPost?: boolean;
		currentUserId?: string;
	} = $props();

	let posts: EventPost[] = $state([]);
	let loading = $state(true);
	let submitting = $state(false);
	let newPostContent = $state('');
	let page = $state(1);
	let hasMore = $state(true);
	let total = $state(0);

	onMount(async () => {
		await loadPosts();
	});

	async function loadPosts() {
		loading = true;
		try {
			const response = await getEventPosts(eventId, page, 20);
			posts = response.posts || [];
			total = response.total || 0;
			hasMore = posts.length < total;
		} catch (err) {
			console.error('Failed to load posts:', err);
		} finally {
			loading = false;
		}
	}

	async function loadMore() {
		page++;
		try {
			const response = await getEventPosts(eventId, page, 20);
			posts = [...posts, ...(response.posts || [])];
			hasMore = posts.length < total;
		} catch (err) {
			console.error('Failed to load more posts:', err);
			page--;
		}
	}

	async function submitPost() {
		if (!newPostContent.trim() || submitting) return;

		submitting = true;
		try {
			const newPost = await createEventPost(eventId, newPostContent.trim());
			posts = [newPost, ...posts];
			newPostContent = '';
			total++;
		} catch (err) {
			console.error('Failed to create post:', err);
			alert('Failed to post. You may need to be an attendee to post.');
		} finally {
			submitting = false;
		}
	}

	async function handleDelete(postId: string) {
		if (!confirm('Delete this post?')) return;

		try {
			await deleteEventPost(eventId, postId);
			posts = posts.filter((p) => p.id !== postId);
			total--;
		} catch (err) {
			console.error('Failed to delete post:', err);
		}
	}

	async function handleReact(postId: string, emoji: string) {
		try {
			await reactToEventPost(eventId, postId, emoji);
			// Optimistically update the UI
			posts = posts.map((p) => {
				if (p.id === postId) {
					const existingReaction = p.reactions.find((r) => r.user.id === currentUserId);
					if (existingReaction) {
						// Update existing reaction
						return {
							...p,
							reactions: p.reactions.map((r) => (r.user.id === currentUserId ? { ...r, emoji } : r))
						};
					} else {
						// Add new reaction
						return {
							...p,
							reactions: [
								...p.reactions,
								{
									user: { id: currentUserId, username: '', avatar: '' },
									emoji,
									timestamp: new Date().toISOString()
								}
							]
						};
					}
				}
				return p;
			});
		} catch (err) {
			console.error('Failed to react:', err);
		}
	}

	const reactionEmojis = ['👍', '❤️', '😂', '😮', '😢', '😡'];

	function formatTime(dateStr: string) {
		try {
			return formatDistanceToNow(new Date(dateStr), { addSuffix: true });
		} catch {
			return '';
		}
	}
</script>

<div class="space-y-4">
	<!-- New Post Form -->
	{#if canPost}
		<div class="rounded-xl border border-white/10 bg-white/5 p-4">
			<Textarea
				placeholder="Write something about this event..."
				class="resize-none border-none bg-transparent"
				rows={3}
				bind:value={newPostContent}
			/>
			<div class="mt-3 flex justify-end">
				<Button
					size="sm"
					class="gap-2"
					disabled={!newPostContent.trim() || submitting}
					onclick={submitPost}
				>
					{#if submitting}
						<Loader2 class="h-4 w-4 animate-spin" />
					{:else}
						<Send size={16} />
					{/if}
					Post
				</Button>
			</div>
		</div>
	{/if}

	<!-- Posts List -->
	{#if loading}
		<div class="flex items-center justify-center py-8">
			<Loader2 class="animate-spin text-white" size={24} />
		</div>
	{:else if posts.length === 0}
		<div class="text-muted-foreground py-8 text-center">
			<MessageCircle class="mx-auto mb-2 opacity-50" size={32} />
			<p>No discussion posts yet</p>
			<p class="text-sm">Be the first to start the conversation!</p>
		</div>
	{:else}
		<div class="space-y-4">
			{#each posts as post}
				<div class="rounded-xl border border-white/10 bg-white/5 p-4">
					<!-- Post Header -->
					<div class="flex items-start gap-3">
						<img
							src={post.author.avatar || 'https://github.com/shadcn.png'}
							alt=""
							class="h-10 w-10 rounded-full object-cover"
						/>
						<div class="flex-1">
							<div class="flex items-center gap-2">
								<span class="font-medium">{post.author.full_name || post.author.username}</span>
								<span class="text-muted-foreground text-xs">{formatTime(post.created_at)}</span>
							</div>
							<p class="mt-2 whitespace-pre-wrap text-sm">{post.content}</p>

							<!-- Media -->
							{#if post.media_urls && post.media_urls.length > 0}
								<div class="mt-3 flex gap-2 overflow-x-auto">
									{#each post.media_urls as url}
										<img src={url} alt="" class="h-40 max-w-xs rounded-lg object-cover" />
									{/each}
								</div>
							{/if}

							<!-- Reactions -->
							<div class="mt-3 flex items-center gap-2">
								{#each reactionEmojis as emoji}
									<button
										class="rounded-lg px-2 py-1 text-sm transition-colors hover:bg-white/10 {post.reactions.some(
											(r) => r.emoji === emoji
										)
											? 'bg-white/10'
											: ''}"
										onclick={() => handleReact(post.id, emoji)}
									>
										{emoji}
										{#if post.reactions.filter((r) => r.emoji === emoji).length > 0}
											<span class="ml-1 text-xs">
												{post.reactions.filter((r) => r.emoji === emoji).length}
											</span>
										{/if}
									</button>
								{/each}
							</div>
						</div>

						<!-- Delete button (for author) -->
						{#if post.author.id === currentUserId}
							<button
								class="text-muted-foreground rounded-full p-1 transition-colors hover:text-red-400"
								onclick={() => handleDelete(post.id)}
							>
								<Trash2 size={16} />
							</button>
						{/if}
					</div>
				</div>
			{/each}

			{#if hasMore}
				<div class="text-center">
					<Button variant="ghost" onclick={loadMore}>Load More</Button>
				</div>
			{/if}
		</div>
	{/if}
</div>
