<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { fade, scale } from 'svelte/transition';
	import { X, MessageCircle, Heart, Share2, MapPin, Clock, Tag } from '@lucide/svelte';
	import type { Product } from '$lib/api/marketplace';
	import { toggleSaveProduct } from '$lib/api/marketplace';
	import { auth } from '$lib/stores/auth.svelte';

	export let product: Product;

	const dispatch = createEventDispatcher();

	// Check if current user is the seller
	$: isOwnProduct = auth.state.user?.id === product.seller?.id;
	let currentImageIndex = 0;
	let isSaved = product.is_saved || false;

	function nextImage() {
		if (product.images.length > 1) {
			currentImageIndex = (currentImageIndex + 1) % product.images.length;
		}
	}

	function prevImage() {
		if (product.images.length > 1) {
			currentImageIndex = (currentImageIndex - 1 + product.images.length) % product.images.length;
		}
	}

	async function handleSave() {
		isSaved = !isSaved;
		try {
			await toggleSaveProduct(product.id);
		} catch (err) {
			isSaved = !isSaved;
			console.error('Failed to toggle save:', err);
		}
	}

	function handleMessage() {
		dispatch('message', { product });
	}

	function formatDate(dateString: string) {
		return new Date(dateString).toLocaleDateString(undefined, {
			year: 'numeric',
			month: 'long',
			day: 'numeric'
		});
	}
</script>

<div
	class="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto p-0 sm:p-4"
	transition:fade={{ duration: 200 }}
>
	<div
		class="absolute inset-0 bg-black/80 backdrop-blur-sm"
		on:click={() => dispatch('close')}
	></div>

	<div
		class="relative flex h-full w-full max-w-5xl flex-col overflow-hidden bg-white shadow-2xl sm:rounded-2xl md:h-[85vh] md:flex-row"
		transition:scale={{ duration: 200, start: 0.95 }}
	>
		<button
			class="absolute left-4 top-4 z-20 rounded-full bg-black/50 p-2 text-white hover:bg-black/70 md:hidden"
			on:click={() => dispatch('close')}
		>
			<X size={20} />
		</button>

		<!-- Left: Image Gallery -->
		<div
			class="relative flex min-h-[40vh] w-full items-center justify-center bg-black md:min-h-full md:w-3/5"
		>
			{#if product.images && product.images.length > 0}
				<img
					src={product.images[currentImageIndex]}
					alt={product.title}
					class="max-h-full max-w-full object-contain"
				/>

				{#if product.images.length > 1}
					<button
						class="absolute left-4 top-1/2 -translate-y-1/2 rounded-full bg-white/10 p-2 text-white backdrop-blur-md transition-all hover:bg-white/20"
						on:click={prevImage}
					>
						←
					</button>
					<button
						class="absolute right-4 top-1/2 -translate-y-1/2 rounded-full bg-white/10 p-2 text-white backdrop-blur-md transition-all hover:bg-white/20"
						on:click={nextImage}
					>
						→
					</button>

					<div class="absolute bottom-4 left-1/2 flex -translate-x-1/2 gap-2">
						{#each product.images as _, i}
							<div
								class="h-2 w-2 rounded-full transition-all {i === currentImageIndex
									? 'scale-110 bg-white'
									: 'bg-white/40'}"
							></div>
						{/each}
					</div>
				{/if}
			{:else}
				<div class="text-gray-500">No Images</div>
			{/if}
		</div>

		<!-- Right: Details -->
		<div class="flex h-full w-full flex-col bg-white md:w-2/5">
			<!-- Header (Desktop Close) -->
			<div class="hidden justify-end border-b border-gray-100 p-4 md:flex">
				<div class="flex gap-2">
					<button class="rounded-full p-2 text-gray-500 transition-colors hover:bg-gray-100">
						<Share2 size={20} />
					</button>
					<button
						class="rounded-full p-2 text-gray-500 transition-colors hover:bg-gray-100"
						on:click={() => dispatch('close')}
					>
						<X size={20} />
					</button>
				</div>
			</div>

			<!-- Scrollable Content -->
			<div class="flex-1 space-y-6 overflow-y-auto p-6 md:p-8">
				<div>
					<h1 class="mb-2 text-2xl font-bold text-gray-900">{product.title}</h1>
					<p class="text-3xl font-bold text-gray-900">
						{product.currency}
						{product.price.toLocaleString()}
					</p>
					<div class="mt-2 flex items-center gap-2 text-sm text-gray-500">
						<Clock size={14} />
						<span>Listed on {formatDate(product.created_at)}</span>
					</div>
				</div>

				<!-- Actions -->
				<div class="grid {isOwnProduct ? 'grid-cols-1' : 'grid-cols-2'} gap-3">
					{#if !isOwnProduct}
						<button
							class="flex flex-1 items-center justify-center gap-2 rounded-lg bg-blue-600 py-3 font-bold text-white shadow-lg shadow-blue-200 transition-colors hover:bg-blue-700"
							on:click={handleMessage}
						>
							<MessageCircle size={20} />
							Message Seller
						</button>
					{/if}
					<button
						class="flex flex-1 items-center justify-center gap-2 rounded-lg bg-gray-100 py-3 font-bold text-gray-900 transition-colors hover:bg-gray-200"
						on:click={handleSave}
					>
						<Heart
							size={20}
							fill={isSaved ? 'currentColor' : 'none'}
							class={isSaved ? 'text-red-500' : ''}
						/>
						{isSaved ? 'Saved' : 'Save'}
					</button>
				</div>

				<!-- Description -->
				<div>
					<h3 class="mb-2 font-bold text-gray-900">Details</h3>
					<div class="space-y-3 text-sm text-gray-600">
						<div class="flex items-center gap-3">
							<div class="flex w-8 justify-center"><Tag size={18} /></div>
							<span class="font-medium text-gray-900">Condition: Used - Good</span>
						</div>
						<div class="flex items-center gap-3">
							<div class="flex w-8 justify-center"><MapPin size={18} /></div>
							<span class="font-medium text-gray-900">{product.location}</span>
						</div>
						<div class="flex items-center gap-3">
							<div class="flex w-8 justify-center"><Tag size={20} /></div>
							<span class="font-medium text-gray-900">{product.category?.name || 'Category'}</span>
						</div>
					</div>
				</div>

				<div>
					<h3 class="mb-2 font-bold text-gray-900">Description</h3>
					<p class="whitespace-pre-wrap leading-relaxed text-gray-700">{product.description}</p>
				</div>

				<!-- Seller Info -->
				<div class="mt-6 border-t border-gray-100 pt-6">
					<h3 class="mb-4 font-bold text-gray-900">Seller Information</h3>
					<div class="flex items-center gap-4">
						<div class="h-12 w-12 overflow-hidden rounded-full bg-gray-200">
							{#if product.seller?.avatar}
								<img
									src={product.seller.avatar}
									alt={product.seller.username}
									class="h-full w-full object-cover"
								/>
							{:else}
								<div
									class="flex h-full w-full items-center justify-center bg-blue-100 text-lg font-bold text-blue-600"
								>
									{product.seller?.username?.[0]?.toUpperCase() || 'S'}
								</div>
							{/if}
						</div>
						<div>
							<h4 class="font-bold text-gray-900">
								{product.seller?.full_name || product.seller?.username || 'Unknown Seller'}
							</h4>
							<p class="text-xs text-gray-500">Member since 2024</p>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>
