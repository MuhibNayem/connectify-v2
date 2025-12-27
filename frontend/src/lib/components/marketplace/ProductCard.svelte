<script lang="ts">
	import type { Product } from '$lib/api/marketplace';
	import { Heart, MessageCircle, MapPin } from '@lucide/svelte';
	import { toggleSaveProduct } from '$lib/api/marketplace';
	import { createEventDispatcher } from 'svelte';
	import { fade } from 'svelte/transition';
	import { auth } from '$lib/stores/auth.svelte';

	export let product: Product;

	// Check if this is the user's own product
	$: isOwnProduct = auth.state.user?.id === product.seller?.id;

	let isHovered = false;
	let currentImageIndex = 0;
	let isSaved = product.is_saved || false;

	const dispatch = createEventDispatcher();

	function handleMouseEnter() {
		isHovered = true;
	}

	function handleMouseLeave() {
		isHovered = false;
	}

	async function handleSave(e: MouseEvent) {
		e.stopPropagation();
		isSaved = !isSaved;
		try {
			const res = await toggleSaveProduct(product.id);
			isSaved = res.saved;
		} catch (err) {
			isSaved = !isSaved; // Revert on error
			console.error('Failed to toggle save:', err);
		}
	}

	function handleMessage(e: MouseEvent) {
		e.stopPropagation();
		dispatch('message', { product });
	}

	function handleClick() {
		dispatch('click', { product });
	}
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div
	class="glass-card group relative flex h-full cursor-pointer flex-col overflow-hidden"
	on:mouseenter={handleMouseEnter}
	on:mouseleave={handleMouseLeave}
	on:click={handleClick}
>
	<!-- Image Container -->
	<div class="relative aspect-square w-full overflow-hidden bg-white/5">
		{#if product.images && product.images.length > 0}
			<img
				src={product.images[0]}
				alt={product.title}
				class="h-full w-full object-cover transition-transform duration-500 group-hover:scale-110"
			/>
		{:else}
			<div class="flex h-full w-full items-center justify-center text-gray-400">No Image</div>
		{/if}

		<!-- Overlay Gradient -->
		<div
			class="absolute inset-0 bg-gradient-to-t from-black/40 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100"
		></div>

		<!-- Quick Actions (Visible on Hover) -->
		<div
			class="absolute bottom-3 right-3 flex translate-y-2 transform gap-2 opacity-0 transition-all duration-300 group-hover:translate-y-0 group-hover:opacity-100"
		>
			{#if !isOwnProduct}
				<button
					class="rounded-full bg-white/20 p-2 text-white backdrop-blur-md transition-colors hover:bg-white/40"
					on:click={handleMessage}
					title="Message Seller"
				>
					<MessageCircle size={18} />
				</button>
			{/if}
			<button
				class="rounded-full bg-white/20 p-2 backdrop-blur-md transition-colors hover:bg-white/40 {isSaved
					? 'text-red-500'
					: 'text-white'}"
				on:click={handleSave}
				title="Save"
			>
				<Heart size={18} fill={isSaved ? 'currentColor' : 'none'} />
			</button>
		</div>
	</div>

	<!-- Content -->
	<div class="flex flex-col gap-1 p-3">
		<h3 class="truncate font-medium text-gray-900 transition-colors group-hover:text-blue-600">
			{product.title}
		</h3>

		<div class="flex items-baseline gap-1">
			<span class="text-lg font-bold text-gray-900">
				{product.currency}
				{product.price.toLocaleString()}
			</span>
		</div>

		<div class="mt-1 flex items-center gap-1 text-xs text-gray-500">
			<MapPin size={12} />
			<span class="truncate">{product.location}</span>
		</div>
	</div>
</div>
