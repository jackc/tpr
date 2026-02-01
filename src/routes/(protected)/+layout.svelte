<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { session } from '$lib/session.js';
	import { get } from 'svelte/store';

	onMount(() => {
		// Check authentication using the current store value
		const currentSession = get(session);
		console.log('Protected layout onMount, session:', currentSession);
		if (!currentSession.id) {
			console.log('No session ID, redirecting to login');
			goto('/login');
		} else {
			console.log('Session ID found:', currentSession.id);
		}
	});
</script>

<slot />
