<script lang="ts">
	import { setContext } from 'svelte';

	let { open = $bindable(false), children } = $props();

	function toggle() {
		open = !open;
	}

	function close() {
		open = false;
	}

	function openMenu() {
		open = true;
	}

	setContext('dropdown-menu', {
		get isOpen() {
			return open;
		},
		toggle,
		close,
		openMenu
	});
</script>

<div class="relative inline-block text-left">
	{@render children?.()}
</div>
