import {conn} from './connection.js'

export default class Item {
  markRead() {
    if(this.isRead) {
      return;
    }
    conn.markItemRead(this.id);
    this.isRead = true;
  }
}
