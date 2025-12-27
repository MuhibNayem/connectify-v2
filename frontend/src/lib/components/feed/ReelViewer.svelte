<script lang="ts">
	import { onMount } from 'svelte';
	import {
		X,
		Heart,
		MessageCircle,
		Share2,
		Volume2,
		VolumeX,
		Play,
		Reply,
		ThumbsUp
	} from '@lucide/svelte';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { fade } from 'svelte/transition';
	import { apiRequest } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import { fly } from 'svelte/transition';
	import { Send } from '@lucide/svelte';
	import UserMentionDropdown from './UserMentionDropdown.svelte';

	// Props
	let {
		reels = [],
		initialIndex = 0,
		onClose
	} = $props<{
		reels: any[];
		initialIndex: number;
		onClose: () => void;
	}>();

	let currentUser = $derived(auth.state.user);
	let currentIndex = $state(initialIndex);
	let containerRef: HTMLDivElement | undefined = $state();
	let videoRefs: HTMLVideoElement[] = $state([]);
	let isMuted = $state(false);
	let isPaused = $state(false);

	// Interaction State
	let showComments = $state(false);
	let comments = $state<any[]>([]);
	let newCommentText = $state('');
	let likedReels = $state<Record<string, boolean>>({});

	// We initialize offset based on initial reels length if any, but since we receive a full list from parent...
	// If parent fetched 10, offset should be 10 for next fetch?
	// But `reels` passed in might just be a chunk?
	// Let's assume `reels` prop is the starting list.
	// We need to know the offset to fetch next page.
	// Assuming parent fetched `limit=10`, `offset=0`.
	// Ideally parent should pass pagination state or we manage it fully.
	// We'll manage it here, assuming initial list length is current count.

	// Reply state
	let replyingTo = $state<string | null>(null); // commentID being replied to
	let replyText = $state('');

	// Mention-related state
	let showMentions = $state(false);
	let mentionQuery = $state('');
	let mentionStartPos = $state(-1);
	let mentionedUsers = $state<any[]>([]);

	// Expanded replies state
	// Expanded replies state
	let expandedReplies = $state<Record<string, boolean>>({});

	// Pagination State
	let commentsPage = $state(0);
	let commentsLimit = 20;
	let hasMoreComments = $state(true);
	let loadingComments = $state(false);

	// Infinite Scroll State
	let reelsLimit = 10;
	let reelsOffset = $state(0);
	let hasMoreReels = $state(true);
	let loadingMoreReels = $state(false);

	onMount(() => {
		if (containerRef && initialIndex > 0) {
			containerRef.scrollTo({ top: initialIndex * containerRef.clientHeight, behavior: 'instant' });
		}

		activeObserver = new IntersectionObserver(
			(entries) => {
				entries.forEach((entry) => {
					const index = Number(entry.target.getAttribute('data-index'));
					const video = videoRefs[index];

					if (entry.isIntersecting) {
						currentIndex = index;
						if (video && video.paused) {
							video.currentTime = 0;
							video
								.play()
								.then(() => {
									incrementView(reels[index].id); // Initial view on play
								})
								.catch((e) => console.error('Auto-play error:', e));
							isPaused = false;
						}
						// Preload next video
						const nextReel = reels[index + 1];
						if (nextReel) {
							const link = document.createElement('link');
							link.rel = 'preload';
							link.as = 'video';
							link.href = nextReel.video_url;
							document.head.appendChild(link);
						}

						// Loop Handling: If nearing end, load more
						if (index >= reels.length - 2 && hasMoreReels && !loadingMoreReels) {
							loadMoreReels();
						}
					} else {
						if (video) {
							video.pause();
							if (index === currentIndex) {
								isPaused = true;
							}
						}
					}
				});
			},
			{ threshold: 0.6 }
		);

		const children = containerRef?.children;
		if (children) {
			for (let i = 0; i < children.length; i++) {
				activeObserver.observe(children[i]);
			}
		}

		return () => {
			activeObserver?.disconnect();
		};
	});

	let activeObserver: IntersectionObserver | null = null;
	$effect(() => {
		// When reels changes, we need to observe new children
		// But usually `each` block handles it.
		// If we use an `action` it would be cleaner.
		// For now, let's just re-run observe loop on container children when reels changes.
		if (activeObserver && containerRef) {
			const children = containerRef.children;
			for (let i = 0; i < children.length; i++) {
				activeObserver.observe(children[i]);
			}
		}
	});

	function handleScroll(e: Event) {
		// active index logic handled by IntersectionObserver
	}

	function toggleMute() {
		isMuted = !isMuted;
	}

	function togglePlay(index: number) {
		const video = videoRefs[index];
		if (!video) return;
		if (video.paused) {
			video.play();
			isPaused = false;
			incrementView(reels[index].id); // View on manual play? Maybe redundant if it was just paused?
			// FB: "A view is counted as soon as the video starts to play".
			// If I pause and resume, does it count? "Rewatches count".
			// But resuming isn't a rewatch. A rewatch is a loop.
			// So maybe valid to NOT count here if it's just resuming.
			// But "Initial play" counts.
			// Let's stick to: Count on "Intersection Auto-Play" and "Loop/Ended".
			// And maybe "Manual First Play" if it didn't auto-play?
		} else {
			video.pause();
			isPaused = true;
		}
	}

	async function likeReel(reel: any) {
		const isLiked = likedReels[reel.id];
		likedReels[reel.id] = !isLiked;

		try {
			await apiRequest('POST', `/reels/${reel.id}/react`, { type: 'LIKE' }, true);
		} catch (error) {
			console.error('Failed to like reel', error);
			likedReels[reel.id] = isLiked;
		}
	}

	function toggleComments(reel: any) {
		showComments = !showComments;
		if (showComments) {
			loadComments(reel.id, true);
		}
	}

	async function loadComments(reelID: string, reset = false) {
		if (reset) {
			commentsPage = 0;
			comments = [];
			hasMoreComments = true;
		}

		if (!hasMoreComments || loadingComments) return;

		loadingComments = true;
		try {
			const offset = commentsPage * commentsLimit;
			const res = await apiRequest(
				'GET',
				`/reels/${reelID}/comments?limit=${commentsLimit}&offset=${offset}`,
				undefined,
				true
			);

			if (res && res.length > 0) {
				comments = [...comments, ...res];
				commentsPage++;
				if (res.length < commentsLimit) {
					hasMoreComments = false;
				}
			} else {
				hasMoreComments = false;
			}
		} catch (e) {
			console.error('Failed to load comments', e);
			if (reset) comments = [];
		} finally {
			loadingComments = false;
		}
	}

	async function postComment(reel: any) {
		if (!newCommentText.trim()) return;

		const tempComment = {
			id: 'temp-' + Date.now(),
			content: newCommentText,
			author: currentUser,
			created_at: new Date().toISOString(),
			replies: []
		};

		comments = [...comments, tempComment];
		const textToSend = newCommentText;
		newCommentText = '';

		try {
			const res = await apiRequest(
				'POST',
				`/reels/${reel.id}/comments`,
				{
					content: textToSend,
					mentions: mentionedUsers.map((u) => u.id)
				},
				true
			);
			comments = comments.map((c) => (c.id === tempComment.id ? res : c));
		} catch (e) {
			console.error('Failed to post comment', e);
			comments = comments.filter((c) => c.id !== tempComment.id);
		} finally {
			mentionedUsers = [];
		}
	}

	function handleInput(event: Event) {
		const textarea = event.target as HTMLInputElement;
		const cursorPos = textarea.selectionStart || 0;
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
		const before = newCommentText.substring(0, mentionStartPos);
		const after = newCommentText.substring(mentionStartPos + 1 + mentionQuery.length);

		newCommentText = `${before}@${user.username} ${after}`;

		if (!mentionedUsers.some((u) => u.id === user.id)) {
			mentionedUsers.push(user);
		}

		showMentions = false;
	}

	async function postReply(reel: any, commentId: string) {
		if (!replyText.trim()) return;

		const tempReply = {
			id: 'temp-' + Date.now(),
			content: replyText,
			author: currentUser,
			created_at: new Date().toISOString()
		};

		// Optimistic update
		comments = comments.map((c) => {
			if (c.id === commentId) {
				return { ...c, replies: [...(c.replies || []), tempReply] };
			}
			return c;
		});

		const textToSend = replyText;
		replyText = '';
		replyingTo = null;

		try {
			const res = await apiRequest(
				'POST',
				`/reels/${reel.id}/comments/${commentId}/replies`,
				{ content: textToSend },
				true
			);
			// Replace temp reply with real one
			comments = comments.map((c) => {
				if (c.id === commentId) {
					return { ...c, replies: c.replies.map((r: any) => (r.id === tempReply.id ? res : r)) };
				}
				return c;
			});
		} catch (e) {
			console.error('Failed to post reply', e);
			// Remove temp reply on error
			comments = comments.map((c) => {
				if (c.id === commentId) {
					return { ...c, replies: c.replies.filter((r: any) => r.id !== tempReply.id) };
				}
				return c;
			});
		}
	}

	async function reactToComment(reel: any, commentId: string, reactionType: string) {
		try {
			await apiRequest(
				'POST',
				`/reels/${reel.id}/comments/${commentId}/react`,
				{ reaction_type: reactionType },
				true
			);
			// Reload comments to get updated counts
			await loadComments(reel.id);
		} catch (e) {
			console.error('Failed to react', e);
		}
	}

	function toggleReplies(commentId: string) {
		expandedReplies[commentId] = !expandedReplies[commentId];
	}

	function startReply(commentId: string) {
		replyingTo = commentId;
	}

	function cancelReply() {
		replyingTo = null;
		replyText = '';
	}

	// Parse @mentions like PostCard
	function parseContent(content: string, mentions: any[]) {
		if (!content) return '';
		return content.replace(/@(\w+)/g, (match, username) => {
			const user = (mentions || []).find((u: any) => u.username === username);
			if (user) {
				return `<a href="/profile/${user.id}" class="text-blue-400 font-semibold">${username}</a>`;
			}
			return match;
		});
	}

	function formatNumber(num: number): string {
		if (num >= 1000000) {
			return (num / 1000000).toFixed(1) + 'M';
		}
		if (num >= 1000) {
			return (num / 1000).toFixed(1) + 'k';
		}
		return num.toString();
	}

	// viewTimer removed as views are now immediate on play/loop

	async function loadMoreReels() {
		if (loadingMoreReels || !hasMoreReels) return;
		loadingMoreReels = true;

		// Current offset = current length
		const currentOffset = reels.length;

		try {
			const res = await apiRequest(
				'GET',
				`/reels?limit=${reelsLimit}&offset=${currentOffset}`,
				undefined,
				true
			);
			const newReels = res || [];

			if (newReels.length < reelsLimit) {
				hasMoreReels = false;
			}

			// Append new reels
			if (newReels.length > 0) {
				const currentLen = reels.length;
				reels = [...reels, ...newReels];

				// Observe new elements - Wait for DOM update
				setTimeout(() => {
					if (containerRef && activeObserver) {
						const children = containerRef.children;
						// Observe only new children
						for (let i = currentLen; i < children.length; i++) {
							activeObserver.observe(children[i]);
						}
					}
				}, 100);
			}
		} catch (error) {
			console.error('Failed to load more reels', error);
		} finally {
			loadingMoreReels = false;
		}
	}

	async function incrementView(reelID: string) {
		const reel = reels.find((r) => r.id === reelID);
		if (!reel || !currentUser) return;
		if (reel.user_id === currentUser.id || reel.author?.id === currentUser.id) return;

		try {
			await apiRequest('POST', `/reels/${reelID}/view`, {}, true);
		} catch (e) {
			console.error('Failed to increment view', e);
		}
	}
</script>

<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-black"
	transition:fade={{ duration: 200 }}
>
	<!-- Close Button -->
	<button
		class="absolute right-4 top-4 z-50 rounded-full bg-black/20 p-2 text-white backdrop-blur-md transition-colors hover:bg-black/40"
		onclick={onClose}
	>
		<X size={24} />
	</button>

	<!-- Main Feed Container -->
	<div
		bind:this={containerRef}
		class="no-scrollbar h-full w-full snap-y snap-mandatory overflow-y-scroll bg-black md:w-[450px]"
		onscroll={handleScroll}
	>
		{#each reels as reel, i (reel.id)}
			<div
				class="relative flex h-full w-full snap-start items-center justify-center bg-gray-900"
				data-index={i}
			>
				<!-- Video -->
				<!-- svelte-ignore a11y_media_has_caption -->
				<video
					src={reel.video_url}
					poster={reel.thumbnail_url}
					class="h-full w-full object-cover"
					loop
					muted={isMuted}
					playsinline
					bind:this={videoRefs[i]}
					onended={() => {
						videoRefs[i].currentTime = 0;
						videoRefs[i].play();
						incrementView(reel.id); // Valid rewatch view
					}}
					onclick={() => togglePlay(i)}
				></video>

				{#if i === currentIndex && isPaused}
					<div
						class="pointer-events-none absolute inset-0 flex items-center justify-center bg-black/20"
					>
						<Play size={48} class="text-white/80" fill="currentColor" />
					</div>
				{/if}

				<!-- Overlay Info -->
				<div
					class="pointer-events-none absolute inset-0 bg-gradient-to-b from-transparent via-transparent to-black/80"
				></div>

				<div
					class="pointer-events-auto absolute bottom-0 left-0 right-0 flex items-end justify-between p-4 pb-12"
				>
					<!-- Left: Author & Caption -->
					<div class="flex-1 pr-4">
						<div class="mb-3 flex items-center gap-3">
							<Avatar class="h-10 w-10 border border-white/20">
								<AvatarImage src={reel.author?.avatar} />
								<AvatarFallback>{reel.author?.username?.[0]}</AvatarFallback>
							</Avatar>
							<span class="font-bold text-white">{reel.author?.username}</span>
						</div>
						<p class="mb-2 line-clamp-2 text-sm text-white">{reel.caption}</p>
					</div>

					<!-- Right: Actions -->
					<div class="flex flex-col items-center gap-6">
						<button class="group flex flex-col items-center gap-1" onclick={() => likeReel(reel)}>
							<div
								class="rounded-full bg-white/10 p-3 backdrop-blur-md transition-colors group-hover:bg-white/20"
							>
								<Heart
									size={28}
									class="transition-transform group-active:scale-90 {likedReels[reel.id]
										? 'fill-red-500 text-red-500'
										: 'text-white'}"
								/>
							</div>
							<span class="text-xs font-medium text-white"
								>{formatNumber(reel.likes + (likedReels[reel.id] ? 1 : 0))}</span
							>
						</button>

						<button
							class="group flex flex-col items-center gap-1"
							onclick={() => toggleComments(reel)}
						>
							<div
								class="rounded-full bg-white/10 p-3 backdrop-blur-md transition-colors group-hover:bg-white/20"
							>
								<MessageCircle size={28} class="text-white" />
							</div>
							<span class="text-xs font-medium text-white">{formatNumber(reel.comments || 0)}</span>
						</button>

						<button class="group flex flex-col items-center gap-1">
							<div
								class="rounded-full bg-white/10 p-3 backdrop-blur-md transition-colors group-hover:bg-white/20"
							>
								<Share2 size={28} class="text-white" />
							</div>
							<span class="text-xs font-medium text-white">Share</span>
						</button>
					</div>
				</div>

				<!-- Mute Toggle -->
				<button
					class="absolute left-4 top-4 rounded-full bg-black/20 p-2 text-white backdrop-blur-md transition-colors hover:bg-black/40"
					onclick={(e) => {
						e.stopPropagation();
						toggleMute();
					}}
				>
					{#if isMuted}
						<VolumeX size={20} />
					{:else}
						<Volume2 size={20} />
					{/if}
				</button>
			</div>
		{/each}
	</div>

	<!-- Comments Drawer -->
	{#if showComments}
		<div
			class="absolute bottom-0 left-0 right-0 z-50 h-[70%] rounded-t-3xl border-t border-white/10 bg-black/80 shadow-2xl backdrop-blur-xl md:mx-auto md:w-[450px]"
			transition:fly={{ y: 300, duration: 300 }}
		>
			<div class="flex h-full flex-col">
				<!-- Header -->
				<div class="flex items-center justify-between border-b border-white/10 p-4">
					<h3 class="text-sm font-bold text-white">Comments</h3>
					<button
						onclick={() => (showComments = false)}
						class="rounded-full p-1 text-white/50 transition-colors hover:bg-white/10 hover:text-white"
					>
						<X size={20} />
					</button>
				</div>

				<!-- List -->
				<div class="flex-1 space-y-4 overflow-y-auto p-4">
					{#if comments.length === 0}
						<div class="flex h-full flex-col items-center justify-center gap-2 text-white/40">
							<MessageCircle size={32} class="opacity-50" />
							<p class="text-sm">No comments yet.</p>
							<p class="text-xs">Start the conversation!</p>
						</div>
					{/if}
					{#each comments as comment}
						<div class="space-y-2">
							<!-- Main Comment -->
							<div class="flex gap-3">
								<Avatar class="h-8 w-8 flex-shrink-0 border border-white/10">
									<AvatarImage src={comment.author?.avatar} />
									<AvatarFallback>{comment.author?.username?.[0]}</AvatarFallback>
								</Avatar>
								<div class="flex-1 space-y-1">
									<div class="flex items-baseline gap-2">
										<span class="text-xs font-bold text-white/90">{comment.author?.username}</span>
										<span class="text-[10px] text-white/40"
											>{new Date(comment.created_at).toLocaleDateString()}</span
										>
									</div>
									<p class="text-sm text-white/80">
										{@html parseContent(comment.content, comment.mentioned_users || [])}
									</p>

									<!-- Actions -->
									<div class="flex items-center gap-3 pt-1">
										<button
											class="flex items-center gap-1 text-xs text-white/50 hover:text-white/80"
											onclick={() => reactToComment(reels[currentIndex], comment.id, 'LIKE')}
										>
											<ThumbsUp size={12} />
											Like
										</button>
										<button
											class="flex items-center gap-1 text-xs text-white/50 hover:text-white/80"
											onclick={() => startReply(comment.id)}
										>
											<Reply size={12} />
											Reply
										</button>
										{#if comment.replies && comment.replies.length > 0}
											<button
												class="text-xs text-blue-400 hover:text-blue-300"
												onclick={() => toggleReplies(comment.id)}
											>
												{expandedReplies[comment.id] ? 'Hide' : 'View'}
												{comment.replies.length}
												{comment.replies.length === 1 ? 'reply' : 'replies'}
											</button>
										{/if}
									</div>
								</div>
							</div>

							<!-- Replies (Flat Structure) -->
							{#if expandedReplies[comment.id] && comment.replies && comment.replies.length > 0}
								<div class="ml-11 space-y-2 border-l-2 border-white/10 pl-3">
									{#each comment.replies as reply}
										<div class="flex gap-2">
											<Avatar class="h-6 w-6">
												<AvatarImage src={reply.author?.avatar} />
												<AvatarFallback>{reply.author?.username?.[0]}</AvatarFallback>
											</Avatar>
											<div class="flex-1">
												<div class="flex items-baseline gap-2">
													<span class="text-xs font-bold text-white/90"
														>{reply.author?.username}</span
													>
													<span class="text-[10px] text-white/40"
														>{new Date(reply.created_at).toLocaleDateString()}</span
													>
												</div>
												<p class="text-xs text-white/80">
													{@html parseContent(reply.content, reply.mentions || [])}
												</p>
											</div>
										</div>
									{/each}
								</div>
							{/if}

							<!-- Reply Input (Inline) -->
							{#if replyingTo === comment.id}
								<div
									class="ml-11 flex items-center gap-2 rounded-full border border-white/20 bg-white/5 px-3 py-1.5"
								>
									<input
										type="text"
										placeholder="Write a reply..."
										class="flex-1 bg-transparent text-xs text-white placeholder-white/40 focus:outline-none"
										style="-webkit-appearance: none; -moz-appearance: none; appearance: none; border: none; box-shadow: none;"
										bind:value={replyText}
										onkeydown={(e) =>
											e.key === 'Enter' && postReply(reels[currentIndex], comment.id)}
									/>
									<button
										class="text-blue-400 hover:text-blue-300 disabled:opacity-30"
										disabled={!replyText.trim()}
										onclick={() => postReply(reels[currentIndex], comment.id)}
									>
										<Send size={14} />
									</button>
									<button class="text-white/40 hover:text-white/60" onclick={cancelReply}>
										<X size={14} />
									</button>
								</div>
							{/if}
						</div>
					{/each}

					<!-- Load More Button -->
					{#if hasMoreComments && comments.length > 0}
						<button
							class="w-full py-2 text-xs text-blue-400 hover:text-blue-300 disabled:opacity-50"
							onclick={() => loadComments(reels[currentIndex].id)}
							disabled={loadingComments}
						>
							{loadingComments ? 'Loading...' : 'Load more comments'}
						</button>
					{/if}
				</div>

				<!-- Input -->
				<div class="border-t border-white/10 bg-black/40 p-3 md:rounded-b-3xl">
					<div
						class="relative flex items-center gap-2 rounded-full border border-white/20 bg-white/5 px-4 py-2"
					>
						<input
							type="text"
							placeholder="Add a comment..."
							class="flex-1 bg-transparent text-sm text-white placeholder-white/40 focus:outline-none"
							style="-webkit-appearance: none; -moz-appearance: none; appearance: none; border: none; box-shadow: none;"
							bind:value={newCommentText}
							oninput={handleInput}
							onkeydown={(e) => {
								if (e.key === 'Enter' && !showMentions) {
									postComment(reels[currentIndex]);
								}
							}}
						/>
						<button
							class="text-blue-400 hover:text-blue-300 disabled:opacity-30"
							disabled={!newCommentText.trim()}
							onclick={() => postComment(reels[currentIndex])}
						>
							<Send size={18} />
						</button>

						{#if showMentions}
							<div class="glass-card absolute bottom-full left-0 z-50 mb-2 w-full max-w-[300px]">
								<UserMentionDropdown query={mentionQuery} onSelection={handleMentionSelection} />
							</div>
						{/if}
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.no-scrollbar::-webkit-scrollbar {
		display: none;
	}
	.no-scrollbar {
		-ms-overflow-style: none;
		scrollbar-width: none;
	}
</style>
