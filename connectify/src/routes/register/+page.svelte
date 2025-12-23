<script lang="ts">
	import AuthLayout from '$lib/components/auth/AuthLayout.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.svelte'; // Import the auth store

	let username = '';
	let email = '';
	let password = '';
	let confirmPassword = '';
	let errorMessage: string | null = null;
	let isLoading = false;

	async function handleRegister() {
		errorMessage = null;
		isLoading = true;
		if (password !== confirmPassword) {
			errorMessage = 'Passwords do not match!';
			isLoading = false;
			return;
		}
		try {
			await auth.register({ username, email, password });
			goto('/dashboard'); // Redirect to dashboard on success
		} catch (error: any) {
			console.error('Registration failed:', error.message);
			errorMessage = error.message || 'Registration failed. Please try again.';
		} finally {
			isLoading = false;
		}
	}
</script>

<AuthLayout title="Join Connectify" description="Create your new account">
	<form
		onsubmit={(e) => {
			e.preventDefault();
			handleRegister();
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
			<Label for="username" class="text-sm font-medium text-gray-700">Username</Label>
			<Input
				id="username"
				type="text"
				placeholder="yourusername"
				bind:value={username}
				required
				class="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 placeholder-gray-400 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-indigo-500 sm:text-sm"
			/>
		</div>
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
		<div>
			<Label for="confirm-password" class="text-sm font-medium text-gray-700"
				>Confirm Password</Label
			>
			<Input
				id="confirm-password"
				type="password"
				placeholder="••••••••"
				bind:value={confirmPassword}
				required
				class="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 placeholder-gray-400 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-indigo-500 sm:text-sm"
			/>
		</div>
		<Button
			type="submit"
			class="w-full rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
		>
			Register
		</Button>
	</form>

	<div slot="footer" class="text-center">
		<Button variant="link" href="/" class="h-auto p-0 text-sm text-gray-600 hover:text-gray-900">
			Already have an account? Login
		</Button>
	</div>
</AuthLayout>
