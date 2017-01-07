import React from 'react'
import ReactDOM from 'react-dom'
import {conn} from '../connection.js'
import ArchivedItems from '../ArchivedItems.js'
import{toTPRString} from '../date.js'

var viewItem = function(item, e) {
  if(e) {
    e.preventDefault()
  }
  window.open(item.url)
}

export default class ArchivePage extends React.Component {
  constructor(props, context) {
    super(props, context)

    this.collection = new ArchivedItems
    this.state = {
      items: [],
      selected: null
    }

    this.keyDown = this.keyDown.bind(this)
    this.selectNext = this.selectNext.bind(this)
    this.selectPrevious = this.selectPrevious.bind(this)
    this.viewSelected = this.viewSelected.bind(this)
    this.ensureSelectedItemVisible = this.ensureSelectedItemVisible.bind(this)
  }

  componentDidMount() {
    this.collection.changed.add(function() {
      this.setState({items: this.collection.items, selected: this.collection.items[0]})
    }.bind(this))
    this.collection.fetch()

    document.addEventListener("keydown", this.keyDown)
  }

  componentWillUnmount() {
    document.removeEventListener("keydown", this.keyDown)
  }

  componentDidUpdate(prevProps, prevState) {
    if(this.state.selected !== prevState.selected) {
      if(prevState.selected) {
        prevState.selected.markRead()
      }
      this.ensureSelectedItemVisible()
    }
  }

  render() {
    return (
      <div className="home">
        <Actions items={this.state.items} markAllReadFn={this.markAllRead} refreshFn={this.refresh} />

        <ul className="unreadItems">
          {
            this.state.items.map(function(item, index) {
              return (
                <ArchivedItem key={item.id} ref={"itemidx-"+index} item={item} selected={item==this.state.selected} />
              )
            }.bind(this))
          }
        </ul>

        {
          this.state.items.length > 15 ?
          (<Actions items={this.state.items} markAllReadFn={this.markAllRead} refreshFn={this.refresh} />) :
          null
        }
      </div>
    );
  }

  keyDown(e) {
    switch(e.which) {
      // j
      case 74:
        this.selectNext()
        break
      // k
      case 75:
        this.selectPrevious()
        break
      // v
      case 86:
        this.viewSelected()
        break
    }
  }

  selectNext() {
    var items = this.state.items
    if(items.length == 0) {
      return
    }

    var idx = items.indexOf(this.state.selected) + 1
    if(idx >= items.length) {
      return
    }

    this.setState({selected: items[idx]})
  }

  selectPrevious() {
    var items = this.state.items

    if(items.length == 0) {
      return
    }

    var idx = items.indexOf(this.state.selected) - 1
    if(idx < 0) {
      return
    }

    this.setState({selected: items[idx]})
  }

  viewSelected() {
    if(this.state.selected) {
      viewItem(this.state.selected)
    }
  }

  ensureSelectedItemVisible() {
    if(!this.state.selected) {
      return
    }

    var items = this.state.items
    var idx = items.indexOf(this.state.selected)
    var component = this.refs["itemidx-"+idx]
    var el = ReactDOM.findDOMNode(component)
    var rect = el.getBoundingClientRect()
    var entirelyVisible = rect.top >= 0 && rect.left >= 0 && rect.bottom <= window.innerHeight && rect.right <= window.innerWidth

    if(!entirelyVisible) {
      el.scrollIntoView()
    }
  }
}

class Actions extends React.Component {
  constructor(props, context) {
    super(props, context)
  }

  render() {
    if(this.props.items.length > 0) {
      return (
        <div className="pageActions">
          <div className="keyboardShortcuts">
            <dl>
              <dt>Move down:</dt>
              <dd>j</dd>
              <dt>Move up:</dt>
              <dd>k</dd>
              <dt>Open selected:</dt>
              <dd>v</dd>
            </dl>
          </div>
        </div>
      )
    } else {
      return (
        <div className="pageActions">
          <p className="noUnread">No archived items as of {toTPRString(new Date())}.</p>
        </div>
      )
    }
  }
}

class ArchivedItem extends React.Component {
  constructor(props, context) {
    super(props, context)
  }

  shouldComponentUpdate(nextProps, nextState) {
    return nextProps.selected !== this.props.selected
  }

  render() {
    return (
      <li className={this.props.selected ? "selected" : ""}>
        <div className="title">
          <a href={this.props.item.url} onClick={viewItem.bind(null, this.props.item)}>{this.props.item.title}</a>
        </div>
        <span className="meta">
          <span className="feedName">{this.props.item.feed_name}</span>
          {' '}
          on
          {' '}
          <time dateTime={this.props.item.publication_time.toISOString()} className="publication">
            {toTPRString(this.props.item.publication_time)}
          </time>
        </span>
      </li>
    )
  }
}
