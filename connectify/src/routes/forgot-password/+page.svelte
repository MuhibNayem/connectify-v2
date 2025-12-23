<script lang="ts">
    import AuthLayout from '$lib/components/auth/AuthLayout.svelte';
    import { Button } from '$lib/components/ui/button';
    import { Input } from '$lib/components/ui/input';
    import { Label } from '$lib/components/ui/label';
    // Removed 'goto' as we'll use href for navigation links
    // import { goto } from '$app/navigation';

    let email = '';
    let errorMessage: string | null = null; // For displaying API errors

    async function handleForgotPassword() {
        errorMessage = null; // Clear previous errors
        console.log('Sending reset link to:', { email });
        // In a real app, you'd call your backend API here, e.g.:
        // try {
        //     await apiRequest('POST', '/auth/forgot-password', { email }, false);
        //     alert('Password reset link sent to your email!');
        //     goto('/'); // Redirect to login
        // } catch (error: any) {
        //     console.error('Forgot password failed:', error.message);
        //     errorMessage = error.message || 'Failed to send reset link. Please check your email.';
        // }
        alert('Password reset logic would go here!');
        // goto('/'); // Simulate successful request
    }
</script>

<AuthLayout title="Forgot Password?" description="Enter your email to receive a reset link">
    <form on:submit|preventDefault={handleForgotPassword} class="space-y-6">
        {#if errorMessage}
            <div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative" role="alert">
                <span class="block sm:inline">{errorMessage}</span>
            </div>
        {/if}
        <div>
            <Label for="email" class="text-sm font-medium text-gray-700">Email</Label>
            <Input
                id="email"
                type="email"
                placeholder="you@example.com"
                bind:value={email}
                required
                class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm placeholder-gray-400 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
            />
        </div>
        <Button type="submit" class="w-full py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500">
            Send Reset Link
        </Button>
    </form>

    <div slot="footer" class="text-center">
        <Button variant="link" href="/" class="text-sm text-gray-600 hover:text-gray-900 p-0 h-auto">
            Back to Login
        </Button>
    </div>
</AuthLayout>