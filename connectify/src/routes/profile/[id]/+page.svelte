<script lang="ts">
	import { page } from '$app/stores';
	import { apiRequest, uploadFiles, updateUserProfile } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Button } from '$lib/components/ui/button';
	import PostCard from '$lib/components/feed/PostCard.svelte';
	import PostCreator from '$lib/components/feed/PostCreator.svelte';
	import MediaViewer from '$lib/components/ui/MediaViewer.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import MediaSelector from '$lib/components/album/MediaSelector.svelte';
	import {
		Camera,
		MapPin,
		Link as LinkIcon,
		Calendar,
		MoreHorizontal,
		Folder,
		Video,
		Plus,
		X,
		ChevronLeft as IconChevronLeft
	} from '@lucide/svelte';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';

	let userId = $state('');
	let user = $state<any | null>(null);
	let posts = $state<any[]>([]);
	let loadingUser = $state(true);
	let loadingPosts = $state(true);
	let userError = $state<string | null>(null);
	let postsError = $state<string | null>(null);
	let friendshipStatus = $state<'none' | 'pending' | 'friends' | 'blocked'>('none');
	let sendingRequest = $state(false);
	let isCurrentUser = $state(false);
	let activeTab = $state('posts');
	let friends = $state<any[]>([]);
	let loadingFriends = $state(false);

	// File Inputs
	let avatarInput: HTMLInputElement;
	let coverInput: HTMLInputElement;
	let isUploading = $state(false);

	// Media Viewer
	let mediaViewerOpen = $state(false);
	let mediaViewerItems = $state<any[]>([]);
	let mediaViewerIndex = $state(0);

	// Albums
	let albums = $state<any[]>([]);
	let loadingAlbums = $state(false);
	let selectedAlbum = $state<any | null>(null); // If null, show album list. If set, show album content.
	let albumMedia = $state<any[]>([]);
	let loadingAlbumMedia = $state(false);
	let createAlbumDialogOpen = $state(false);
	let newAlbumName = $state('');
	let newAlbumDesc = $state('');
	let newAlbumPrivacy = $state<'PUBLIC' | 'FRIENDS' | 'ONLY_ME'>('PUBLIC');

	// Videos
	let userVideos = $state<any[]>([]);
	let loadingVideos = $state(false);

	async function sendFriendRequest() {
		sendingRequest = true;
		try {
			await apiRequest('POST', '/friendships/requests', { receiver_id: userId });
			friendshipStatus = 'pending';
		} catch (err: any) {
			console.error('Failed to send friend request:', err);
		} finally {
			sendingRequest = false;
		}
	}

	$effect(() => {
		userId = $page.params.id ?? '';
		if (userId) {
			isCurrentUser = auth.state.user?.id === userId;
			fetchUserProfile(userId);
			fetchUserPosts(userId);
			fetchUserFriends(userId);
			fetchUserPhotos(userId); // For sidebar
			activeTab = 'posts';
			selectedAlbum = null; // Reset album view
		}
	});

	async function handleFileChange(type: 'avatar' | 'cover', event: Event) {
		const input = event.target as HTMLInputElement;
		if (!input.files || input.files.length === 0) return;

		const file = input.files[0];
		isUploading = true;

		try {
			// 1. Upload File
			const uploadResult = await uploadFiles([file]);
			if (!uploadResult || uploadResult.length === 0) throw new Error('Upload failed');

			const fileUrl = uploadResult[0].url;

			// 2. Update User Profile
			const payload: any = {};
			if (type === 'avatar') payload.avatar = fileUrl;
			if (type === 'cover') payload.cover_picture = fileUrl;

			const updatedUser = await updateUserProfile(payload);

			// 3. Update Local State (Visually immediate)
			user = { ...user, ...updatedUser }; // Merge cleanly
			if (isCurrentUser) {
				auth.updateUser(user);
			}

			// Refresh posts/albums as new post is created in backend
			setTimeout(() => {
				fetchUserPosts(userId);
				if (activeTab === 'photos') fetchUserAlbums(userId);
			}, 1000);
		} catch (error) {
			console.error(`Failed to update ${type}:`, error);
		} finally {
			isUploading = false;
			input.value = '';
		}
	}

	async function fetchUserProfile(id: string) {
		loadingUser = true;
		userError = null;
		try {
			user = await apiRequest('GET', `/users/${id}`);

			if (!isCurrentUser && auth.state.user) {
				try {
					const friendshipCheck = await apiRequest('GET', `/friendships/check?other_user_id=${id}`);
					if (friendshipCheck.are_friends) {
						friendshipStatus = 'friends';
					} else if (friendshipCheck.request_sent) {
						friendshipStatus = 'pending';
					} else if (friendshipCheck.is_blocked_by_viewer || friendshipCheck.has_blocked_viewer) {
						friendshipStatus = 'blocked';
					} else if (friendshipCheck.request_received) {
						friendshipStatus = 'pending';
					} else {
						friendshipStatus = 'none';
					}
				} catch (err: any) {
					console.warn('Could not check friendship status:', err);
				}
			}
		} catch (err: any) {
			userError = err.message || 'Failed to fetch user profile.';
		} finally {
			loadingUser = false;
		}
	}

	async function fetchUserPosts(id: string) {
		loadingPosts = true;
		postsError = null;
		try {
			const response = await apiRequest('GET', `/posts?user_id=${id}`);
			posts = response.posts || [];
		} catch (err: any) {
			postsError = err.message || 'Failed to fetch user posts.';
		} finally {
			loadingPosts = false;
		}
	}

	async function fetchUserFriends(id: string) {
		loadingFriends = true;
		try {
			const response = await apiRequest('GET', `/friendships?user_id=${id}&status=accepted`);
			const acceptedFriendships: any[] = response.data;

			if (!acceptedFriendships) {
				friends = [];
				return;
			}
			const friendUserPromises = acceptedFriendships.map(async (friendship) => {
				const friendId =
					friendship.requester_id === id ? friendship.receiver_id : friendship.requester_id;
				try {
					const friendDetails = await apiRequest('GET', `/users/${friendId}`);
					return {
						id: friendDetails.id,
						username: friendDetails.username,
						avatar: friendDetails.avatar,
						full_name: friendDetails.full_name
					};
				} catch (e) {
					return null;
				}
			});
			const results = await Promise.all(friendUserPromises);
			friends = results.filter((f) => f !== null);
		} catch (err) {
			console.error('Failed to fetch friends', err);
		} finally {
			loadingFriends = false;
		}
	}

	let userPhotos = $state<any[]>([]); // For sidebar
	let loadingPhotos = $state(false);

	async function fetchUserPhotos(id: string) {
		loadingPhotos = true;
		try {
			const response = await apiRequest(
				'GET',
				`/posts?user_id=${id}&has_media=true&media_type=image&limit=9`
			);
			const photos: any[] = [];
			const postsWithMedia = response.posts || [];
			postsWithMedia.forEach((post: any) => {
				if (post.media) {
					post.media.forEach((item: any) => {
						if (item.type === 'image') photos.push(item);
					});
				}
			});
			userPhotos = photos;
		} catch (err) {
			console.error('Failed to fetch user photos:', err);
		} finally {
			loadingPhotos = false;
		}
	}

	async function fetchUserAlbums(id: string) {
		loadingAlbums = true;
		try {
			const data = await apiRequest('GET', `/users/${id}/albums`);
			albums = data || [];
		} catch (err) {
			console.error('Failed to fetch albums', err);
		} finally {
			loadingAlbums = false;
		}
	}

	// Album Media Pagination
	let albumPage = $state(1);
	const ALBUM_LIMIT = 50;
	let albumHasMore = $state(true);

	async function fetchAlbumMedia(albumId: string, reset = false) {
		loadingAlbumMedia = true;
		if (reset) {
			albumMedia = [];
			albumPage = 1;
			albumHasMore = true;
		}

		try {
			const response = await apiRequest(
				'GET',
				`/albums/${albumId}/media?limit=${ALBUM_LIMIT}&page=${albumPage}`
			);

			// Handle page-based pagination format
			const media = response.media || [];
			const total = response.total || 0;

			if (media && media.length > 0) {
				albumMedia = [...albumMedia, ...media];
				albumPage++;
				// Calculate hasMore: if we got less than limit OR current items >= total
				albumHasMore = media.length >= ALBUM_LIMIT && albumMedia.length < total;
			} else {
				albumHasMore = false;
			}
		} catch (err) {
			console.error('Failed to fetch album media', err);
		} finally {
			loadingAlbumMedia = false;
		}
	}

	let albumSentinel = $state<HTMLElement>();

	$effect(() => {
		if (selectedAlbum && albumSentinel && albumHasMore && !loadingAlbumMedia) {
			const observer = new IntersectionObserver(
				(entries) => {
					if (entries[0].isIntersecting && albumHasMore && !loadingAlbumMedia) {
						fetchAlbumMedia(selectedAlbum.id);
					}
				},
				{ threshold: 0.1, rootMargin: '100px' }
			);
			observer.observe(albumSentinel);
			return () => observer.disconnect();
		}
	});

	function openAlbum(album: any) {
		selectedAlbum = album;
		fetchAlbumMedia(album.id, true);
	}

	async function fetchUserVideos(id: string) {
		loadingVideos = true;
		try {
			const response = await apiRequest(
				'GET',
				`/posts?user_id=${id}&has_media=true&media_type=video&limit=50`
			);
			const videos: any[] = [];
			const postsWithMedia = response.posts || [];
			postsWithMedia.forEach((post: any) => {
				if (post.media) {
					post.media.forEach((item: any) => {
						if (item.type === 'video') videos.push(item);
					});
				}
			});
			userVideos = videos;
		} catch (err) {
			console.error('Failed to fetch videos', err);
		} finally {
			loadingVideos = false;
		}
	}

	async function createAlbum() {
		if (!newAlbumName) return;
		try {
			const newAlbum = await apiRequest('POST', '/albums', {
				name: newAlbumName,
				description: newAlbumDesc,
				privacy: newAlbumPrivacy
			});
			albums = [newAlbum, ...albums];
			createAlbumDialogOpen = false;
			newAlbumName = '';
			newAlbumDesc = '';
			newAlbumPrivacy = 'PUBLIC';
		} catch (err) {
			console.error('Failed to create album', err);
		}
	}

	$effect(() => {
		if (activeTab === 'photos' && userId) {
			fetchUserAlbums(userId);
			selectedAlbum = null;
			albumMedia = [];
		}
		if (activeTab === 'videos' && userId) {
			fetchUserVideos(userId);
		}
	});

	function handlePostCreated(event: CustomEvent) {
		posts = [event.detail, ...posts];
	}

	let mediaViewerOnReachEnd = $state<(() => void) | undefined>(undefined);

	function openMediaViewer(items: any[], index: number, onReachEnd?: () => void) {
		mediaViewerItems = items;
		mediaViewerIndex = index;
		mediaViewerOnReachEnd = onReachEnd;
		mediaViewerOpen = true;
	}

	// Media Selector Logic
	let mediaSelectorOpen = $state(false);
	let addingMedia = $state(false);

	async function handleAddMediaToAlbum(selectedItems: any[]) {
		if (!selectedAlbum) return;
		addingMedia = true;
		try {
			// Prepare payload: array of { url, type }
			const mediaPayload = selectedItems.map((item) => ({
				url: item.url,
				type: item.type
			}));

			await apiRequest('POST', `/albums/${selectedAlbum.id}/media`, {
				media: mediaPayload
			});

			// Refresh album media
			await fetchAlbumMedia(selectedAlbum.id, true);

			// Optimistically update cover if empty
			if (!selectedAlbum.cover_url && mediaPayload.length > 0) {
				fetchUserAlbums(userId); // Refresh album list to get new cover
				selectedAlbum = { ...selectedAlbum, cover_url: mediaPayload[0].url };
			}
		} catch (err) {
			console.error('Failed to add media to album', err);
			alert('Failed to add photos to album.');
		} finally {
			addingMedia = false;
		}
	}
	import { goto } from '$app/navigation';

	async function handleMessage() {
		if (!auth.state.user?.id || !user?.id) return;
		await goto(`/messages/user-${user.id}`);
	}
</script>

{#if loadingUser}
	<div class="flex h-[50vh] items-center justify-center">
		<div
			class="border-primary h-8 w-8 animate-spin rounded-full border-4 border-t-transparent"
		></div>
	</div>
{:else if userError}
	<div class="container mx-auto p-4 text-center">
		<p class="text-red-500">Error: {userError}</p>
		<Button href="/" variant="link">Go Home</Button>
	</div>
{:else if user}
	<MediaViewer
		open={mediaViewerOpen}
		media={mediaViewerItems}
		initialIndex={mediaViewerIndex}
		onClose={() => (mediaViewerOpen = false)}
		onReachEnd={mediaViewerOnReachEnd}
	/>

	<!-- Create Album Dialog -->
	<!-- Create Album Dialog Custom -->
	<!-- Create Album Dialog -->
	<Dialog.Root bind:open={createAlbumDialogOpen}>
		<Dialog.Content>
			<Dialog.Header>
				<Dialog.Title>Create Album</Dialog.Title>
				<Dialog.Description>Create a new album to organize your photos.</Dialog.Description>
			</Dialog.Header>
			<div class="space-y-4 py-4">
				<div class="space-y-2">
					<Label>Name</Label>
					<Input bind:value={newAlbumName} placeholder="Album Name" />
				</div>
				<div class="space-y-2">
					<Label>Description</Label>
					<Input bind:value={newAlbumDesc} placeholder="Description (optional)" />
				</div>
				<div class="space-y-2">
					<Label>Privacy</Label>
					<select
						bind:value={newAlbumPrivacy}
						class="border-input bg-background ring-offset-background focus:ring-ring w-full rounded-md border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-offset-2"
					>
						<option value="PUBLIC">🌍 Public - Anyone can see</option>
						<option value="FRIENDS">👥 Friends - Only friends can see</option>
						<option value="ONLY_ME">🔒 Only Me - Private</option>
					</select>
				</div>
			</div>
			<Dialog.Footer>
				<Button variant="ghost" onclick={() => (createAlbumDialogOpen = false)}>Cancel</Button>
				<Button onclick={createAlbum} disabled={!newAlbumName}>Create</Button>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>

	<div class="bg-card pb-4 shadow-sm">
		<div class="mx-auto max-w-[1095px]">
			<!-- Cover Photo -->
			<div class="relative h-[350px] w-full overflow-hidden rounded-b-xl bg-gray-200 md:h-[400px]">
				{#if user.cover_picture}
					<img
						src={user.cover_picture}
						alt="Cover"
						class="h-full w-full cursor-pointer object-cover"
						onclick={() => openMediaViewer([{ type: 'image', url: user.cover_picture }], 0)}
					/>
				{:else}
					<div
						class="flex h-full w-full items-center justify-center bg-gradient-to-r from-gray-100 to-gray-200 text-gray-400"
					></div>
				{/if}

				{#if isCurrentUser}
					<button
						class="absolute bottom-4 right-4 flex items-center gap-2 rounded-md bg-white px-3 py-2 text-sm font-semibold text-black shadow-sm hover:bg-gray-100"
						onclick={() => coverInput.click()}
						disabled={isUploading}
					>
						<Camera size={18} />
						<span>Edit cover photo</span>
					</button>
					<input
						type="file"
						accept="image/*"
						bind:this={coverInput}
						onchange={(e) => handleFileChange('cover', e)}
						hidden
					/>
				{/if}
			</div>

			<!-- Profile Info Header -->
			<div class="mx-auto max-w-[1030px] px-4 pb-4">
				<div class="relative flex flex-col items-center md:flex-row md:items-end md:gap-6">
					<!-- Avatar -->
					<div class="relative -mt-20 md:-mt-8">
						<div class="bg-card relative rounded-full p-1">
							<Avatar
								class="border-card h-[168px] w-[168px] cursor-pointer border-4 bg-white object-cover"
								onclick={() =>
									openMediaViewer(
										[{ type: 'image', url: user.avatar || 'https://github.com/shadcn.png' }],
										0
									)}
							>
								<AvatarImage
									src={user.avatar || 'https://github.com/shadcn.png'}
									alt={user.username}
									class="object-cover"
								/>
								<AvatarFallback class="text-6xl"
									>{user.username?.charAt(0).toUpperCase()}</AvatarFallback
								>
							</Avatar>
							{#if isCurrentUser}
								<button
									class="border-card absolute bottom-2 right-2 flex h-9 w-9 cursor-pointer items-center justify-center rounded-full border-2 bg-gray-200 text-black hover:bg-gray-300"
									onclick={(e) => {
										e.stopPropagation();
										avatarInput.click();
									}}
									disabled={isUploading}
								>
									<Camera size={20} />
								</button>
								<input
									type="file"
									accept="image/*"
									bind:this={avatarInput}
									onchange={(e) => handleFileChange('avatar', e)}
									hidden
								/>
							{/if}
						</div>
					</div>

					<!-- Info -->
					<div class="mb-4 mt-2 flex-grow text-center md:mb-8 md:mt-0 md:text-left">
						<h1 class="text-3xl font-bold">{user.full_name || user.username}</h1>
						{#if user.full_name}<p class="text-muted-foreground font-semibold">
								@{user.username}
							</p>{/if}
						{#if user.bio}
							<p class="text-muted-foreground mx-auto mt-1 max-w-md md:mx-0">{user.bio}</p>
						{/if}

						{#if friendshipStatus === 'friends'}
							<p class="text-muted-foreground mt-1 text-sm font-semibold">
								{friends.length} friends
							</p>
						{/if}
					</div>

					<!-- Actions -->
					<div class="mb-8 flex flex-col gap-2 md:flex-row">
						{#if !isCurrentUser}
							{#if friendshipStatus === 'none'}
								<Button onclick={sendFriendRequest} disabled={sendingRequest}>
									{sendingRequest ? 'Sending...' : 'Add Friend'}
								</Button>
							{:else if friendshipStatus === 'pending'}
								<Button variant="secondary" disabled>Request Sent</Button>
							{:else if friendshipStatus === 'friends'}
								<Button variant="secondary">Friends</Button>
								<!-- Message Button (Only for friends) -->
								<Button variant="secondary" onclick={handleMessage}>Message</Button>
							{:else if friendshipStatus === 'blocked'}
								<Button variant="destructive" disabled>Blocked</Button>
							{/if}
						{:else}
							<Button variant="secondary" class="bg-secondary/50 font-semibold" href="/settings">
								<div class="mr-2">✏️</div>
								Edit profile
							</Button>
						{/if}
					</div>
				</div>

				<hr class="border-border/40 my-1" />

				<!-- Navigation Tabs -->
				<div class="flex items-center gap-1 overflow-x-auto py-1">
					<Button
						variant="ghost"
						class={`hover:bg-secondary/50 h-12 rounded-lg px-4 font-semibold ${activeTab === 'posts' ? 'text-primary border-primary rounded-none border-b-2' : 'text-muted-foreground'}`}
						onclick={() => (activeTab = 'posts')}
					>
						Posts
					</Button>
					<Button
						variant="ghost"
						class={`hover:bg-secondary/50 h-12 rounded-lg px-4 font-semibold ${activeTab === 'about' ? 'text-primary border-primary rounded-none border-b-2' : 'text-muted-foreground'}`}
						onclick={() => (activeTab = 'about')}
					>
						About
					</Button>
					<Button
						variant="ghost"
						class={`hover:bg-secondary/50 h-12 rounded-lg px-4 font-semibold ${activeTab === 'friends' ? 'text-primary border-primary rounded-none border-b-2' : 'text-muted-foreground'}`}
						onclick={() => (activeTab = 'friends')}
					>
						Friends
					</Button>
					<Button
						variant="ghost"
						class={`hover:bg-secondary/50 h-12 rounded-lg px-4 font-semibold ${activeTab === 'photos' ? 'text-primary border-primary rounded-none border-b-2' : 'text-muted-foreground'}`}
						onclick={() => (activeTab = 'photos')}
					>
						Photos
					</Button>

					<DropdownMenu.Root>
						<DropdownMenu.Trigger>
							<Button
								variant="ghost"
								class={`hover:bg-secondary/50 h-12 rounded-lg px-4 font-semibold ${activeTab === 'videos' ? 'text-primary border-primary rounded-none border-b-2' : 'text-muted-foreground'}`}
							>
								More <MoreHorizontal size={16} class="ml-1" />
							</Button>
						</DropdownMenu.Trigger>
						<DropdownMenu.Content align="start">
							<DropdownMenu.Item onclick={() => (activeTab = 'videos')}>
								<Video class="mr-2 h-4 w-4" />
								<span>Videos</span>
							</DropdownMenu.Item>
						</DropdownMenu.Content>
					</DropdownMenu.Root>
				</div>
			</div>
		</div>
	</div>

	<!-- Main Content -->
	<div class="mx-auto max-w-[1095px] px-4 py-4">
		{#if activeTab === 'posts'}
			<div class="grid grid-cols-1 gap-4 md:grid-cols-12">
				<!-- Left Sidebar (Intro, Photos, Friends) -->
				<div class="space-y-4 md:col-span-5">
					<!-- Intro Card -->
					<div class="glass-card bg-card rounded-xl border border-white/5 p-4 shadow-sm">
						<h2 class="mb-4 text-xl font-bold">Intro</h2>
						{#if user.bio}
							<div class="mb-4 text-center">
								<p class="text-sm">{user.bio}</p>
							</div>
						{/if}
						<div class="mb-4 space-y-3">
							{#if user.location}
								<div class="text-muted-foreground flex items-center gap-2">
									<MapPin size={20} />
									<span
										>Lives in <span class="text-foreground font-semibold">{user.location}</span
										></span
									>
								</div>
							{/if}
							{#if user.website}
								<div class="text-muted-foreground flex items-center gap-2">
									<LinkIcon size={20} />
									<a
										href={user.website}
										target="_blank"
										class="truncate text-blue-500 hover:underline">{user.website}</a
									>
								</div>
							{/if}
							<div class="text-muted-foreground flex items-center gap-2">
								<Calendar size={20} />
								<span>Joined {new Date(user.created_at).toLocaleDateString()}</span>
							</div>
						</div>

						<Button variant="secondary" class="bg-secondary/50 mb-3 w-full font-semibold"
							>Edit details</Button
						>
					</div>

					<!-- Photos Widget -->
					<div class="glass-card bg-card rounded-xl border border-white/5 p-4 shadow-sm">
						<div class="mb-2 flex items-center justify-between">
							<h2 class="text-xl font-bold">Photos</h2>
							<Button variant="link" class="text-primary p-0" onclick={() => (activeTab = 'photos')}
								>See all photos</Button
							>
						</div>
						<div class="grid grid-cols-3 gap-1 overflow-hidden rounded-lg">
							{#if userPhotos.length > 0}
								{#each userPhotos.slice(0, 9) as photo, i}
									<div
										class="bg-secondary aspect-square cursor-pointer"
										onclick={() => openMediaViewer(userPhotos, i)}
									>
										<img src={photo.url} alt="User photo" class="h-full w-full object-cover" />
									</div>
								{/each}
							{:else}
								<div class="text-muted-foreground col-span-3 py-4 text-center text-xs">
									No photos
								</div>
							{/if}
						</div>
					</div>

					<!-- Friends Widget -->
					<div class="glass-card bg-card rounded-xl border border-white/5 p-4 shadow-sm">
						<div class="mb-2 flex items-center justify-between">
							<h2 class="text-xl font-bold">Friends</h2>
							<Button
								variant="link"
								class="text-primary p-0"
								onclick={() => (activeTab = 'friends')}>See all friends</Button
							>
						</div>
						<div class="text-muted-foreground mb-1 text-sm">{friends.length} friends</div>
						<div class="grid grid-cols-3 gap-2">
							{#each friends.slice(0, 9) as friend}
								<a href="/profile/{friend.id}" class="group">
									<div class="bg-secondary mb-1 aspect-square overflow-hidden rounded-lg">
										{#if friend.avatar}
											<img
												src={friend.avatar}
												alt={friend.username}
												class="h-full w-full object-cover transition group-hover:scale-105"
											/>
										{:else}
											<div class="flex h-full w-full items-center justify-center text-xs">👤</div>
										{/if}
									</div>
									<div class="truncate text-[11px] font-semibold group-hover:underline">
										{friend.full_name || friend.username}
									</div>
								</a>
							{/each}
							{#if friends.length === 0 && !loadingFriends}
								<div class="text-muted-foreground col-span-3 py-4 text-center text-xs">
									No friends found
								</div>
							{/if}
						</div>
					</div>
				</div>

				<!-- Right Feed -->
				<div class="space-y-4 md:col-span-7">
					{#if isCurrentUser}
						<div class="mb-4">
							<PostCreator on:postCreated={handlePostCreated} communityId="" />
						</div>
					{/if}

					<!-- Filters Widget -->
					<div
						class="glass-card bg-card flex items-center justify-between rounded-xl border border-white/5 p-4 shadow-sm"
					>
						<h3 class="text-lg font-bold">Posts</h3>
						<div class="flex gap-2">
							<Button variant="secondary" size="sm" class="bg-secondary/50"
								><div class="mr-1">⚙️</div>
								Filters</Button
							>
							<Button variant="secondary" size="sm" class="bg-secondary/50"
								><div class="mr-1">⚙️</div>
								Manage posts</Button
							>
						</div>
					</div>

					{#if loadingPosts}
						<div class="space-y-4">
							<!-- Skeleton Loaders -->
							<div class="bg-card h-40 animate-pulse rounded-xl"></div>
							<div class="bg-card h-40 animate-pulse rounded-xl"></div>
						</div>
					{:else if postsError}
						<p class="text-red-500">Error: {postsError}</p>
					{:else if posts.length === 0}
						<div
							class="glass-card bg-card rounded-xl border border-white/5 p-8 text-center shadow-sm"
						>
							<h3 class="mb-2 text-xl font-bold">No posts available</h3>
							<p class="text-muted-foreground">This user hasn't posted anything yet.</p>
						</div>
					{:else}
						<div class="space-y-4">
							{#each posts as post (post.id)}
								<PostCard
									{post}
									on:viewMedia={(e) => openMediaViewer(e.detail.media, e.detail.index)}
								/>
							{/each}
						</div>
					{/if}
				</div>
			</div>
		{:else if activeTab === 'friends'}
			<div class="glass-card bg-card rounded-xl border border-white/5 p-4 shadow-sm">
				<h2 class="mb-4 text-2xl font-bold">Friends</h2>

				{#if loadingFriends}
					<div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
						{#each Array(6) as _}
							<div class="bg-secondary/50 h-24 animate-pulse rounded-xl"></div>
						{/each}
					</div>
				{:else if friends.length === 0}
					<div class="text-muted-foreground p-8 text-center">No friends found.</div>
				{:else}
					<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
						{#each friends as friend}
							<div
								class="bg-secondary/10 hover:bg-secondary/20 flex items-center justify-between rounded-xl border border-white/5 p-4 transition"
							>
								<a href="/profile/{friend.id}" class="flex items-center gap-3">
									<div class="bg-secondary h-20 w-20 overflow-hidden rounded-lg">
										{#if friend.avatar}
											<img
												src={friend.avatar}
												alt={friend.username}
												class="h-full w-full object-cover"
											/>
										{:else}
											<div class="flex h-full w-full items-center justify-center">👤</div>
										{/if}
									</div>
									<div>
										<h3 class="text-lg font-bold hover:underline">
											{friend.full_name || friend.username}
										</h3>
										<p class="text-muted-foreground text-sm">@{friend.username}</p>
									</div>
								</a>
								<Button variant="secondary">Friend</Button>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		{:else if activeTab === 'photos'}
			<div class="glass-card bg-card min-h-[500px] rounded-xl border border-white/5 p-4 shadow-sm">
				{#if selectedAlbum}
					<!-- Album Content View -->
					<div class="mb-4 flex items-center justify-between gap-4">
						<div class="flex items-center gap-4">
							<Button
								variant="secondary"
								onclick={() => {
									selectedAlbum = null;
									albumMedia = [];
								}}
							>
								<IconChevronLeft class="mr-2" size={20} /> Back to Albums
							</Button>
							<div>
								<h2 class="text-2xl font-bold">{selectedAlbum.name}</h2>
								{#if selectedAlbum.description}
									<p class="text-muted-foreground text-sm">{selectedAlbum.description}</p>
								{/if}
							</div>
						</div>
						{#if isCurrentUser && selectedAlbum.type === 'custom'}
							<Button onclick={() => (mediaSelectorOpen = true)} disabled={addingMedia}>
								{#if addingMedia}
									<div
										class="mr-2 h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
									></div>
									Adding...
								{:else}
									<Plus size={18} class="mr-2" /> Add Photos
								{/if}
							</Button>
						{/if}
					</div>

					{#if loadingAlbumMedia}
						<div class="grid grid-cols-2 gap-2 md:grid-cols-3 lg:grid-cols-4">
							{#each Array(8) as _}
								<div class="bg-secondary/50 aspect-square animate-pulse rounded-lg"></div>
							{/each}
						</div>
					{:else if albumMedia.length > 0}
						<div class="grid grid-cols-2 gap-2 md:grid-cols-3 lg:grid-cols-4">
							{#each albumMedia as item, i}
								<div
									class="group relative aspect-square cursor-pointer overflow-hidden rounded-lg bg-black"
									onclick={() =>
										openMediaViewer(albumMedia, i, async () => {
											if (!selectedAlbum) return;
											await fetchAlbumMedia(selectedAlbum.id);
											mediaViewerItems = albumMedia;
										})}
								>
									{#if item.type === 'image'}
										<img
											src={item.url}
											alt="Album media"
											class="h-full w-full object-cover transition duration-300 group-hover:scale-110 group-hover:opacity-90"
										/>
									{:else if item.type === 'video'}
										<video
											src={item.url}
											class="h-full w-full object-cover transition duration-300 group-hover:scale-110 group-hover:opacity-90"
										></video>
										<div class="absolute inset-0 flex items-center justify-center">
											<div class="rounded-full bg-black/50 p-2 text-white">▶</div>
										</div>
									{/if}
								</div>
							{/each}
						</div>

						<!-- Infinite Scroll Sentinel -->
						{#if albumHasMore}
							<div bind:this={albumSentinel} class="py-4 text-center">
								{#if loadingAlbumMedia}
									<div class="text-muted-foreground">Loading more...</div>
								{/if}
							</div>
						{/if}
					{:else}
						<div class="text-muted-foreground col-span-full py-12 text-center">
							<div
								class="bg-secondary/30 mx-auto mb-3 flex h-16 w-16 items-center justify-center rounded-full"
							>
								<Folder size={32} class="opacity-50" />
							</div>
							<p>This album is empty.</p>
							{#if isCurrentUser && selectedAlbum.type === 'custom'}
								<Button variant="outline" class="mt-4" onclick={() => (mediaSelectorOpen = true)}
									>Add Photos</Button
								>
							{/if}
						</div>
					{/if}
				{:else}
					<!-- Albums List View -->
					<div class="mb-4 flex items-center justify-between">
						<h2 class="text-2xl font-bold">Albums</h2>
						{#if isCurrentUser}
							<Button onclick={() => (createAlbumDialogOpen = true)}
								><Plus class="mr-2 h-4 w-4" /> Create Album</Button
							>
						{/if}
					</div>
					{#if loadingAlbums}
						<div class="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4">
							{#each Array(4) as _}
								<div class="bg-secondary/50 aspect-square animate-pulse rounded-lg"></div>
							{/each}
						</div>
					{:else if albums.length === 0}
						<div class="text-muted-foreground p-8 text-center">No albums found.</div>
					{:else}
						<div class="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4">
							{#each albums as album}
								<div class="group cursor-pointer space-y-2" onclick={() => openAlbum(album)}>
									<div
										class="bg-secondary relative aspect-square overflow-hidden rounded-lg border border-white/5"
									>
										{#if album.cover_url}
											<img
												src={album.cover_url}
												alt={album.name}
												class="h-full w-full object-cover transition group-hover:scale-105"
											/>
										{:else}
											<div
												class="text-muted-foreground flex h-full w-full flex-col items-center justify-center"
											>
												<Folder size={40} strokeWidth={1.5} />
											</div>
										{/if}
										<div
											class="absolute inset-0 bg-black/0 transition group-hover:bg-black/10"
										></div>
									</div>
									<div>
										<h3 class="truncate font-semibold leading-tight group-hover:underline">
											{album.name}
										</h3>
										<p class="text-muted-foreground text-xs capitalize">
											{album.type === 'custom' ? 'Album' : album.type.replace('_', ' ')}
										</p>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				{/if}
			</div>
		{:else if activeTab === 'videos'}
			<div class="glass-card bg-card rounded-xl border border-white/5 p-4 shadow-sm">
				<h2 class="mb-4 text-2xl font-bold">Videos</h2>
				{#if loadingVideos}
					<div class="grid grid-cols-2 gap-2 md:grid-cols-3 lg:grid-cols-4">
						{#each Array(4) as _}
							<div class="bg-secondary/50 aspect-square animate-pulse rounded-lg"></div>
						{/each}
					</div>
				{:else if userVideos.length === 0}
					<div class="text-muted-foreground p-8 text-center">No videos found.</div>
				{:else}
					<div class="grid grid-cols-2 gap-2 md:grid-cols-3 lg:grid-cols-4">
						{#each userVideos as video}
							<div
								class="group relative aspect-square cursor-pointer overflow-hidden rounded-lg bg-black"
								onclick={() => openMediaViewer([video], 0)}
							>
								<video
									src={video.url}
									class="h-full w-full object-cover transition duration-300 group-hover:scale-110 group-hover:opacity-75"
								></video>
								<div class="absolute inset-0 flex items-center justify-center">
									<div class="rounded-full bg-black/50 p-2 text-white">▶</div>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		{/if}
	</div>
{:else}
	<p>User not found.</p>
{/if}

<!-- Media Selector Component -->
<MediaSelector
	bind:open={mediaSelectorOpen}
	userId={auth.state.user?.id || ''}
	onSelect={handleAddMediaToAlbum}
/>
