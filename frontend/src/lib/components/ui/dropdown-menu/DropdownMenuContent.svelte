<script lang="ts">
	import { getContext } from 'svelte';
	import { fade } from 'svelte/transition';

	let { align = 'end', children } = $props();
	const menu = getContext<any>('dropdown-menu');
</script>

{#if menu.isOpen}
	<!-- Backdrop to close on click outside -->
	<div class="fixed inset-0 z-30" onclick={menu.close} role="presentation"></div>

	<div
		class={`bg-popover text-popover-foreground absolute z-50 mt-2 min-w-[8rem] overflow-hidden rounded-md border p-1 shadow-md
            ${align === 'end' ? 'right-0' : align === 'center' ? 'left-1/2 -translate-x-1/2' : 'left-0'}
        `}
		transition:fade={{ duration: 100 }}
		onclick={(e) => e.stopPropagation()}
	>
		{@render children?.()}
	</div>
{/if}
