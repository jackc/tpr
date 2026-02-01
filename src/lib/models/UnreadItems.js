import { writable } from 'svelte/store';
import { api } from '../api.js';
import Item from './Item.js';

export default class UnreadItems {
	constructor() {
		this.changed = writable(0);
		this.items = [];
	}

	async fetch() {
		const data = await api.getUnreadItems();
		this.items = data.map((record) => {
			const model = new Item();
			Object.assign(model, record);
			return model;
		});
		this.changed.update((n) => n + 1);
	}

	async markAllRead() {
		const itemIDs = this.items.map((i) => i.id);
		await api.markAllRead(itemIDs);
		this.changed.update((n) => n + 1);
		await this.fetch();
	}
}
