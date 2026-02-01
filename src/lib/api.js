import { writable, get } from 'svelte/store';
import { session } from './session.js';

export const ajaxPending = writable(0);

class APIClient {
	async request(url, method = 'GET', options = {}) {
		ajaxPending.update(n => n + 1);

		try {
			const currentSession = get(session);
			const headers = {
				'Content-Type': 'application/json',
				...options.headers
			};

			// Add auth header if session exists
			if (currentSession.id) {
				headers['X-Authentication'] = currentSession.id;
			}

			// Don't set Content-Type for FormData (browser will set it with boundary)
			if (options.body instanceof FormData) {
				delete headers['Content-Type'];
			}

			const response = await fetch(url, {
				method,
				headers,
				body: options.body
			});

			const contentType = response.headers.get('Content-Type');
			const data = contentType?.includes('json')
				? await response.json()
				: await response.text();

			// Handle session expiration
			if (response.status === 403 && data === 'Bad or missing X-Authentication header') {
				session.clear();
				if (typeof window !== 'undefined') {
					window.location.href = '/login';
					return; // Stop execution after redirect
				}
			}

			if (!response.ok) {
				throw { data, response, status: response.status };
			}

			return data;
		} finally {
			ajaxPending.update(n => n - 1);
		}
	}

	async get(url) {
		return this.request(url, 'GET');
	}

	async post(url, data) {
		const body = data instanceof FormData ? data : JSON.stringify(data);
		return this.request(url, 'POST', { body });
	}

	async patch(url, data) {
		return this.request(url, 'PATCH', { body: JSON.stringify(data) });
	}

	async delete(url) {
		return this.request(url, 'DELETE');
	}

	// Authentication endpoints
	async login(credentials) {
		return this.post('/api/sessions', credentials);
	}

	async logout() {
		const currentSession = get(session);
		return this.delete(`/api/sessions/${currentSession.id}`);
	}

	async register(registration) {
		return this.post('/api/register', registration);
	}

	async requestPasswordReset(email) {
		return this.post('/api/request_password_reset', { email });
	}

	async resetPassword(reset) {
		return this.post('/api/reset_password', reset);
	}

	// Account endpoints
	async getAccount() {
		return this.get('/api/account');
	}

	async updateAccount(update) {
		return this.patch('/api/account', update);
	}

	// Feed endpoints
	async getFeeds() {
		const data = await this.get('/api/feeds');
		// Convert Unix timestamps to Date objects
		return data.map(feed => {
			['last_fetch_time', 'last_failure_time', 'last_publication_time'].forEach(name => {
				if (feed[name]) {
					feed[name] = new Date(feed[name] * 1000);
				}
			});
			return feed;
		});
	}

	async subscribe(url) {
		return this.post('/api/subscriptions', { url });
	}

	async importOPML(formData) {
		return this.post('/api/feeds/import', formData);
	}

	async deleteSubscription(feedID) {
		return this.delete(`/api/subscriptions/${feedID}`);
	}

	// Item endpoints
	async getUnreadItems() {
		const data = await this.get('/api/items/unread');
		// Convert Unix timestamps to Date objects
		return data.map(item => ({
			...item,
			publication_time: new Date(item.publication_time * 1000)
		}));
	}

	async markItemRead(itemID) {
		return this.delete(`/api/items/unread/${itemID}`);
	}

	async markAllRead(itemIDs) {
		return this.post('/api/items/unread/mark_multiple_read', { itemIDs });
	}

	async getArchivedItems() {
		const data = await this.get('/api/items/archived');
		// Convert Unix timestamps to Date objects
		return data.map(item => ({
			...item,
			publication_time: new Date(item.publication_time * 1000)
		}));
	}
}

export const api = new APIClient();
