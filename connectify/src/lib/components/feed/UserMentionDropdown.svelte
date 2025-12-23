<script lang="ts">
	import { apiRequest } from '$lib/api';

	let { query, onSelection } = $props<{
		query: string;
		onSelection: (user: any) => void;
	}>();

	let users = $state<any[]>([]);
	let loading = $state(false);
	let selectedIndex = $state(0);

	async function searchUsers(searchQuery: string) {
		if (!searchQuery) {
			users = [];
			return;
		}
		loading = true;
		try {
			const result = await apiRequest('GET', `/users?search=${searchQuery}&limit=5`);
			users = result.users || [];
			selectedIndex = 0;
		} catch (error) {
			console.error('Failed to search users:', error);
			users = [];
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		searchUsers(query);
	});

	$effect(() => {
		function handleListKeydown(event: KeyboardEvent) {
			if (users.length === 0) return;

			if (event.key === 'ArrowDown') {
				event.preventDefault();
				selectedIndex = (selectedIndex + 1) % users.length;
			} else if (event.key === 'ArrowUp') {
				event.preventDefault();
				selectedIndex = (selectedIndex - 1 + users.length) % users.length;
			            } else if (event.key === 'Enter') {
			                event.preventDefault();
			                event.stopPropagation(); // Prevent event from bubbling to the textarea
			                onSelection(users[selectedIndex]);
			            }		}

		window.addEventListener('keydown', handleListKeydown);

		return () => {
			window.removeEventListener('keydown', handleListKeydown);
		};
	});
</script>

{#if loading}
	<div class="p-2 text-sm text-gray-500">Searching...</div>
{:else if users.length > 0}
	<ul class="max-h-60 overflow-y-auto rounded-md border border-gray-200 bg-white shadow-lg">
		{#each users as user, i}
			<li>
				<button
					type="button"
					class="w-full p-2 text-left hover:bg-gray-100 {i === selectedIndex ? 'bg-gray-100' : ''}"
					onclick={(e) => {
						e.preventDefault();
						onSelection(user);
					}}
				>
					<div class="flex items-center space-x-2">
						<img
							src={user.avatar || 'https://github.com/shadcn.png'}
							alt={user.username}
							class="h-6 w-6 rounded-full"
						/>
						<span class="font-semibold">{user.username}</span>
					</div>
				</button>
			</li>
		{/each}
	</ul>
{:else if query}
	<div class="p-2 text-sm text-gray-500">No users found.</div>
{/if}
