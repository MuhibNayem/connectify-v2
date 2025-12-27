<script lang="ts">
	import AuthLayout from '$lib/components/auth/AuthLayout.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Checkbox } from '$lib/components/ui/checkbox';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.svelte'; // Import the new auth store

	let email = '';
	let password = '';
	let rememberMe = false;
	let errorMessage: string | null = null;

	async function handleLogin() {
		errorMessage = null;
		try {
			await auth.login({ email, password }); // Use the new auth store
			goto('/dashboard'); // Redirect to dashboard on success
		} catch (error: any) {
			console.error('Login failed:', error.message);
			errorMessage = error.message || 'Login failed. Please check your credentials.';
		}
	}
</script>

<AuthLayout title="Welcome Back" description="Sign in to your Connectify account">
	<form
		onsubmit={(e) => {
			e.preventDefault();
			handleLogin();
		}}
		class="space-y-6"
	>
		{#if errorMessage}
			<div
				class="relative rounded border border-red-400 bg-red-100 px-4 py-3 text-red-700"
				role="alert"
			>
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
				class="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 placeholder-gray-400 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-indigo-500 sm:text-sm"
			/>
		</div>
		<div>
			<Label for="password" class="text-sm font-medium text-gray-700">Password</Label>
			<Input
				id="password"
				type="password"
				placeholder="••••••••"
				bind:value={password}
				required
				class="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 placeholder-gray-400 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-indigo-500 sm:text-sm"
			/>
		</div>
		<div class="flex items-center justify-between">
			<div class="flex items-center">
				<Checkbox
					id="remember-me"
					bind:checked={rememberMe}
					class="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
				/>
				<Label for="remember-me" class="ml-2 block text-sm text-gray-900">Remember me</Label>
			</div>
			<Button
				variant="link"
				href="/forgot-password"
				class="h-auto p-0 text-sm text-indigo-600 hover:text-indigo-500"
			>
				Forgot password?
			</Button>
		</div>
		<Button
			type="submit"
			class="w-full rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
		>
			Login
		</Button>
	</form>

	<div slot="footer" class="text-center">
		<Button
			variant="link"
			href="/register"
			class="h-auto p-0 text-sm text-gray-600 hover:text-gray-900"
		>
			Don't have an account? Register
		</Button>
	</div>
</AuthLayout>
