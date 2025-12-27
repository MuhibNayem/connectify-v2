<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		X,
		ChevronLeft,
		ChevronRight,
		Volume2,
		VolumeX,
		Heart,
		ThumbsUp,
		Laugh,
		Frown,
		Angry,
		Eye
	} from '@lucide/svelte';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { fade, scale, slide } from 'svelte/transition';
	import { apiRequest } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';

	// Props
	let {
		storyGroups = [],
		initialGroupIndex = 0,
		onClose
	} = $props<{
		storyGroups: any[];
		initialGroupIndex: number;
		onClose: () => void;
	}>();

	// State
	let currentGroupIndex = $state(initialGroupIndex);
	let currentStoryIndex = $state(0);
	let progress = $state(0);
	let isPaused = $state(false);
	let isMuted = $state(false);
	let videoEl: HTMLVideoElement | undefined = $state();

	// Viewers State
	let showViewersList = $state(false);
	let viewers = $state<any[]>([]);
	let loadingViewers = $state(false);

	// Floating emoji state
	let floatingReactions = $state<{ id: number; emoji: string; x: number }[]>([]);

	let currentUser = $derived(auth.state.user);
	let currentGroup = $derived(storyGroups[currentGroupIndex]);
	let currentStory = $derived(currentGroup?.stories[currentStoryIndex]);
	let isOwnStory = $derived(
		currentUser &&
			currentStory &&
			(currentStory.author?.id === currentUser.id || currentStory.user_id === currentUser.id)
	);

	let timer: any;
	const STORY_DURATION = 5000;
	const TICK_RATE = 100;

	const REACTION_EMOJIS: Record<string, string> = {
		LIKE: '❤️',
		LOVE: '😍',
		HAHA: '😂',
		SAD: '😢',
		ANGRY: '😡'
	};

	onMount(() => {
		startTimer();
	});

	onDestroy(() => {
		clearInterval(timer);
	});

	// Effect to reset/start timer when story changes
	$effect(() => {
		if (currentStory) {
			resetTimer();
			recordView(currentStory.id);
			// Reset viewers list state
			showViewersList = false;
			viewers = [];
			floatingReactions = [];
		}
	});

	async function recordView(storyId: string) {
		if (!currentUser || isOwnStory) return; // Don't record own views
		try {
			// Fire and forget view recording
			apiRequest('POST', `/stories/${storyId}/view`, {}, true).catch(console.error);
		} catch (e) {
			console.error(e);
		}
	}

	async function sendReaction(type: string) {
		if (!currentStory) return;

		const emoji = REACTION_EMOJIS[type] || '❤️';

		// Spawn multiple floating emojis for "Messenger effect"
		for (let i = 0; i < 5; i++) {
			setTimeout(() => {
				const id = Date.now() + Math.random();
				const x = 50 + (Math.random() * 20 - 10); // Random X around center
				floatingReactions = [...floatingReactions, { id, emoji, x }];

				// Remove after animation
				setTimeout(() => {
					floatingReactions = floatingReactions.filter((r) => r.id !== id);
				}, 1500);
			}, i * 100);
		}

		try {
			await apiRequest('POST', `/stories/${currentStory.id}/react`, { type }, true);
		} catch (e) {
			console.error('Failed to react:', e);
		}
	}

	async function fetchViewers() {
		if (!currentStory || !isOwnStory) return;

		loadingViewers = true;
		try {
			const res = await apiRequest('GET', `/stories/${currentStory.id}/viewers`, undefined, true);
			viewers = res || [];
		} catch (e) {
			console.error('Failed to fetch viewers:', e);
		} finally {
			loadingViewers = false;
		}
	}

	function toggleViewers() {
		showViewersList = !showViewersList;
		if (showViewersList) {
			isPaused = true;
			fetchViewers();
		} else {
			isPaused = false;
		}
	}

	// Add global style for animation if not present
	$effect.root(() => {
		if (typeof document !== 'undefined' && !document.getElementById('story-animations')) {
			const style = document.createElement('style');
			style.id = 'story-animations';
			style.innerHTML = `
                @keyframes float-up {
                    0% { transform: translateY(0) scale(0.5); opacity: 0; }
                    10% { opacity: 1; scale: 1.2; }
                    100% { transform: translateY(-300px) scale(1); opacity: 0; }
                }
                .animate-float-up {
                    animation: float-up 1.5s ease-out forwards;
                }
            `;
			document.head.appendChild(style);
		}
	});

	function resetTimer() {
		clearInterval(timer);
		progress = 0;

		if (currentStory && currentStory.media_type === 'video') {
			// Video handles its own progress via timeupdate
		} else {
			startTimer();
		}
	}

	function startTimer() {
		clearInterval(timer);
		timer = setInterval(() => {
			if (!isPaused && !showViewersList) {
				progress += (TICK_RATE / STORY_DURATION) * 100;
				if (progress >= 100) {
					nextStory();
				}
			}
		}, TICK_RATE);
	}

	function nextStory() {
		if (showViewersList) return; // Don't advance if list open
		if (currentStoryIndex < currentGroup.stories.length - 1) {
			currentStoryIndex++;
		} else {
			// End of this user's stories, go to next user
			if (currentGroupIndex < storyGroups.length - 1) {
				currentGroupIndex++;
				currentStoryIndex = 0;
			} else {
				onClose(); // End of all stories
			}
		}
	}

	function prevStory() {
		if (showViewersList) return;
		if (currentStoryIndex > 0) {
			currentStoryIndex--;
		} else {
			// Go to previous user
			if (currentGroupIndex > 0) {
				currentGroupIndex--;
				currentStoryIndex = storyGroups[currentGroupIndex].stories.length - 1; // Last story of prev user
			}
		}
	}

	function handleVideoTimeUpdate(e: Event) {
		const target = e.target as HTMLVideoElement;
		if (target.duration) {
			progress = (target.currentTime / target.duration) * 100;
		}
	}

	function handleVideoEnded() {
		nextStory();
	}
</script>

<!-- Backdrop -->
<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-black"
	transition:fade={{ duration: 200 }}
>
	<!-- Close Button -->
	<button class="absolute right-4 top-4 z-50 text-white hover:text-gray-300" onclick={onClose}>
		<X size={32} />
	</button>

	<!-- Desktop Navigation Arrows -->
	<button
		class="absolute left-4 z-40 hidden rounded-full bg-white/10 p-2 text-white/50 hover:text-white md:flex"
		onclick={prevStory}
	>
		<ChevronLeft size={32} />
	</button>
	<button
		class="absolute right-4 z-40 hidden rounded-full bg-white/10 p-2 text-white/50 hover:text-white md:flex"
		onclick={nextStory}
	>
		<ChevronRight size={32} />
	</button>

	<!-- Main Content Area -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="relative h-full w-full overflow-hidden bg-gray-900 md:h-[85vh] md:max-w-[400px] md:rounded-xl"
		onclick={(e) => {
			if (showViewersList) return; // Don't navigate if list open
			// Click left/right side logic
			const rect = e.currentTarget.getBoundingClientRect();
			const x = e.clientX - rect.left;
			if (x < rect.width / 3) prevStory();
			else nextStory();
		}}
	>
		<!-- Progress Bars -->
		<div class="absolute left-0 right-0 top-0 z-20 flex gap-1 p-2">
			{#if currentGroup}
				{#each currentGroup.stories as story, i}
					<div class="h-1 flex-1 overflow-hidden rounded-full bg-white/30">
						<div
							class="h-full bg-white transition-all duration-100 ease-linear"
							style="width: {i < currentStoryIndex
								? '100%'
								: i === currentStoryIndex
									? `${progress}%`
									: '0%'}"
						></div>
					</div>
				{/each}
			{/if}
		</div>

		<!-- User Info Header -->
		{#if currentGroup}
			<div class="absolute left-0 right-0 top-4 z-20 flex items-center justify-between px-4 pt-2">
				<div class="flex items-center gap-2">
					<Avatar class="h-8 w-8 border border-white/50">
						<AvatarImage src={currentGroup.user.avatar} />
						<AvatarFallback>{currentGroup.user.username?.[0]}</AvatarFallback>
					</Avatar>
					<div class="flex flex-col">
						<span class="text-sm font-semibold text-white shadow-black drop-shadow-md"
							>{currentGroup.user.username}</span
						>
					</div>
				</div>

				<!-- Controls -->
				<div class="flex gap-2">
					{#if currentStory && currentStory.media_type === 'video'}
						<button
							onclick={(e) => {
								e.stopPropagation();
								isMuted = !isMuted;
							}}
						>
							{#if isMuted}
								<VolumeX size={20} class="text-white" />
							{:else}
								<Volume2 size={20} class="text-white" />
							{/if}
						</button>
					{/if}
				</div>
			</div>
		{/if}

		<!-- Media -->
		<div class="flex h-full w-full items-center justify-center bg-black">
			{#if currentStory}
				{#if currentStory.media_type === 'video'}
					<video
						src={currentStory.media_url}
						class="h-full w-full object-contain"
						autoplay
						muted={isMuted}
						playsinline
						onpause={() => (isPaused = true)}
						onplay={() => (isPaused = false)}
						ontimeupdate={handleVideoTimeUpdate}
						onended={handleVideoEnded}
						bind:this={videoEl}
					></video>
				{:else}
					<img
						src={currentStory.media_url}
						alt="Story"
						class="h-full w-full object-cover md:object-contain"
					/>
				{/if}
			{/if}
		</div>

		<!-- Footer overlay -->
		<div
			class="absolute bottom-0 left-0 right-0 z-30 flex flex-col items-center bg-gradient-to-t from-black/80 to-transparent p-4 pb-8"
		>
			{#if isOwnStory}
				<!-- Viewers Button (Author View) -->
				<button
					class="mb-2 flex items-center gap-2 rounded-full px-3 py-2 text-white transition-colors hover:bg-white/10"
					onclick={(e) => {
						e.stopPropagation();
						toggleViewers();
					}}
				>
					<Eye size={20} />
					<span class="font-semibold">{currentStory?.view_count || 0} Viewers</span>
				</button>
			{:else}
				<!-- Reaction Bar (Viewer View) -->
				<div class="flex items-center gap-4">
					<!-- Helper to prevent bubble -->
					<!-- svelte-ignore a11y_click_events_have_key_events -->
					<!-- svelte-ignore a11y_no_static_element_interactions -->
					<div
						class="flex gap-2 rounded-full border border-white/10 bg-black/40 p-2 px-4 backdrop-blur-md"
						onclick={(e) => e.stopPropagation()}
					>
						{#each ['LIKE', 'LOVE', 'HAHA', 'SAD', 'ANGRY'] as type}
							<button
								class="text-2xl transition-transform hover:scale-125 active:scale-95"
								onclick={() => sendReaction(type)}
							>
								{#if type === 'LIKE'}
									❤️
								{:else if type === 'LOVE'}
									😍
								{:else if type === 'HAHA'}
									😂
								{:else if type === 'SAD'}
									😢
								{:else if type === 'ANGRY'}
									😡
								{/if}
							</button>
						{/each}
					</div>
				</div>
			{/if}
		</div>

		<!-- Floating Reactions Container -->
		<!-- Pointer events none so clicks pass through -->
		<div class="pointer-events-none absolute inset-0 z-40 overflow-hidden">
			{#each floatingReactions as reaction (reaction.id)}
				<div
					class="animate-float-up absolute text-4xl"
					style="left: {reaction.x}%; bottom: 100px;"
					transition:fade={{ duration: 1000 }}
				>
					{reaction.emoji}
				</div>
			{/each}
		</div>

		<!-- Viewers List Sheet -->
		{#if showViewersList}
			<div
				class="absolute inset-x-0 bottom-0 top-20 z-50 flex flex-col rounded-t-2xl border-t border-white/10 bg-black/90 p-4 backdrop-blur-xl"
				transition:slide={{ axis: 'y', duration: 300 }}
				onclick={(e) => e.stopPropagation()}
			>
				<div class="mb-4 flex items-center justify-between border-b border-white/10 pb-2">
					<h3 class="text-lg font-semibold text-white">Viewers ({viewers.length})</h3>
					<button onclick={toggleViewers} class="text-white/70 hover:text-white">
						<X size={20} />
					</button>
				</div>

				<div class="flex-1 space-y-3 overflow-y-auto">
					{#if loadingViewers}
						<div class="py-4 text-center text-white/50">Loading...</div>
					{:else if viewers.length === 0}
						<div class="py-4 text-center text-white/50">No views yet.</div>
					{:else}
						{#each viewers as viewer}
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-3">
									<Avatar class="h-10 w-10">
										<AvatarImage src={viewer.user.avatar} />
										<AvatarFallback>{viewer.user.username?.[0]}</AvatarFallback>
									</Avatar>
									<div class="font-medium text-white">{viewer.user.username}</div>
								</div>
								{#if viewer.reaction_type}
									<div class="text-2xl">
										{#if viewer.reaction_type === 'LIKE'}
											❤️
										{:else if viewer.reaction_type === 'LOVE'}
											😍
										{:else if viewer.reaction_type === 'HAHA'}
											😂
										{:else if viewer.reaction_type === 'SAD'}
											😢
										{:else if viewer.reaction_type === 'ANGRY'}
											😡
										{/if}
									</div>
								{/if}
							</div>
						{/each}
					{/if}
				</div>
			</div>
		{/if}
	</div>
</div>
