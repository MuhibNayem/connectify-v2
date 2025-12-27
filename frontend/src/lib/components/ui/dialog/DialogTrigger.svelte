<script lang="ts">
	import { getContext } from 'svelte';

	let { children } = $props();
	const dialog = getContext<any>('dialog');
</script>

<div
	onclick={dialog.openDialog}
	role="button"
	tabindex="0"
	onkeydown={(e) => e.key === 'Enter' && dialog.openDialog()}
	class="inline-block"
>
	{@render children?.()}
</div>
