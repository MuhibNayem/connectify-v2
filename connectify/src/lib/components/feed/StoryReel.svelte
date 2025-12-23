<script lang="ts">
	import { auth } from '$lib/stores/auth.svelte';
	import { Plus, Video, Image as ImageIcon, Film } from '@lucide/svelte';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { apiRequest } from '$lib/api';
	import { onMount } from 'svelte';

	let currentUser = $derived(auth.state.user);
	let activeTab = $state<'stories' | 'reels'>('stories');
	let stories = $state<any[]>([]);
	let reels = $state<any[]>([]);
	let isLoading = $state(false);
	import { uploadFiles } from '$lib/api';
	import StoryComposer from './StoryComposer.svelte';
	import StoryViewer from './StoryViewer.svelte';
	import ReelViewer from './ReelViewer.svelte';

	let fileInput: HTMLInputElement;
	let showViewer = $state(false);
	let showReelViewer = $state(false);
	let viewingGroupIndex = $state(0);
	let viewingReelIndex = $state(0);
	// Grouped stories for the viewer: array of { user: Author, stories: Story[] }
	let storyGroups = $state<any[]>([]);

	// Pagination State
	let storiesOffset = $state(0);
	let storiesLimit = 10;
	let hasMoreStories = $state(true);
	let isLoadingStories = $state(false);

	// Composer State
	let showComposer = $state(false);
	let selectedFile = $state<File | null>(null);

	async function fetchStories() {
		if (isLoadingStories || !hasMoreStories) return;
		isLoadingStories = true;

		try {
			// Fetch friends' active stories with pagination
			const res = await apiRequest(
				'GET',
				`/stories?limit=${storiesLimit}&offset=${storiesOffset}`,
				undefined,
				true
			);
			const rawStories = res || [];

			if (rawStories.length < storiesLimit) {
				hasMoreStories = false; // No more stories from server?
				// Note: Since we paginate AUTHORS, not stories, this check might be tricky if backend returns flat list of stories.
				// But backend aggregates authors. So if we requested 10 authors, and got stories for 10 authors...
				// Actually, GetStoriesFeed returns a FLAT LIST of models.Story.
				// If we have < limit*N stories, we don't know if we exhausted authors.
				// WAIT: If Backend returns stories for X authors. We don't know how many authors unless we check unique user_ids.
				// BUT: The backend logic creates userIDs list from friends. Then paginates authors.
				// So if we get 0 stories, likely no more authors with stories.
				// Let's assume if rawStories is empty, we are done.
			}
			if (rawStories.length === 0) {
				hasMoreStories = false;
			}

			// Group by user
			const groups: Record<string, any> = {};

			// We need to merge with existing groups if we already have them, OR just append new groups?
			// Since we paginate authors, new fetch should return NEW authors.
			// So strict append should work.

			rawStories.forEach((story: any) => {
				const userId = story.author?.id || story.user_id;
				if (!groups[userId]) {
					groups[userId] = {
						user: story.author || { username: 'Unknown', avatar: '' }, // Fallback
						stories: []
					};
				}
				groups[userId].stories.push(story);
			});

			const newGroups = Object.values(groups);
			storyGroups = [...storyGroups, ...newGroups];
			stories = [...stories, ...rawStories];

			// Increment offset for next fetch
			// Offset should be number of AUTHORS fetched? Or simple skip?
			// Backend `GetActiveStoryAuthors` takes `limit, offset`.
			// So we should increment offset by `storiesLimit`.
			storiesOffset += storiesLimit;
		} catch (error) {
			console.error('Failed to fetch stories:', error);
			hasMoreStories = false;
		} finally {
			isLoadingStories = false;
		}
	}

	async function fetchReels() {
		try {
			const res = await apiRequest('GET', '/reels?limit=10', undefined, true);
			reels = res || [];
		} catch (error) {
			console.error('Failed to fetch reels:', error);
		}
	}

	onMount(() => {
		fetchStories();
		fetchReels();
	});

	function openViewer(index: number) {
		viewingGroupIndex = index;
		showViewer = true;
	}

	function openReelViewer(index: number) {
		viewingReelIndex = index;
		showReelViewer = true;
	}

	// Privacy State (Def. to Friends, user changes in composer)

	async function handleCreate() {
		fileInput?.click();
	}

	async function handleFileSelect(e: Event) {
		const target = e.target as HTMLInputElement;
		if (target.files && target.files.length > 0) {
			const file = target.files[0];

			// 1. Validate File Size
			const maxSize = activeTab === 'reels' ? 100 * 1024 * 1024 : 50 * 1024 * 1024;
			if (file.size > maxSize) {
				alert(`File too large. Max size is ${activeTab === 'reels' ? '100MB' : '50MB'}.`);
				target.value = '';
				return;
			}

			// 2. Validate Video Constraints
			if (file.type.startsWith('video/')) {
				const videoUrl = URL.createObjectURL(file);
				const videoEl = document.createElement('video');
				videoEl.src = videoUrl;

				await new Promise((resolve, reject) => {
					videoEl.onloadedmetadata = () => {
						const maxDuration = activeTab === 'reels' ? 60 : 30;
						if (videoEl.duration > maxDuration) {
							alert(`Video too long. Max duration is ${maxDuration} seconds.`);
							reject('Metadata check failed');
							return;
						}
						// Max 1080p height
						if (videoEl.videoHeight > 1920) {
							alert(`Resolution too high. Max 1080p (1920px height).`);
							reject('Metadata check failed');
							return;
						}
						resolve(true);
					};
					videoEl.onerror = () => reject('Failed to load video metadata');
				}).catch(() => {
					target.value = '';
					URL.revokeObjectURL(videoUrl);
					return;
				});
				URL.revokeObjectURL(videoUrl);
				if (!target.value) return; // Validation failed
			}

			// OPEN COMPOSER INSTEAD OF UPLOADING
			selectedFile = file;
			showComposer = true;

			// Reset input so selecting same file works again if cancelled
			target.value = '';
		}
	}

	async function generateThumbnail(videoFile: File): Promise<File> {
		return new Promise((resolve, reject) => {
			const video = document.createElement('video');
			video.preload = 'metadata';
			video.src = URL.createObjectURL(videoFile);
			video.muted = true;
			video.playsInline = true;
			video.currentTime = 0.5; // Capture frame at 0.5s

			video.onloadeddata = () => {
				// Wait for seek
			};

			video.onseeked = () => {
				const canvas = document.createElement('canvas');
				canvas.width = video.videoWidth;
				canvas.height = video.videoHeight;
				const ctx = canvas.getContext('2d');
				if (!ctx) {
					reject(new Error('Failed to get canvas context'));
					return;
				}
				ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
				canvas.toBlob(
					(blob) => {
						if (blob) {
							const file = new File([blob], 'thumbnail.jpg', { type: 'image/jpeg' });
							resolve(file);
						} else {
							reject(new Error('Failed to create blob'));
						}
						URL.revokeObjectURL(video.src);
					},
					'image/jpeg',
					0.8
				);
			};

			video.onerror = (e) => {
				reject(e);
				URL.revokeObjectURL(video.src);
			};
		});
	}

	async function handlePost(privacy: string, allowed: string[], blocked: string[]) {
		if (!selectedFile) return;

		try {
			let filesToUpload = [selectedFile];
			let thumbFile: File | null = null;

			// Generate thumbnail for Reels
			if (activeTab === 'reels') {
				try {
					thumbFile = await generateThumbnail(selectedFile);
					filesToUpload.push(thumbFile);
				} catch (e) {
					console.error('Thumbnail generation failed, using fallback/video url', e);
				}
			}

			// 1. Upload file(s)
			const uploaded = await uploadFiles(filesToUpload);

			if (uploaded && uploaded.length > 0) {
				const mediaUrl = uploaded[0].url;

				if (activeTab === 'stories') {
					const mediaType = selectedFile.type.startsWith('video') ? 'video' : 'image';
					await apiRequest(
						'POST',
						'/stories',
						{
							media_url: mediaUrl,
							media_type: mediaType,
							privacy: privacy,
							allowed_viewers: allowed,
							blocked_viewers: blocked
						},
						true
					);
					// Reset pagination and reload
					storiesOffset = 0;
					storyGroups = [];
					stories = [];
					hasMoreStories = true;
					fetchStories();
				} else {
					// For reels, use the second uploaded file as thumbnail if available, else fallback to video url
					const thumbnailUrl = uploaded.length > 1 ? uploaded[1].url : mediaUrl;

					await apiRequest(
						'POST',
						'/reels',
						{
							video_url: mediaUrl,
							thumbnail_url: thumbnailUrl,
							caption: 'New Reel',
							duration: 0,
							privacy: privacy,
							allowed_viewers: allowed,
							blocked_viewers: blocked
						},
						true
					);
					fetchReels();
				}
			}
		} catch (error) {
			console.error('Failed to create story/reel:', error);
			alert('Failed to upload. Please try again.');
		} finally {
			showComposer = false;
			selectedFile = null;
		}
	}
</script>

<input
	type="file"
	accept={activeTab === 'stories' ? 'image/*,video/*' : 'video/*'}
	class="hidden"
	bind:this={fileInput}
	onchange={handleFileSelect}
/>

{#if showViewer}
	<StoryViewer
		{storyGroups}
		initialGroupIndex={viewingGroupIndex}
		onClose={() => (showViewer = false)}
	/>
{/if}

{#if showComposer && selectedFile}
	<StoryComposer
		file={selectedFile}
		mediaType={selectedFile.type.startsWith('video') ? 'video' : 'image'}
		{activeTab}
		onClose={() => {
			showComposer = false;
			selectedFile = null;
		}}
		onPost={handlePost}
	/>
{/if}

<div class="relative w-full py-2">
	<!-- Tabs -->
	<div class="mb-4 flex space-x-4 px-1">
		<button
			class="flex items-center space-x-2 rounded-full px-4 py-2 text-sm font-semibold transition-all {activeTab ===
			'stories'
				? 'bg-primary/20 text-primary'
				: 'text-muted-foreground hover:bg-black/5'}"
			onclick={() => (activeTab = 'stories')}
		>
			<div
				class="bg-primary/10 flex h-8 w-8 items-center justify-center rounded-full {activeTab ===
				'stories'
					? 'bg-primary text-white'
					: ''}"
			>
				<ImageIcon size={16} />
			</div>
			<span>Stories</span>
		</button>
		<button
			class="flex items-center space-x-2 rounded-full px-4 py-2 text-sm font-semibold transition-all {activeTab ===
			'reels'
				? 'bg-primary/20 text-primary'
				: 'text-muted-foreground hover:bg-black/5'}"
			onclick={() => (activeTab = 'reels')}
		>
			<div
				class="bg-primary/10 flex h-8 w-8 items-center justify-center rounded-full {activeTab ===
				'reels'
					? 'bg-primary text-white'
					: ''}"
			>
				<Film size={16} />
			</div>
			<span>Reels</span>
		</button>
	</div>

	<div
		class="no-scrollbar flex space-x-2 overflow-x-auto pb-2"
		onscroll={(e) => {
			const target = e.target as HTMLDivElement;
			// Check if scrolled near end horizontal
			if (
				target.scrollWidth - target.scrollLeft - target.clientWidth < 200 &&
				activeTab === 'stories' &&
				hasMoreStories &&
				!isLoadingStories
			) {
				fetchStories();
			}
		}}
	>
		<!-- Create Card -->
		<div
			class="glass-card group relative h-48 w-32 flex-shrink-0 cursor-pointer overflow-hidden rounded-xl transition-transform hover:scale-[1.02]"
			onclick={handleCreate}
			role="button"
			tabindex="0"
			onkeydown={(e) => e.key === 'Enter' && handleCreate()}
		>
			<div class="absolute inset-0 bg-gradient-to-b from-transparent to-black/60"></div>
			{#if currentUser}
				<img
					src={currentUser.avatar || 'https://github.com/shadcn.png'}
					alt="Your Story"
					class="h-full w-full object-cover transition-transform duration-500 group-hover:scale-110"
				/>
				<div class="absolute bottom-0 left-0 right-0 flex flex-col items-center p-2">
					<div
						class="bg-primary relative -mt-6 mb-1 flex h-8 w-8 items-center justify-center rounded-full border-4 border-black/20 text-white shadow-lg"
					>
						<Plus size={16} strokeWidth={3} />
					</div>
					<span class="text-xs font-semibold text-white"
						>Create {activeTab === 'stories' ? 'Story' : 'Reel'}</span
					>
				</div>
			{/if}
		</div>

		{#if activeTab === 'stories'}
			<!-- Display FETCHED STORIES Grouped by User -->
			{#each storyGroups as group, i (group.user.id || i)}
				{@const previewStory = group.stories[0]}
				<div
					class="glass-card border-primary/20 group relative h-48 w-32 flex-shrink-0 cursor-pointer overflow-hidden rounded-xl border-2 p-[2px] transition-transform hover:scale-[1.02]"
					onclick={() => openViewer(i)}
					role="button"
					tabindex="0"
					onkeydown={(e) => e.key === 'Enter' && openViewer(i)}
				>
					<!-- Show the latest story (or first) as preview -->
					<div
						class="absolute inset-0 z-10 rounded-lg bg-gradient-to-b from-transparent to-black/60"
					></div>
					{#if previewStory.media_type === 'video'}
						<video
							src={previewStory.media_url}
							class="h-full w-full rounded-lg object-cover transition-transform duration-500 group-hover:scale-110"
							preload="metadata"
							muted
							playsinline
						></video>
					{:else}
						<img
							src={previewStory.media_url}
							alt="Story"
							class="h-full w-full rounded-lg object-cover transition-transform duration-500 group-hover:scale-110"
						/>
					{/if}
					<div
						class="border-primary absolute left-2 top-2 z-20 rounded-full border-2 bg-white p-[2px]"
					>
						<Avatar class="h-8 w-8 border border-gray-200">
							<AvatarImage src={group.user.avatar} />
							<AvatarFallback>{group.user.username?.[0]}</AvatarFallback>
						</Avatar>
					</div>
					<span
						class="absolute bottom-2 left-2 z-20 text-xs font-bold text-white shadow-black/50 drop-shadow-md"
					>
						{group.user.username}
					</span>
				</div>
			{/each}
		{:else}
			<!-- Display REELS (Flat Feed) -->
			{#each reels as reel, i (reel.id)}
				<div
					class="glass-card group relative h-48 w-32 flex-shrink-0 cursor-pointer overflow-hidden rounded-xl transition-transform hover:scale-[1.02]"
					onclick={() => openReelViewer(i)}
					role="button"
					tabindex="0"
					onkeydown={(e) => e.key === 'Enter' && openReelViewer(i)}
				>
					<div class="absolute inset-0 z-10 bg-gradient-to-b from-transparent to-black/60"></div>
					<img
						src={reel.thumbnail_url || 'https://via.placeholder.com/150'}
						alt="Reel"
						class="h-full w-full object-cover transition-transform duration-500 group-hover:scale-110"
					/>
					<div
						class="absolute inset-0 z-20 flex items-center justify-center opacity-0 transition-opacity group-hover:opacity-100"
					>
						<div class="rounded-full bg-black/40 p-2 backdrop-blur-sm">
							<Video size={24} class="text-white" />
						</div>
					</div>
					<div class="absolute bottom-2 left-2 z-20 flex flex-col">
						<span class="text-xs font-bold text-white shadow-black/50 drop-shadow-md"
							>{reel.author?.username}</span
						>
						<span class="line-clamp-1 text-[10px] text-white/80">{reel.views} views</span>
					</div>
				</div>
			{/each}
		{/if}
	</div>
</div>

{#if showReelViewer}
	<ReelViewer {reels} initialIndex={viewingReelIndex} onClose={() => (showReelViewer = false)} />
{/if}

<style>
	.no-scrollbar::-webkit-scrollbar {
		display: none;
	}
	.no-scrollbar {
		-ms-overflow-style: none;
		scrollbar-width: none;
	}
</style>
