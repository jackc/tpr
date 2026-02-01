import { goto } from '$app/navigation';
import { page } from '$app/stores';

let isInitialized = false;
let pageUnsubscribe;

export function initHashRouting() {
	if (isInitialized) return;
	isInitialized = true;

	// Handle hash changes (user clicks browser back/forward or manually changes hash)
	const handleHashChange = () => {
		const hash = window.location.hash.slice(1) || '/';
		goto(hash, { replaceState: true, noScroll: false });
	};

	window.addEventListener('hashchange', handleHashChange);

	// Initialize from current hash
	const initialHash = window.location.hash.slice(1) || '/';
	if (initialHash !== window.location.pathname) {
		goto(initialHash, { replaceState: true });
	}

	// Update hash when SvelteKit route changes
	pageUnsubscribe = page.subscribe(($page) => {
		const newHash = '#' + $page.url.pathname;
		if (window.location.hash !== newHash) {
			// Update hash without triggering hashchange event
			history.replaceState(history.state, '', newHash);
		}
	});

	// Return cleanup function
	return () => {
		window.removeEventListener('hashchange', handleHashChange);
		if (pageUnsubscribe) {
			pageUnsubscribe();
		}
		isInitialized = false;
	};
}
