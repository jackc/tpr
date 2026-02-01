<script>
	import { goto } from '$app/navigation';
	import { session } from '$lib/session.js';
	import { api } from '$lib/api.js';

	async function logout(e) {
		e.preventDefault();
		await api.logout();
		session.clear();
		goto('/login');
	}
</script>

<header>
	<h1>The Pithy Reader</h1>
	{#if $session.id}
		<nav>
			<a href="/home">Home</a>
			{' '}
			<a href="/archive">Archive</a>
			{' '}
			<a href="/feeds">Feeds</a>
			{' '}
			<a href="/account">Account</a>
			{' '}
			<a href="/login" on:click={logout}>Logout</a>
		</nav>
	{/if}
</header>
