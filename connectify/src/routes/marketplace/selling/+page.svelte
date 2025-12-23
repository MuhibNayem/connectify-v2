<script lang="ts">
	import * as Icons from '@lucide/svelte';
	import { goto } from '$app/navigation';
</script>

<div class="flex h-screen overflow-hidden bg-[#f0f2f5]">
	<!-- Sidebar -->
	<div
		class="hidden h-full w-80 flex-shrink-0 overflow-y-auto border-r border-gray-200 bg-white md:block"
	>
		<div class="sticky top-0 z-10 bg-white p-4">
			<h1 class="mb-4 text-2xl font-bold text-gray-900">Marketplace</h1>

			<!-- Navigation Tabs -->
			<div class="mb-2 flex gap-2 overflow-x-auto pb-2">
				<a
					href="/marketplace"
					class="whitespace-nowrap rounded-full bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200"
				>
					Browse
				</a>
				<a
					href="/marketplace/selling"
					class="whitespace-nowrap rounded-full bg-blue-100 px-4 py-2 text-sm font-medium text-blue-600 transition-colors"
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
		</div>
	</div>

	<!-- Main Content Area -->
	<div class="relative flex h-full flex-1 flex-col overflow-hidden">
		<div class="flex h-full flex-col items-center justify-center text-gray-500">
			<Icons.Tag size={48} class="mb-4" />
			<p class="text-lg font-medium">Your Listings</p>
			<p class="text-sm text-gray-400">Coming Soon</p>
			<a
				href="/marketplace"
				class="mt-4 rounded-full bg-blue-600 px-6 py-2 font-medium text-white hover:bg-blue-700"
			>
				Create First Listing
			</a>
		</div>
	</div>
</div>
