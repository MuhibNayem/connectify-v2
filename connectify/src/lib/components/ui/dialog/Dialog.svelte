<script lang="ts">
	import { setContext } from 'svelte';

	let { open = $bindable(false), children } = $props();

	function toggle() {
		open = !open;
	}

	function close() {
		open = false;
	}

	function openDialog() {
		open = true;
	}

	setContext('dialog', {
		get isOpen() {
			return open;
		},
		toggle,
		close,
		openDialog
	});
</script>

{@render children?.()}
