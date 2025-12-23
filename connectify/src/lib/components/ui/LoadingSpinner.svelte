<script lang="ts">
    export let size: 'sm' | 'md' | 'lg' = 'md';
    export let color: string = 'text-blue-500';

    let spinnerSize: string;
    switch (size) {
        case 'sm':
            spinnerSize = 'w-5 h-5';
            break;
        case 'lg':
            spinnerSize = 'w-12 h-12';
            break;
        case 'md':
        default:
            spinnerSize = 'w-8 h-8';
            break;
    }
</script>

<div class="flex items-center justify-center">
    <div
        class="inline-block animate-spin rounded-full border-4 border-solid border-current border-r-transparent align-[-0.125em] motion-reduce:animate-[spin_1.5s_linear_infinite]"
        class:{spinnerSize} 
        class:{color}
        role="status">
        <span
            class="!absolute !-m-px !h-px !w-px !overflow-hidden !whitespace-nowrap !border-0 !p-0 ![clip:rect(0,0,0,0)]"
            >Loading...</span
        >
    </div>
</div>
