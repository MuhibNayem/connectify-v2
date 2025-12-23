<script lang="ts">
	let { children } = $props();
</script>

<h3 class="text-lg font-semibold leading-none tracking-tight">
	{@render children?.()}
</h3>
