<script lang="ts">
	import { getContext } from 'svelte';

	let { href = undefined, children, onclick, ...restProps } = $props();
	const menu = getContext<any>('dropdown-menu');

	function handleClick(e: MouseEvent) {
		if (onclick) onclick(e);
		menu.close();
	}
</script>

{#if href}
	<a
		{href}
		class="hover:bg-accent hover:text-accent-foreground relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors"
		onclick={handleClick}
		{...restProps}
	>
		{@render children?.()}
	</a>
{:else}
	<div
		class="hover:bg-accent hover:text-accent-foreground relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors"
		role="menuitem"
		tabindex="0"
		onclick={handleClick}
		onkeydown={(e) => e.key === 'Enter' && handleClick(e as any)}
		{...restProps}
	>
		{@render children?.()}
	</div>
{/if}
