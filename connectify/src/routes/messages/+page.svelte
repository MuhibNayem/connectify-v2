<script lang="ts">
    import ConversationList from '$lib/components/messages/ConversationList.svelte';
</script>

<div class="flex h-screen bg-gray-100 font-sans">
    <!-- Left Pane: Conversation List -->
    <aside class="w-full md:w-1/3 lg:w-1/4 h-screen flex flex-col">
        <ConversationList />
    </aside>

    <!-- Right Pane: Empty Chat Window (or instructions) -->
    <main class="flex-1 flex items-center justify-center bg-gray-50 p-4 h-screen">
        <div class="text-center text-gray-500">
            <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
            </svg>
            <h2 class="mt-2 text-2xl font-semibold">Select a chat</h2>
            <p class="mt-1 text-sm">Choose from your existing conversations to start messaging.</p>
        </div>
    </main>
</div>