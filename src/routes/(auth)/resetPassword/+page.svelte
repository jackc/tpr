<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { api } from '$lib/api.js';
	import { session } from '$lib/session.js';

	let token = '';
	let password = '';

	onMount(() => {
		token = $page.url.searchParams.get('token') || '';
	});

	async function resetPassword() {
		try {
			const data = await api.resetPassword({ token, password });
			alert('Successfully reset password');
			session.set({ id: data.sessionID, name: data.name });
			goto('/home');
		} catch (error) {
			alert('Failure resetting password');
		}
	}
</script>

<div class="lostPassword">
	<header>
		<h1>The Pithy Reader</h1>
	</header>
	<form on:submit|preventDefault={resetPassword}>
		<dl>
			<dt>
				<label for="password">Password</label>
			</dt>
			<dd>
				<input type="password" id="password" autofocus bind:value={password} />
			</dd>
		</dl>

		<input type="submit" value="Reset Password" />
	</form>
</div>
