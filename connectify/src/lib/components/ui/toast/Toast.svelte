<script lang="ts">
  import { toastStore } from '$lib/stores/toast';
  import { fly } from 'svelte/transition';
</script>

<div class="fixed bottom-5 right-5 z-50 flex flex-col space-y-2">
  {#each $toastStore as toast (toast.id)}
    <div
      in:fly={{ y: 20, duration: 300 }}
      out:fly={{ y: 20, duration: 300 }}
      class="max-w-sm rounded-md p-4 text-white shadow-lg"
      class:bg-blue-500={toast.type === 'info'}
      class:bg-green-500={toast.type === 'success'}
      class:bg-red-500={toast.type === 'error'}
    >
      {toast.message}
    </div>
  {/each}
</div>
