import { api } from '../api.js';

export default class Item {
	markRead() {
		if (this.isRead) {
			return;
		}
		api.markItemRead(this.id);
		this.isRead = true;
	}
}
