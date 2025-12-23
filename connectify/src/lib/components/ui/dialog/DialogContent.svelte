<script lang="ts">
	import { getContext } from 'svelte';
	import { fade, scale } from 'svelte/transition';
	import { X } from '@lucide/svelte';

	let { children } = $props();
	const dialog = getContext<any>('dialog');
</script>

{#if dialog.isOpen}
	<div
		class="fixed inset-0 z-50 flex items-start justify-center bg-black/80 backdrop-blur-sm sm:items-center"
		transition:fade={{ duration: 150 }}
		onclick={(e) => {
			if (e.target === e.currentTarget) dialog.close();
		}}
		role="dialog"
		aria-modal="true"
	>
		<div
			class="bg-background fixed left-[50%] top-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4 border p-6 shadow-lg sm:rounded-lg"
			transition:scale={{ duration: 200, start: 0.95 }}
		>
			{@render children?.()}
			<button
				onclick={dialog.close}
				class="ring-offset-background focus:ring-ring absolute right-4 top-4 rounded-sm opacity-70 transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:pointer-events-none"
			>
				<X class="h-4 w-4" />
				<span class="sr-only">Close</span>
			</button>
		</div>
	</div>
{/if}
