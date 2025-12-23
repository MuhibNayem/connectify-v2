<script lang="ts">
    export let message: {
        id: number;
        sender: string;
        content: string;
        time: string;
        isSelf: boolean;
    };
</script>

<div class="flex {message.isSelf ? 'justify-end' : 'justify-start'}">
    <div class="flex items-end {message.isSelf ? 'flex-row-reverse space-x-reverse' : 'flex-row space-x-2'}">
        <!-- Avatar (only for others' messages) -->
        {#if !message.isSelf}
            <div class="w-8 h-8 rounded-full bg-gray-300 flex items-center justify-center text-gray-600 text-sm font-bold">
                {message.sender.substring(0, 2)}
            </div>
        {/if}

        <!-- Message Content Bubble -->
        <div class="max-w-xs lg:max-w-md px-4 py-2 rounded-xl {message.isSelf ? 'bg-indigo-500 text-white rounded-br-none' : 'bg-gray-200 text-gray-800 rounded-bl-none'} shadow-sm">
            {#if !message.isSelf}
                <p class="font-semibold text-sm mb-1">{message.sender}</p>
            {/if}
            <p class="text-sm">{message.content}</p>
            <span class="text-xs {message.isSelf ? 'text-indigo-200' : 'text-gray-500'} block text-right mt-1">{message.time}</span>
        </div>
    </div>
</div>
