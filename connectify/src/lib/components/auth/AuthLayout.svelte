<script lang="ts">
    // Removed Card imports as it's no longer used directly in this layout
    // import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '$lib/components/ui/card';

    export let title: string;
    export let description: string | undefined = undefined;
</script>

<div class="flex min-h-screen w-full flex-col md:flex-row font-sans">
    <!-- Left Side: Visuals and Taglines -->
    <div class="relative flex w-full items-center justify-center p-8 md:w-1/2 lg:w-2/5 bg-gradient-to-br from-blue-600 to-indigo-800 text-white shadow-2xl overflow-hidden">
        <!-- Abstract background shapes/pattern for modern feel -->
        <div class="absolute inset-0 opacity-15">
            <svg class="w-full h-full" viewBox="0 0 100 100" preserveAspectRatio="none">
                <defs>
                    <radialGradient id="gradient1" cx="50%" cy="50%" r="50%" fx="50%" fy="50%">
                        <stop offset="0%" stop-color="rgba(255,255,255,0.15)"/>
                        <stop offset="100%" stop-color="rgba(255,255,255,0)"/>
                    </radialGradient>
                    <radialGradient id="gradient2" cx="50%" cy="50%" r="50%" fx="50%" fy="50%">
                        <stop offset="0%" stop-color="rgba(255,255,255,0.1)"/>
                        <stop offset="100%" stop-color="rgba(255,255,255,0)"/>
                    </radialGradient>
                </defs>
                <circle cx="20" cy="20" r="15" fill="url(#gradient1)"/>
                <circle cx="80" cy="80" r="25" fill="url(#gradient2)"/>
                <rect x="10" y="60" width="30" height="10" rx="5" ry="5" fill="rgba(255,255,255,0.08)"/>
                <polygon points="50,10 60,30 40,30" fill="rgba(255,255,255,0.1)"/>
                <circle cx="50" cy="50" r="10" fill="url(#gradient1)"/>
            </svg>
        </div>

        <div class="relative z-10 text-center space-y-6 p-4">
            <h1 class="text-6xl font-extrabold tracking-tight leading-tight drop-shadow-lg">Connectify</h1>
            <p class="text-2xl font-light opacity-90 drop-shadow-md">Connect. Share. Belong.</p>
            <p class="text-lg opacity-80 max-w-sm mx-auto leading-relaxed">Your seamless social experience starts here. Join the conversation and build meaningful connections.</p>
        </div>
    </div>

    <!-- Right Side: Authentication Form Container -->
    <div class="flex w-full items-center justify-center p-8 md:w-1/2 lg:w-3/5 bg-white">
        <div class="w-full max-w-md p-8 bg-white rounded-xl shadow-2xl border border-gray-100 space-y-8">
            <div class="text-center">
                <h2 class="text-4xl font-bold text-gray-900">{title}</h2>
                {#if description}
                    <p class="text-gray-600 mt-2">{description}</p>
                {/if}
            </div>
            
            <!-- Slot for the form content -->
            <slot />

            <!-- Slot for the footer content (e.g., register/login link) -->
            <div class="text-center pt-4 border-t border-gray-100 mt-8">
                <slot name="footer" />
            </div>
        </div>
    </div>
</div>

<style lang="postcss">
    /* Ensure font-sans is applied globally or via tailwind.config.js */
    /* @import 'tailwindcss/base';
    @import 'tailwindcss/components';
    @import 'tailwindcss/utilities'; */
</style>
