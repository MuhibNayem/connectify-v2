<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { fade, scale } from 'svelte/transition';
	import { X, Upload, DollarSign, MapPin, Tag } from '@lucide/svelte';
	import { createProduct } from '$lib/api/marketplace';
	import type { CreateProductRequest } from '$lib/api/marketplace';
	import { uploadFiles } from '$lib/api'; // Assuming generic file upload exists
	import { getCategories } from '$lib/api/marketplace';
	import type { Category } from '$lib/api/marketplace';

	const dispatch = createEventDispatcher();

	let title = '';
	let price = '';
	let description = '';
	let location = '';
	let categoryId = '';
	let selectedImages: File[] = [];
	let imagePreviews: string[] = [];
	let isSubmitting = false;
	let categories: Category[] = [];

	// Load categories on mount
	import { onMount } from 'svelte';
	onMount(async () => {
		try {
			categories = await getCategories();
		} catch (err) {
			console.error('Failed to load categories', err);
		}
	});

	function handleFiles(event: Event) {
		const target = event.target as HTMLInputElement;
		if (target.files) {
			const files = Array.from(target.files);

			if (selectedImages.length + files.length > 5) {
				alert('You can only upload a maximum of 5 images.');
				return;
			}

			selectedImages = [...selectedImages, ...files];

			// Generate previews
			files.forEach((file) => {
				const reader = new FileReader();
				reader.onload = (e) => {
					imagePreviews = [...imagePreviews, e.target?.result as string];
				};
				reader.readAsDataURL(file);
			});
		}
	}

	function removeImage(index: number) {
		selectedImages = selectedImages.filter((_, i) => i !== index);
		imagePreviews = imagePreviews.filter((_, i) => i !== index);
	}

	async function handleSubmit() {
		if (!title || !price || !categoryId || selectedImages.length === 0) {
			alert('Please fill in all required fields and add at least one photo.');
			return;
		}

		isSubmitting = true;
		try {
			// 1. Upload Images
			const uploadRes = await uploadFiles(selectedImages);
			const imageUrls = uploadRes.map((f) => f.url);

			// 2. Create Product
			const productData: CreateProductRequest = {
				title,
				price: parseFloat(price),
				currency: 'USD', // Default currency
				description,
				location,
				category_id: categoryId,
				images: imageUrls,
				tags: [] // Parse tags if needed
			};

			await createProduct(productData);
			dispatch('success');
			dispatch('close');
		} catch (err) {
			console.error('Failed to create listing:', err);
			alert('Failed to create listing');
		} finally {
			isSubmitting = false;
		}
	}
</script>

<div
	class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6"
	transition:fade={{ duration: 200 }}
>
	<div
		class="absolute inset-0 bg-black/50 backdrop-blur-sm"
		on:click={() => dispatch('close')}
	></div>

	<div
		class="relative flex max-h-[90vh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl"
		transition:scale={{ duration: 200, start: 0.95 }}
	>
		<!-- Header -->
		<div class="z-10 flex items-center justify-between border-b border-gray-100 bg-white px-6 py-4">
			<h2 class="text-xl font-bold text-gray-900">Create New Listing</h2>
			<button
				class="rounded-full p-2 text-gray-500 transition-colors hover:bg-gray-100"
				on:click={() => dispatch('close')}
			>
				<X size={20} />
			</button>
		</div>

		<!-- Body (Scrollable) -->
		<div class="flex-1 space-y-6 overflow-y-auto p-6">
			<!-- Image Upload -->
			<div>
				<label class="mb-2 block text-sm font-medium text-gray-700">Photos (Required)</label>
				<div class="grid grid-cols-3 gap-4 sm:grid-cols-4">
					{#each imagePreviews as preview, i}
						<div
							class="group relative aspect-square overflow-hidden rounded-lg border border-gray-200"
						>
							<img src={preview} alt="Preview" class="h-full w-full object-cover" />
							<button
								class="absolute right-1 top-1 rounded-full bg-black/50 p-1 text-white opacity-0 transition-opacity hover:bg-black/70 group-hover:opacity-100"
								on:click={() => removeImage(i)}
							>
								<X size={12} />
							</button>
						</div>
					{/each}

					<label
						class="flex aspect-square cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 transition-colors hover:border-blue-500 hover:bg-blue-50"
					>
						<Upload class="mb-1 text-gray-400" size={24} />
						<span class="text-xs text-gray-500">Add Photo</span>
						<input type="file" accept="image/*" multiple class="hidden" on:change={handleFiles} />
					</label>
				</div>
			</div>

			<!-- Title & Price -->
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label class="mb-1 block text-sm font-medium text-gray-700">Title</label>
					<input
						type="text"
						bind:value={title}
						placeholder="What are you selling?"
						class="w-full rounded-lg border border-gray-200 bg-gray-50 px-4 py-2 transition-all focus:outline-none focus:ring-2 focus:ring-blue-500"
					/>
				</div>
				<div>
					<label class="mb-1 block text-sm font-medium text-gray-700">Price</label>
					<div class="relative">
						<DollarSign class="absolute left-3 top-2.5 text-gray-400" size={16} />
						<input
							type="number"
							bind:value={price}
							placeholder="0.00"
							class="w-full rounded-lg border border-gray-200 bg-gray-50 py-2 pl-9 pr-4 transition-all focus:outline-none focus:ring-2 focus:ring-blue-500"
						/>
					</div>
				</div>
			</div>

			<!-- Category & Location -->
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label class="mb-1 block text-sm font-medium text-gray-700">Category</label>
					<select
						bind:value={categoryId}
						class="w-full rounded-lg border border-gray-200 bg-gray-50 px-4 py-2 transition-all focus:outline-none focus:ring-2 focus:ring-blue-500"
					>
						<option value="" disabled selected>Select a category</option>
						{#each categories as category}
							<option value={category.id}>{category.name}</option>
						{/each}
					</select>
				</div>
				<div>
					<label class="mb-1 block text-sm font-medium text-gray-700">Location</label>
					<div class="relative">
						<MapPin class="absolute left-3 top-2.5 text-gray-400" size={16} />
						<input
							type="text"
							bind:value={location}
							placeholder="City, Neighborhood"
							class="w-full rounded-lg border border-gray-200 bg-gray-50 py-2 pl-9 pr-4 transition-all focus:outline-none focus:ring-2 focus:ring-blue-500"
						/>
					</div>
				</div>
			</div>

			<!-- Description -->
			<div>
				<label class="mb-1 block text-sm font-medium text-gray-700">Description</label>
				<textarea
					bind:value={description}
					rows="4"
					placeholder="Describe your item (condition, reason for selling, etc.)"
					class="w-full resize-none rounded-lg border border-gray-200 bg-gray-50 px-4 py-2 transition-all focus:outline-none focus:ring-2 focus:ring-blue-500"
				></textarea>
			</div>
		</div>

		<!-- Footer -->
		<div class="flex justify-end gap-3 rounded-b-2xl border-t border-gray-100 bg-gray-50 px-6 py-4">
			<button
				class="rounded-lg px-5 py-2 font-medium text-gray-700 transition-colors hover:bg-gray-200"
				on:click={() => dispatch('close')}
				disabled={isSubmitting}
			>
				Cancel
			</button>
			<button
				class="flex items-center gap-2 rounded-lg bg-blue-600 px-5 py-2 font-medium text-white shadow-sm transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
				on:click={handleSubmit}
				disabled={isSubmitting}
			>
				{#if isSubmitting}
					<div
						class="h-4 w-4 animate-spin rounded-full border-2 border-white/30 border-t-white"
					></div>
					Publishing...
				{:else}
					Publish Listing
				{/if}
			</button>
		</div>
	</div>
</div>
