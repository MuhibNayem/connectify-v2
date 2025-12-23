<script lang="ts">
	import { getContext } from 'svelte';

	let { children, ...restProps } = $props();
	const menu = getContext<any>('dropdown-menu');
</script>

<div
	onclick={menu.toggle}
	role="button"
	tabindex="0"
	onkeydown={(e) => e.key === 'Enter' && menu.toggle()}
	class="inline-flex"
	{...restProps}
>
	{@render children?.()}
</div>
