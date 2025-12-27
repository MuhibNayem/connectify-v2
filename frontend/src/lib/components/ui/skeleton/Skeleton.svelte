<script lang="ts">
	import { cn } from '$lib/utils';
	let { class: className, ...rest } = $props();
</script>

<div class={cn('bg-muted/10 animate-pulse rounded-md', className)} {...rest}></div>
