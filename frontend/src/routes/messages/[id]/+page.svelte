<script lang="ts">
	import { page } from '$app/stores';
	import ConversationList from '$lib/components/messages/ConversationList.svelte';
	import ChatWindow from '$lib/components/messages/ChatWindow.svelte';

	$: conversationId = $page.params.id;
</script>

<div class="flex h-screen bg-gray-100 font-sans">
	<!-- Left Pane: Conversation List -->
	<aside class="flex h-screen w-full flex-col md:w-1/3 lg:w-1/4">
		<ConversationList />
	</aside>

	<!-- Right Pane: Chat Window -->
	<main class="flex h-screen flex-1 flex-col">
		<ChatWindow conversationId={conversationId ?? ''} />
	</main>
</div>
