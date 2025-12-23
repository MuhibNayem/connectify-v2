<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import type { Category } from '$lib/api/marketplace';
	import { getCategories } from '$lib/api/marketplace';
	import * as Icons from '@lucide/svelte';

	export let activeCategoryId: string | undefined = undefined;

	let categories: Category[] = [];
	let minPrice: number | undefined;
	let maxPrice: number | undefined;
	let location: string = '';

	const dispatch = createEventDispatcher();

	onMount(async () => {
		try {
			categories = await getCategories();
		} catch (err) {
			console.error('Failed to load categories:', err);
		}
	});

	function handleCategoryClick(id: string) {
		activeCategoryId = id === activeCategoryId ? undefined : id;
		dispatch('filter', { category_id: activeCategoryId });
	}

	function handleApplyFilters() {
		dispatch('filter', {
			category_id: activeCategoryId,
			min_price: minPrice,
			max_price: maxPrice,
			location
		});
	}

	function getIcon(name: string) {
		// Dynamic icon loading from lucide-svelte
		// @ts-ignore
		return Icons[name] || Icons.Box;
	}
</script>

<div class="hidden w-64 flex-shrink-0 space-y-6 p-4 md:block">
	<!-- Categories -->
	<div>
		<h3 class="mb-3 px-2 text-lg font-bold text-gray-900">Categories</h3>
		<div class="space-y-1">
			<button
				class="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors
				{activeCategoryId === undefined ? 'bg-blue-50 text-blue-600' : 'text-gray-700 hover:bg-gray-100'}"
				on:click={() => handleCategoryClick('')}
			>
				<div class="rounded-full bg-gray-200 p-1.5"><Icons.Store size={18} /></div>
				<span class="font-medium">Browse All</span>
			</button>

			{#each categories as category}
				<button
					class="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors
					{activeCategoryId === category.id ? 'bg-blue-50 text-blue-600' : 'text-gray-700 hover:bg-gray-100'}"
					on:click={() => handleCategoryClick(category.id)}
				>
					<div class="rounded-full bg-gray-200 p-1.5">
						<svelte:component this={getIcon(category.icon)} size={18} />
					</div>
					<span class="font-medium">{category.name}</span>
				</button>
			{/each}
		</div>
	</div>

	<!-- Filters -->
	<div>
		<h3 class="mb-3 px-2 text-lg font-bold text-gray-900">Filters</h3>
		<div class="space-y-4 px-2">
			<!-- Price Range -->
			<div>
				<label class="mb-1 block text-sm font-medium text-gray-700">Price Range</label>
				<div class="flex gap-2">
					<input
						type="number"
						placeholder="Min"
						bind:value={minPrice}
						class="w-full rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
					/>
					<input
						type="number"
						placeholder="Max"
						bind:value={maxPrice}
						class="w-full rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
					/>
				</div>
			</div>

			<!-- Location -->
			<div>
				<label class="mb-1 block text-sm font-medium text-gray-700">Location</label>
				<div class="relative">
					<Icons.MapPin class="absolute left-3 top-2.5 text-gray-400" size={16} />
					<input
						type="text"
						placeholder="Enter location"
						bind:value={location}
						class="w-full rounded-lg border border-gray-200 bg-gray-50 py-2 pl-9 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
					/>
				</div>
			</div>

			<button
				class="w-full rounded-lg bg-blue-600 py-2 font-medium text-white shadow-sm transition-colors hover:bg-blue-700"
				on:click={handleApplyFilters}
			>
				Apply Filters
			</button>
		</div>
	</div>
</div>
