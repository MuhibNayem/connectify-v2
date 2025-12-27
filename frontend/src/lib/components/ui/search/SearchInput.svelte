<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import { Input } from '$lib/components/ui/input';
    import { Button } from '$lib/components/ui/button';
    import { Search } from '@lucide/svelte'; 

    const dispatch = createEventDispatcher();
    let searchQuery: string = '';

    function handleSearch() {
        if (searchQuery.trim()) {
            dispatch('search', searchQuery.trim());
        }
    }

    function handleKeyPress(event: KeyboardEvent) {
        if (event.key === 'Enter') {
            handleSearch();
        }
    }
</script>

<div class="flex w-full max-w-sm items-center space-x-2">
    <Input
        type="text"
        placeholder="Search users or posts..."
        bind:value={searchQuery}
        onkeypress={handleKeyPress}
        class="flex-grow"
    />
    <Button onclick={handleSearch} disabled={!searchQuery.trim()}>
        <Search class="h-4 w-4" />
        <span class="sr-only">Search</span>
    </Button>
</div>
