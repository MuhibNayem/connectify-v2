<script lang="ts">
	import { callState } from '$lib/stores/voice-call.svelte';
	import CallInterface from './CallInterface.svelte';
	import IncomingCallModal from './IncomingCallModal.svelte';

	// Reactive status derived from store
	let status = $derived($callState.status);
</script>

{#if status === 'incoming'}
	<IncomingCallModal />
{:else if status === 'calling' || status === 'connected'}
	<CallInterface />
{/if}
