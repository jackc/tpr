import signals from 'signals'
import {conn} from './connection.js'
import Item from './Item.js'

export default class UnreadItems {
  constructor() {
    this.changed = new signals.Signal()
    this.items = []
  }

  fetch() {
    conn.getArchivedItems({ succeeded: (data)=> {
      this.items = data.map(function(record) {
        var model = new Item;
        for (var k in record) {
          model[k] = record[k]
        }
        model.isRead = true
        return model
      })
      this.changed.dispatch()
    }})
  }
}
