<script lang="ts">
	let { children } = $props();
</script>

<p class="text-muted-foreground text-sm">
	{@render children?.()}
</p>
