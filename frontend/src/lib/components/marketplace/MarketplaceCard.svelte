<script lang="ts">
	let { item } = $props();
</script>

<div
	class="glass-card group cursor-pointer overflow-hidden rounded-xl transition-transform hover:bg-white/5 active:scale-95"
>
	<!-- Image -->
	<div class="relative aspect-square w-full overflow-hidden bg-gray-800">
		<img
			src={item.image}
			alt={item.title}
			class="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105"
		/>
	</div>

	<!-- Content -->
	<div class="p-3">
		<div class="flex items-baseline space-x-1">
			<span class="text-foreground text-lg font-bold">${item.price.toLocaleString()}</span>
		</div>
		<h3 class="text-foreground line-clamp-1 text-sm font-medium">{item.title}</h3>
		<p class="text-muted-foreground mt-0.5 text-xs">{item.location}</p>
	</div>
</div>
