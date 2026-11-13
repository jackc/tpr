<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';

	onMount(() => {
		// Only redirect if we're actually on the root path
		const currentPath = page.url.pathname;
		if (currentPath === '/') {
			goto('/home', { replaceState: true });
		}
	});
</script>
