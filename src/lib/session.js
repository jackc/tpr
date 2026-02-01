import { writable } from 'svelte/store';

function createSession() {
	// Load initial session from localStorage
	let initialSession = { id: null, name: null };
	if (typeof localStorage !== 'undefined') {
		const stored = localStorage.getItem('session');
		if (stored) {
			try {
				initialSession = JSON.parse(stored);
			} catch (e) {
				console.error('Failed to parse session from localStorage', e);
			}
		}
	}

	const { subscribe, set, update } = writable(initialSession);

	return {
		subscribe,
		set: (value) => {
			if (typeof localStorage !== 'undefined') {
				localStorage.setItem('session', JSON.stringify(value));
			}
			set(value);
		},
		update: (fn) => {
			update((current) => {
				const updated = fn(current);
				if (typeof localStorage !== 'undefined') {
					localStorage.setItem('session', JSON.stringify(updated));
				}
				return updated;
			});
		},
		clear: () => {
			if (typeof localStorage !== 'undefined') {
				localStorage.clear();
			}
			set({ id: null, name: null });
		},
		isAuthenticated: () => {
			if (typeof localStorage === 'undefined') return false;
			const stored = localStorage.getItem('session');
			if (!stored) return false;
			try {
				const session = JSON.parse(stored);
				return !!session.id;
			} catch (e) {
				return false;
			}
		}
	};
}

export const session = createSession();
