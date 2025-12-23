<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { apiRequest, updatePost } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import { onMount, createEventDispatcher } from 'svelte';
	import { formatDistanceToNow } from 'date-fns';
	import { websocketMessages } from '$lib/websocket';
	import CommentSection from './CommentSection.svelte';
	import { goto } from '$app/navigation';
	import { Heart, MessageSquare, Share2, MoreHorizontal } from '@lucide/svelte';

	// Use named imports for better safety
	import {
		Root as DropdownMenuRoot,
		Trigger as DropdownMenuTrigger,
		Content as DropdownMenuContent,
		Item as DropdownMenuItem
	} from '$lib/components/ui/dropdown-menu';

	const dispatch = createEventDispatcher();

	export let post: {
		id: string;
		user_id: string;
		author: {
			id: string;
			username: string;
			avatar?: string;
			full_name?: string;
		};
		content: string;
		media?: { url: string; type: string }[];
		location?: string;
		privacy: string;
		comments: any[];
		mentions: string[]; // List of IDs
		mentioned_users?: { id: string; username: string }[];
		specific_reaction_counts: { [key: string]: number };
		hashtags: string[];
		created_at: string;
		updated_at: string;
		total_reactions: number;
		total_comments: number;
	};

	export let isDetailedView = false;

	let showComments = isDetailedView;

	// Reactive user ID from the new auth store
	$: currentUserId = auth.state.user?.id;

	// Local state copy to avoid mutating props directly (Svelte 5 ownership rules)
	let localReactions = 0;
	let localComments = 0;
	let localContent = '';
	let localUpdatedAt = '';
	let localSpecificReactions: { [key: string]: number } = {};

	// Sync local state with prop on changes
	$: {
		localReactions = post?.total_reactions || 0;
		localComments = post?.total_comments || 0;
		localContent = post?.content || '';
		localUpdatedAt = post?.updated_at || '';
		localSpecificReactions = post?.specific_reaction_counts || {};
	}

	// Defensive defaults using local state
	$: safePost = {
		...post,
		total_reactions: localReactions,
		total_comments: localComments,
		content: localContent,
		updated_at: localUpdatedAt,
		specific_reaction_counts: localSpecificReactions,
		media: post?.media || [],
		mentioned_users: post?.mentioned_users || []
	};

	let userReactionId: string | null = null;
	let userReactionType: string | null = null;
	$: isLikedByCurrentUser = userReactionId !== null;

	// Edit mode state
	let isEditing = false;
	let editedContent = '';
	let isSaving = false;

	onMount(() => {
		(async () => {
			if (currentUserId && safePost.id) {
				try {
					const reactions = await apiRequest('GET', `/posts/${safePost.id}/reactions`);
					const userReaction = reactions?.find(
						(r: any) => r.user_id === currentUserId && r.type === 'LIKE'
					);
					if (userReaction) {
						userReactionId = userReaction.id;
						userReactionType = userReaction.type;
					} else {
						userReactionId = null;
						userReactionType = null;
					}
				} catch (e) {
					console.error('Failed to fetch reactions for post:', e);
				}
			}
		})();

		const unsubscribe = websocketMessages.subscribe((event) => {
			if (!event?.data) return;

			// Handle ReactionCreated
			if (event.type === 'ReactionCreated' && event.data.target_id === safePost.id) {
				if (event.data.user_id != currentUserId) localReactions += 1;

				if (event.data.user_id === currentUserId && event.data.type === 'LIKE') {
					userReactionId = event.data.id;
				}
			}

			// Handle ReactionDeleted
			if (event.type === 'ReactionDeleted' && event.data.target_id === safePost.id) {
				if (event.data.user_id != currentUserId) localReactions = Math.max(0, localReactions - 1);

				if (event.data.user_id === currentUserId && event.data.type === 'LIKE') {
					userReactionId = null;
				}
			}

			// Handle CommentCreated
			if (event.type === 'CommentCreated' && event.data.post_id === safePost.id) {
				localComments += 1;
			}

			// Handle CommentDeleted
			if (event.type === 'CommentDeleted' && event.data.post_id === safePost.id) {
				localComments = Math.max(0, localComments - 1);
			}
		});

		return () => {
			unsubscribe();
		};
	});

	async function handleLike() {
		if (!currentUserId) {
			alert('Please log in to react.');
			return;
		}

		const type = 'LIKE';

		try {
			if (isLikedByCurrentUser) {
				await apiRequest(
					'DELETE',
					`/reactions/${userReactionId}?targetId=${safePost.id}&targetType=post`
				);
				userReactionId = null;
				localReactions--;
				if (localSpecificReactions && localSpecificReactions[userReactionType!]) {
					localSpecificReactions[userReactionType!]--;
				}
				userReactionType = null;
			} else {
				// Add reaction
				const newReaction = await apiRequest('POST', '/reactions', {
					target_id: safePost.id,
					target_type: 'post',
					type: type
				});
				userReactionId = newReaction.id;
				localReactions += 1;
				userReactionType = type;
				if (localSpecificReactions) {
					localSpecificReactions[type] = (localSpecificReactions[type] || 0) + 1;
				}
			}
		} catch (e: any) {
			alert(`Failed to toggle like: ${e.message}`);
			console.error('Toggle like error:', e);
		}
	}

	function handleComment() {
		showComments = !showComments;
	}

	function handleShare() {
		alert(`Sharing post by ${safePost?.author?.username}`);
	}

	function handleNavigate() {
		if (!isDetailedView) {
			goto(`/posts/${safePost.id}`);
		}
	}

	function handleMediaClick(index: number, isOverlay: boolean = false) {
		if (isOverlay || isDetailedView) {
			handleNavigate();
		} else {
			dispatch('viewMedia', { media: safePost.media, index });
		}
	}

	function startEditing() {
		isEditing = true;
		editedContent = post.content || '';
	}

	async function saveEdit() {
		if (!editedContent.trim()) return;
		isSaving = true;
		try {
			const updated = await updatePost(safePost.id, { content: editedContent });
			localContent = updated.content;
			localUpdatedAt = updated.updated_at;
			isEditing = false;
		} catch (err) {
			console.error('Failed to update post:', err);
			alert('Failed to update post');
		} finally {
			isSaving = false;
		}
	}

	function cancelEdit() {
		isEditing = false;
		editedContent = '';
	}

	function handleDelete() {
		if (!confirm('Are you sure you want to delete this post?')) return;

		apiRequest('DELETE', `/posts/${safePost.id}`)
			.then(() => {
				dispatch('postDeleted', safePost);
			})
			.catch((err) => {
				alert('Failed to delete post: ' + err.message);
			});
	}

	function parseContent(content: string) {
		if (!content) return '';

		return content.replace(/@(\w+)/g, (match, username) => {
			const mentionedUsers = safePost.mentioned_users || [];
			const user = mentionedUsers.find((u) => u.username === username);

			if (user) {
				return `<a href="/profile/${user.id}" class="text-blue-600 font-semibold">${username}</a>`;
			}
			return match;
		});
	}

	function getFormattedDate(dateStr: string) {
		try {
			if (!dateStr) return '';
			return formatDistanceToNow(new Date(dateStr), { addSuffix: true });
		} catch (e) {
			console.error('Invalid date', dateStr, e);
			return 'just now';
		}
	}
</script>

{#if safePost && safePost.author}
	<div class="glass-card mx-auto w-full max-w-2xl space-y-3 p-4">
		<div class="flex items-center space-x-3">
			<Avatar class="h-10 w-10">
				<AvatarImage
					src={safePost.author.avatar || 'https://github.com/shadcn.png'}
					alt={safePost?.author?.username}
				/>
				<AvatarFallback>{safePost.author?.username?.charAt(0).toUpperCase()}</AvatarFallback>
			</Avatar>
			<div class="flex-1">
				<div class="flex items-center justify-between">
					<div class="flex flex-col">
						<div class="flex items-center space-x-1">
							<p class="text-foreground font-semibold">
								{safePost.author.username}
							</p>
							{#if safePost.mentioned_users && safePost.mentioned_users.length > 0}
								<span class="text-muted-foreground font-normal">with</span>
								<span class="text-foreground font-medium">
									{safePost.mentioned_users.length} people
								</span>
							{:else if safePost.mentions && safePost.mentions.length > 0}
								<span class="text-muted-foreground font-normal">with</span>
								<span class="text-foreground font-medium">
									{safePost.mentions.length} people
								</span>
							{/if}
							{#if safePost.location}
								<span class="text-muted-foreground font-normal">is at</span>
								<span class="text-foreground font-medium">{safePost.location}</span>
							{/if}
						</div>
						<p class="text-muted-foreground text-xs">
							{getFormattedDate(safePost.created_at)} •
							<span class="capitalize"
								>{safePost.privacy
									? safePost.privacy.replace('_', ' ').toLowerCase()
									: 'public'}</span
							>
						</p>
					</div>
					{#if currentUserId === safePost.author.id}
						<DropdownMenuRoot>
							<DropdownMenuTrigger>
								<Button variant="ghost" size="icon" class="h-8 w-8 rounded-full">
									<MoreHorizontal size={18} />
								</Button>
							</DropdownMenuTrigger>
							<DropdownMenuContent align="end">
								<DropdownMenuItem onclick={startEditing}>Edit</DropdownMenuItem>
								<DropdownMenuItem onclick={handleDelete}>
									<span class="text-destructive">Delete</span>
								</DropdownMenuItem>
							</DropdownMenuContent>
						</DropdownMenuRoot>
					{/if}
				</div>
			</div>
		</div>

		<div class="text-foreground/90 leading-relaxed">
			{#if isEditing}
				<textarea
					bind:value={editedContent}
					class="border-border bg-background focus:ring-primary min-h-[100px] w-full resize-none rounded-lg border p-3 focus:outline-none focus:ring-2"
					placeholder="What's on your mind?"
				></textarea>
				<div class="mt-2 flex gap-2">
					<Button size="sm" onclick={saveEdit} disabled={isSaving || !editedContent.trim()}>
						{isSaving ? 'Saving...' : 'Save'}
					</Button>
					<Button size="sm" variant="outline" onclick={cancelEdit} disabled={isSaving}>
						Cancel
					</Button>
				</div>
			{:else}
				{#if !isDetailedView && safePost.content.length > 200}
					<p>
						{@html parseContent(safePost.content.slice(0, 200))}...
						<button class="text-primary font-semibold hover:underline" onclick={handleNavigate}>
							See more
						</button>
					</p>
				{:else}
					<p>{@html parseContent(safePost.content)}</p>
				{/if}
				{#if post.created_at !== post.updated_at}
					<span class="text-muted-foreground text-xs italic">• Edited</span>
				{/if}
			{/if}
		</div>

		{#if safePost.media && safePost.media.length > 0}
			{#if isDetailedView}
				<div class="mt-3 space-y-4">
					{#each safePost.media as item, i}
						<div
							class="w-full cursor-pointer overflow-hidden rounded-lg bg-black/5"
							onclick={() => handleMediaClick(i)}
						>
							{#if item.type === 'image' || item.type?.startsWith('image')}
								<img src={item.url} alt="Post media" class="w-full object-contain" />
							{:else if item.type === 'video' || item.type?.startsWith('video')}
								<video src={item.url} controls class="w-full"></video>
							{/if}
						</div>
					{/each}
				</div>
			{:else}
				{@const mediaCount = safePost.media.length}
				{@const displayMedia = safePost.media.slice(0, 4)}
				{@const remainingCount = mediaCount > 4 ? mediaCount - 3 : 0}
				<div
					class={`mt-3 grid gap-1 overflow-hidden rounded-xl ${
						mediaCount === 1
							? 'h-[300px] grid-cols-1'
							: mediaCount === 2
								? 'h-[250px] grid-cols-2'
								: mediaCount === 3
									? 'h-[400px] grid-rows-2'
									: 'h-[400px] grid-cols-2 grid-rows-2' // 4 or more
					}`}
				>
					{#each displayMedia as item, i}
						{@const isLastItem = i === 3}
						{@const isOverlayNeeded = mediaCount > 4 && isLastItem}

						<div
							class={`relative cursor-pointer overflow-hidden bg-black/5 ${
								mediaCount === 3 && i === 0 ? 'col-span-2 row-span-1' : ''
							} ${
								/* Image/Video sizing */
								'h-full w-full'
							}`}
							onclick={() => handleMediaClick(i, isOverlayNeeded)}
							role="button"
							tabindex="0"
							onkeydown={(e) => e.key === 'Enter' && handleMediaClick(i, isOverlayNeeded)}
						>
							{#if item.type === 'image' || item.type?.startsWith('image')}
								<img
									src={item.url}
									alt="Post media"
									class="h-full w-full object-cover transition-transform duration-500 hover:scale-105"
								/>
							{:else if item.type === 'video' || item.type?.startsWith('video')}
								<video src={item.url} class="h-full w-full object-cover"></video>
								<div class="pointer-events-none absolute inset-0 flex items-center justify-center">
									<div class="rounded-full bg-black/50 p-2 text-white">▶</div>
								</div>
							{/if}

							{#if isOverlayNeeded}
								<div
									class="absolute inset-0 flex items-center justify-center bg-black/60 transition-colors hover:bg-black/70"
								>
									<span class="text-3xl font-bold text-white">+{mediaCount - 3}</span>
								</div>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		{/if}

		<div
			class="text-muted-foreground flex items-center justify-between border-b border-white/10 pb-2 text-sm"
		>
			<span>{safePost.total_reactions || 0} Likes</span>
			<span>{safePost.total_comments || 0} Comments</span>
		</div>

		<div class="flex justify-around pt-2">
			<Button
				variant="ghost"
				class="text-muted-foreground hover:bg-primary/10 hover:text-primary flex flex-1 items-center justify-center space-x-2 rounded-lg transition-all {isLikedByCurrentUser
					? 'text-red-500 hover:text-red-600'
					: ''}"
				onclick={handleLike}
			>
				<Heart size={20} class={isLikedByCurrentUser ? 'fill-current' : ''} />
				<span class="font-medium">{isLikedByCurrentUser ? 'Liked' : 'Like'}</span>
			</Button>
			<Button
				variant="ghost"
				class="text-muted-foreground hover:bg-primary/10 hover:text-primary flex flex-1 items-center justify-center space-x-2 rounded-lg transition-all"
				onclick={handleComment}
			>
				<MessageSquare size={20} />
				<span class="font-medium">Comment</span>
			</Button>
			<Button
				variant="ghost"
				class="text-muted-foreground hover:bg-primary/10 hover:text-primary flex flex-1 items-center justify-center space-x-2 rounded-lg transition-all"
				onclick={handleShare}
			>
				<Share2 size={20} />
				<span class="font-medium">Share</span>
			</Button>
		</div>

		{#if showComments}
			<CommentSection postId={safePost.id} />
		{/if}
	</div>
{/if}
