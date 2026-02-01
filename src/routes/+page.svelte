<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { get } from 'svelte/store';

	onMount(() => {
		// Only redirect if we're actually on the root path
		const currentPath = get(page).url.pathname;
		if (currentPath === '/') {
			goto('/home', { replaceState: true });
		}
	});
</script>
