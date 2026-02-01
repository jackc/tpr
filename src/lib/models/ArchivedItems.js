import { writable } from 'svelte/store';
import { api } from '../api.js';
import Item from './Item.js';

export default class ArchivedItems {
	constructor() {
		this.changed = writable(0);
		this.items = [];
	}

	async fetch() {
		const data = await api.getArchivedItems();
		this.items = data.map((record) => {
			const model = new Item();
			Object.assign(model, record);
			model.isRead = true;
			return model;
		});
		this.changed.update((n) => n + 1);
	}
}
