import { goto as svelteGoto } from '$app/navigation';

/**
 * Navigate using hash routing
 * @param {string} path - The path to navigate to (e.g., '/home', '/login')
 * @param {object} opts - Navigation options
 */
export function goto(path, opts = {}) {
	// Always use hash-based navigation
	const hashPath = path.startsWith('#') ? path : `#${path}`;

	// Update the hash without triggering a full page reload
	if (typeof window !== 'undefined') {
		window.location.hash = hashPath;
	}

	// Also call SvelteKit's goto for internal routing
	return svelteGoto(path, opts);
}
