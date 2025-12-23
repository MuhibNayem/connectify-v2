<script lang="ts">
	import { addMessageReaction, removeMessageReaction, type MessageReaction } from '$lib/api';
	import { auth } from '$lib/stores/auth.svelte';

	let { reactions = [], messageId } = $props<{
		reactions: MessageReaction[];
		messageId: string;
	}>();

	// Group reactions by emoji
	let groupedReactions = $derived(() => {
		const groups: Record<string, { count: number; users: string[]; hasCurrentUser: boolean }> = {};

		reactions.forEach((reaction: MessageReaction) => {
			if (!groups[reaction.emoji]) {
				groups[reaction.emoji] = { count: 0, users: [], hasCurrentUser: false };
			}
			groups[reaction.emoji].count++;
			groups[reaction.emoji].users.push(reaction.user_id);
			if (reaction.user_id === auth.state.user?.id) {
				groups[reaction.emoji].hasCurrentUser = true;
			}
		});

		return groups;
	});

	async function toggleReaction(emoji: string, hasCurrentUser: boolean) {
		try {
			if (hasCurrentUser) {
				await removeMessageReaction(messageId, emoji);
			} else {
				await addMessageReaction(messageId, emoji);
			}
		} catch (error) {
			console.error('Failed to toggle reaction:', error);
		}
	}
</script>

{#if reactions && reactions.length > 0}
	<div class="mt-1 flex flex-wrap gap-1">
		{#each Object.entries(groupedReactions()) as [emoji, data]}
			<button
				type="button"
				class="flex items-center gap-1 rounded-full px-2 py-0.5 text-xs transition-all {data.hasCurrentUser
					? 'bg-blue-100 text-blue-700 ring-1 ring-blue-300 hover:bg-blue-200'
					: 'bg-gray-100 text-gray-700 hover:bg-gray-200'}"
				onclick={() => toggleReaction(emoji, data.hasCurrentUser)}
				title={`${data.count} ${data.count === 1 ? 'person' : 'people'} reacted`}
			>
				<span>{emoji}</span>
				<span class="font-semibold">{data.count}</span>
			</button>
		{/each}
	</div>
{/if}
