(function() {
  "use strict"

  var viewItem = function(item, e) {
    if(e) {
      e.preventDefault()
    }
    item.markRead()
    window.open(item.url)
  }

  App.Views.HomePage = React.createClass({
    getInitialState: function() {
      this.collection = new App.Collections.UnreadItems
      return {
        items: [],
        selected: null
      }
    },

    componentDidMount: function() {
      this.collection.changed.add(function() {
        this.setState({items: this.collection.items, selected: this.collection.items[0]})
      }.bind(this))
      this.collection.fetch()

      document.addEventListener("keydown", this.keyDown)
    },

    componentWillUnmount: function() {
      document.removeEventListener("keydown", this.keyDown)
    },

    componentDidUpdate: function(prevProps, prevState) {
      if(this.state.selected !== prevState.selected) {
        if(prevState.selected) {
          prevState.selected.markRead()
        }
        this.ensureSelectedItemVisible()
      }
    },

    render: function() {
      return (
        <div className="home">
          <App.Views.Actions items={this.state.items} markAllReadFn={this.markAllRead} refreshFn={this.refresh} />

          <ul className="unreadItems">
            {
              this.state.items.map(function(item, index) {
                return (
                  <App.Views.UnreadItem key={item.id} ref={"itemidx-"+index} item={item} selected={item==this.state.selected} />
                )
              }.bind(this))
            }
          </ul>
        </div>
      );
    },

    keyDown: function(e) {
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
        // a
        case 65:
          if(e.shiftKey) {
            this.collection.markAllRead()
          }
          break
      }
    },

    markAllRead: function(e) {
      e.preventDefault()
      this.collection.markAllRead()
    },

    refresh: function(e) {
      e.preventDefault()
      this.collection.fetch()
    },

    selectNext: function() {
      var items = this.state.items
      if(items.length == 0) {
        return
      }

      var idx = items.indexOf(this.state.selected) + 1
      if(idx >= items.length) {
        return
      }

      this.setState({selected: items[idx]})
    },

    selectPrevious: function() {
      var items = this.state.items

      if(items.length == 0) {
        return
      }

      var idx = items.indexOf(this.state.selected) - 1
      if(idx < 0) {
        return
      }

      this.setState({selected: items[idx]})
    },

    viewSelected: function() {
      if(this.state.selected) {
        viewItem(this.state.selected)
      }
    },

    ensureSelectedItemVisible: function() {
      if(!this.state.selected) {
        return
      }

      var items = this.state.items
      var idx = items.indexOf(this.state.selected)
      var component = this.refs["itemidx-"+idx]
      var el = React.findDOMNode(component)
      var rect = el.getBoundingClientRect()
      var entirelyVisible = rect.top >= 0 && rect.left >= 0 && rect.bottom <= window.innerHeight && rect.right <= window.innerWidth

      if(!entirelyVisible) {
        el.scrollIntoView()
      }
    },
  })

  App.Views.Actions = React.createClass({
    render: function() {
      if(this.props.items.length > 0) {
        return (
          <div className="pageActions">
            <a href="#" className="markAllRead button" onClick={this.props.markAllReadFn}>Mark All Read</a>
            <div className="keyboardShortcuts">
              <dl>
                <dt>Move down:</dt>
                <dd>j</dd>
                <dt>Move up:</dt>
                <dd>k</dd>
                <dt>Open selected:</dt>
                <dd>v</dd>
                <dt>Mark all read:</dt>
                <dd>shift+a</dd>
              </dl>
            </div>
          </div>
        )
      } else {
        return (
          <div className="pageActions">
            <a href="#" className="refresh button" onClick={this.props.refreshFn}>Refresh</a>
            <p className="noUnread">No unread items as of {new Date().toTPRString()}.</p>
          </div>
        )
      }
    },

  })

  App.Views.UnreadItem = React.createClass({
    render: function() {
      return (
        <li className={this.props.selected ? "selected" : ""}>
          <div className="title">
            <a href={this.props.item.url} onClick={viewItem.bind(null, this.props.item)}>{this.props.item.title}</a>
          </div>
          <span className="meta">
            <span className="feedName">{this.props.item.feed_name}</span>
            on
            <time dateTime={this.props.item.publication_time.toISOString()} className="publication">
              {this.props.item.publication_time.toTPRString()}
            </time>
          </span>
        </li>
      )
    },
  })
})()
