import signals from 'signals'
import {conn} from './connection.js'
import Item from './Item.js'

export default class UnreadItems {
  constructor() {
    this.changed = new signals.Signal()
    this.items = []
  }

  fetch() {
    conn.getUnreadItems({ succeeded: (data)=> {
      this.items = data.map(function(record) {
        var model = new Item;
        for (var k in record) {
          model[k] = record[k]
        }
        return model
      })
      this.changed.dispatch()
    }})
  }

  markAllRead() {
    var itemIDs = this.items.map(function(i) { return i.id })
    conn.markAllRead(itemIDs, { succeeded: ()=> {
      this.changed.dispatch()
      this.fetch()
    }})
  }
}
