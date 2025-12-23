<script lang="ts">
	let { children } = $props();
</script>

<div class="flex flex-col space-y-1.5 text-center sm:text-left">
	{@render children?.()}
</div>
