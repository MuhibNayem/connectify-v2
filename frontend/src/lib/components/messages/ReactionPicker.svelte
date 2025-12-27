<script lang="ts">
	import { addMessageReaction } from '$lib/api';
	import { createEventDispatcher, onMount } from 'svelte';

	let { messageId } = $props<{ messageId: string }>();

	const dispatch = createEventDispatcher();

	onMount(async () => {
		if (typeof window !== 'undefined') {
			await import('emoji-picker-element');
		}
	});

	function setupEmojiPicker(node: HTMLElement) {
		const handleEmojiClick = async (event: any) => {
			if (event.detail && event.detail.unicode) {
				try {
					await addMessageReaction(messageId, event.detail.unicode);
					dispatch('close');
				} catch (error) {
					console.error('Failed to add reaction:', error);
				}
			}
		};
		node.addEventListener('emoji-click', handleEmojiClick);
		return {
			destroy() {
				node.removeEventListener('emoji-click', handleEmojiClick);
			}
		};
	}
</script>

<div
	class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-xl dark:border-gray-700 dark:bg-gray-800"
	role="toolbar"
	aria-label="Reaction picker"
>
	<emoji-picker use:setupEmojiPicker class="light"></emoji-picker>
</div>
