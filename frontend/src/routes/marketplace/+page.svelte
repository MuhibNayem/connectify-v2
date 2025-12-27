<script lang="ts">
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import * as Icons from '@lucide/svelte';
	import ProductCard from '$lib/components/marketplace/ProductCard.svelte';
	import FilterSidebar from '$lib/components/marketplace/FilterSidebar.svelte';
	import CreateListingModal from '$lib/components/marketplace/CreateListingModal.svelte';
	import ProductDetailsModal from '$lib/components/marketplace/ProductDetailsModal.svelte';
	import { getProducts, getProduct } from '$lib/api/marketplace';
	import type { Product, ProductFilter } from '$lib/api/marketplace';
	import { auth } from '$lib/stores/auth.svelte';
	import { goto, replaceState, afterNavigate } from '$app/navigation';
	import { page } from '$app/stores';
	import { sendMessage } from '$lib/api';

	// State using Svelte 5 runes
	let products = $state<Product[]>([]);
	let loading = $state(true);
	let searchQuery = $state('');
	let activeCategoryId = $state<string | undefined>(undefined);

	// Modal State
	let showCreateModal = $state(false);
	let selectedProduct = $state<Product | null>(null);

	// This page is the browse tab
	const activeTab = 'browse';

	async function loadProducts(filter: ProductFilter = {}) {
		loading = true;
		try {
			const res = await getProducts(filter);
			products = res.products;
		} catch (err) {
			console.error('Failed to load products:', err);
		} finally {
			loading = false;
		}
	}

	// Track last product ID to avoid re-fetching
	let lastProductId = $state<string | null>(null);

	// Function to open product modal from URL
	async function openProductFromUrl(productId: string) {
		if (productId && productId !== lastProductId) {
			lastProductId = productId;
			try {
				const product = await getProduct(productId);
				selectedProduct = product;
			} catch (err) {
				console.error('Failed to load product from URL:', err);
			}
		}
	}

	// Watch URL for product parameter using $effect (Svelte 5 runes)
	$effect(() => {
		const productId = $page.url.searchParams.get('product');
		if (productId) {
			openProductFromUrl(productId);
		} else if (!productId && lastProductId) {
			lastProductId = null;
		}
	});

	onMount(() => {
		loadProducts();

		// Check URL on mount for product param (handles direct navigation/refresh)
		const productId = $page.url.searchParams.get('product');
		if (productId) {
			openProductFromUrl(productId);
		}
	});

	// Handle navigation events (like clicking product link from chat)
	afterNavigate((navigation) => {
		const productId = navigation.to?.url.searchParams.get('product');
		if (productId) {
			openProductFromUrl(productId);
		}
	});

	function handleSearch() {
		loadProducts({ q: searchQuery, category_id: activeCategoryId });
	}

	function handleFilter(event: CustomEvent) {
		const filter = event.detail;
		activeCategoryId = filter.category_id;
		loadProducts({ ...filter, q: searchQuery });
	}

	async function handleMessageSeller(event: CustomEvent) {
		const { product } = event.detail;
		if (!auth.state.user) {
			alert('Please log in to message seller');
			return;
		}
		if (product.seller && product.seller.id === auth.state.user.id) {
			alert('You cannot message yourself');
			return;
		}

		// Close modal
		selectedProduct = null;

		// Navigate to inbox with product info - let user send manually
		// The inbox page will pre-fill the message input
		const params = new URLSearchParams();
		params.set('seller', product.seller.id);
		params.set('product_id', product.id);
		params.set('product_title', product.title);
		goto(`/marketplace/inbox?${params.toString()}`);
	}

	function handleProductClick(event: CustomEvent) {
		const product = event.detail.product;
		selectedProduct = product;
		// Update URL to reflect selected product
		const url = new URL($page.url);
		url.searchParams.set('product', product.id);
		replaceState(url, {});
	}

	function handleCloseProductModal() {
		selectedProduct = null;
		lastProductId = null;
		// Clear product param from URL
		const url = new URL($page.url);
		url.searchParams.delete('product');
		replaceState(url, {});
	}

	function handleListingCreated() {
		loadProducts(); // Refresh list
	}
</script>

<div class="flex h-screen overflow-hidden bg-[#f0f2f5]">
	<!-- Modals -->
	{#if showCreateModal}
		<CreateListingModal
			on:close={() => (showCreateModal = false)}
			on:success={handleListingCreated}
		/>
	{/if}

	{#if selectedProduct}
		<ProductDetailsModal
			product={selectedProduct}
			on:close={handleCloseProductModal}
			on:message={handleMessageSeller}
		/>
	{/if}

	<!-- Sidebar -->
	<div
		class="hidden h-full w-80 flex-shrink-0 overflow-y-auto border-r border-gray-200 bg-white md:block"
	>
		<div class="sticky top-0 z-10 bg-white p-4">
			<h1 class="mb-4 text-2xl font-bold text-gray-900">Marketplace</h1>

			<!-- Search -->
			<div class="relative mb-4">
				<div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
					<Icons.Search class="text-gray-400" size={20} />
				</div>
				<input
					type="text"
					placeholder="Search Marketplace"
					bind:value={searchQuery}
					onkeydown={(e) => e.key === 'Enter' && handleSearch()}
					class="w-full rounded-full border-none bg-gray-100 py-2 pl-10 pr-4 text-gray-900 transition-all placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
				/>
			</div>

			<!-- Navigation Tabs -->
			<div class="mb-2 flex gap-2 overflow-x-auto pb-2">
				<a
					href="/marketplace"
					class="whitespace-nowrap rounded-full bg-blue-100 px-4 py-2 text-sm font-medium text-blue-600 transition-colors"
				>
					Browse
				</a>
				<a
					href="/marketplace/selling"
					class="whitespace-nowrap rounded-full bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200"
				>
					Selling
				</a>
				<a
					href="/marketplace/inbox"
					class="whitespace-nowrap rounded-full bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200"
				>
					Inbox
				</a>
			</div>

			<div class="my-4 border-t border-gray-200"></div>

			<button
				class="mb-6 flex w-full items-center justify-center gap-2 rounded-lg bg-blue-100 py-2.5 font-semibold text-blue-700 transition-colors hover:bg-blue-200"
				onclick={() => (showCreateModal = true)}
			>
				<Icons.Plus size={20} />
				Create New Listing
			</button>

			<!-- Filter Component -->
			<FilterSidebar on:filter={handleFilter} {activeCategoryId} />
		</div>
	</div>

	<!-- Main Content Area -->
	<div class="relative flex h-full flex-1 flex-col overflow-hidden">
		<div class="h-full overflow-y-auto p-4 md:p-8">
			<div class="mx-auto max-w-7xl">
				<div class="mb-6 flex items-center justify-between">
					<h2 class="text-xl font-bold text-gray-900">Today's Picks</h2>
				</div>

				{#if loading}
					<div class="grid grid-cols-1 gap-4 p-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
						{#each Array(8) as _}
							<div class="h-80 animate-pulse rounded-xl bg-white shadow-sm"></div>
						{/each}
					</div>
				{:else if products.length === 0}
					<div class="flex flex-col items-center justify-center py-20 text-gray-500">
						<Icons.ShoppingBag size={48} class="mb-4 opacity-50" />
						<p class="text-lg font-medium">No items found</p>
						<p class="text-sm">Try adjusting your filters or search query</p>
					</div>
				{:else}
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
						{#each products as product (product.id)}
							<div in:fade={{ duration: 200 }}>
								<ProductCard
									{product}
									on:message={handleMessageSeller}
									on:click={handleProductClick}
								/>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	</div>
</div>
