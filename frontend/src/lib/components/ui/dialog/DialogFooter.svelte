<script lang="ts">
	let { children } = $props();
</script>

<div class="flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2">
	{@render children?.()}
</div>
