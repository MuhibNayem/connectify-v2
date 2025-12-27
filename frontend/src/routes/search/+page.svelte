<script lang="ts">
	import { page } from '$app/stores';
	import { apiRequest } from '$lib/api';
	import { Card, CardHeader, CardTitle, CardContent } from '$lib/components/ui/card';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Button } from '$lib/components/ui/button';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.svelte';
	import { UserPlus, Check, Clock, User } from '@lucide/svelte';

	interface FriendshipStatus {
		are_friends: boolean;
		request_sent: boolean;
		request_received: boolean;
		is_blocked_by_viewer: boolean;
		has_blocked_viewer: boolean;
	}

	interface UserResult {
		id: string;
		username: string;
		full_name?: string;
		avatar?: string;
		friendship_status?: FriendshipStatus;
	}

	let searchQuery: string = '';
	let searchResults: { users: UserResult[]; posts: any[]; total: number } | null = null;
	let loading: boolean = false;
	let error: string | null = null;
	let pendingRequests: Record<string, boolean> = {};

	$: {
		const newQuery = $page.url.searchParams.get('query') || '';
		if (newQuery !== searchQuery) {
			searchQuery = newQuery;
			if (searchQuery) {
				fetchSearchResults(searchQuery);
			} else {
				searchResults = null;
			}
		}
	}

	async function fetchSearchResults(query: string) {
		loading = true;
		error = null;
		try {
			const response = await apiRequest('GET', `/search?query=${encodeURIComponent(query)}`);
			searchResults = response;
			console.log('Search results:', searchResults);
		} catch (err: any) {
			error = err.message || 'Failed to fetch search results.';
			console.error('Search error:', err);
		} finally {
			loading = false;
		}
	}

	async function handleAddFriend(userId: string) {
		if (pendingRequests[userId]) return;

		pendingRequests[userId] = true;
		pendingRequests = { ...pendingRequests };

		try {
			await apiRequest('POST', '/friendships/requests', { receiver_id: userId });
			// Update the embedded friendship status
			if (searchResults?.users) {
				searchResults.users = searchResults.users.map((user) => {
					if (user.id === userId) {
						return {
							...user,
							friendship_status: {
								are_friends: false,
								request_sent: true,
								request_received: false,
								is_blocked_by_viewer: false,
								has_blocked_viewer: false
							}
						};
					}
					return user;
				});
			}
		} catch (err: any) {
			console.error('Failed to send friend request:', err);
			alert(err.message || 'Failed to send friend request');
		} finally {
			pendingRequests[userId] = false;
			pendingRequests = { ...pendingRequests };
		}
	}

	function navigateToUser(userId: string) {
		goto(`/profile/${userId}`);
	}

	function navigateToPost(postId: string) {
		goto(`/post/${postId}`);
	}

	function getButtonState(
		user: UserResult
	): 'self' | 'friends' | 'request_sent' | 'request_received' | 'add' | 'blocked' {
		if (user.id === auth.state.user?.id) return 'self';

		const status = user.friendship_status;
		if (!status) return 'add';

		if (status.has_blocked_viewer || status.is_blocked_by_viewer) return 'blocked';
		if (status.are_friends) return 'friends';
		if (status.request_sent) return 'request_sent';
		if (status.request_received) return 'request_received';

		return 'add';
	}
</script>

<div class="container mx-auto p-4">
	<h1 class="mb-6 text-3xl font-bold">Search Results for "{searchQuery}"</h1>

	{#if loading}
		<p>Loading search results...</p>
	{:else if error}
		<p class="text-red-500">Error: {error}</p>
	{:else if searchResults && searchResults.total === 0}
		<p>No results found for "{searchQuery}".</p>
	{:else if searchResults}
		<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
			<!-- Users Section -->
			<Card>
				<CardHeader>
					<CardTitle>Users ({searchResults?.users?.length})</CardTitle>
				</CardHeader>
				<CardContent>
					{#if searchResults?.users?.length > 0}
						<div class="space-y-4">
							{#each searchResults.users as user}
								{@const buttonState = getButtonState(user)}
								<div class="flex items-center space-x-4 rounded-md border p-2">
									<Avatar>
										<AvatarImage
											src={user.avatar || 'https://github.com/shadcn.png'}
											alt={user.username}
										/>
										<AvatarFallback>{user.username.charAt(0).toUpperCase()}</AvatarFallback>
									</Avatar>
									<div class="flex-grow">
										<p class="font-semibold">{user.username}</p>
										<p class="text-sm text-gray-500">{user.full_name || 'No full name'}</p>
									</div>
									<div class="flex gap-2">
										{#if buttonState === 'self'}
											<!-- Don't show add friend for self -->
										{:else if buttonState === 'friends'}
											<Button variant="outline" disabled class="text-green-600">
												<Check class="mr-1 h-4 w-4" /> Friends
											</Button>
										{:else if buttonState === 'request_sent'}
											<Button variant="outline" disabled class="text-yellow-600">
												<Clock class="mr-1 h-4 w-4" /> Request Sent
											</Button>
										{:else if buttonState === 'request_received'}
											<Button variant="outline" onclick={() => goto('/friends')}>
												<Clock class="mr-1 h-4 w-4" /> Respond
											</Button>
										{:else if buttonState === 'blocked'}
											<Button variant="outline" disabled class="text-red-600">Blocked</Button>
										{:else}
											<Button
												variant="default"
												onclick={() => handleAddFriend(user.id)}
												disabled={pendingRequests[user.id]}
											>
												<UserPlus class="mr-1 h-4 w-4" /> Add Friend
											</Button>
										{/if}
										<Button variant="outline" onclick={() => navigateToUser(user.id)}>
											<User class="mr-1 h-4 w-4" /> Profile
										</Button>
									</div>
								</div>
							{/each}
						</div>
					{:else}
						<p>No users found.</p>
					{/if}
				</CardContent>
			</Card>

			<!-- Posts Section -->
			<Card>
				<CardHeader>
					<CardTitle>Posts ({searchResults?.posts?.length})</CardTitle>
				</CardHeader>
				<CardContent>
					{#if searchResults?.posts?.length > 0}
						<div class="space-y-4">
							{#each searchResults.posts as post}
								<div class="rounded-md border p-2">
									<p class="mb-1 font-semibold">
										{post.content.substring(0, 100)}{post.content.length > 100 ? '...' : ''}
									</p>
									<p class="text-sm text-gray-500">by User ID: {post.user_id}</p>
									<Button variant="link" onclick={() => navigateToPost(post.id)}>Read More</Button>
								</div>
							{/each}
						</div>
					{:else}
						<p>No posts found.</p>
					{/if}
				</CardContent>
			</Card>
		</div>
	{/if}
</div>
