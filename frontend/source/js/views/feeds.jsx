(function() {
  "use strict"

  App.Views.FeedsPage = React.createClass({
    getInitialState: function() {
      return {
        feeds: []
      }
    },

    componentDidMount: function() {
      this.fetch()
    },

    fetch: function() {
      conn.getFeeds({
        succeeded: function(data) {
          this.setState({feeds: data})
        }.bind(this)
      })
    },

    render: function() {
      return (
        <div className="feeds">
          <App.Views.LoggedInHeaderR />
          <ul>
            {
              this.state.feeds.map(function(feed, index) {
                return (
                  <li>
                    <div className="name"><a href={feed.url}>{feed.name}</a></div>
                    {(function() {
                      if(feed.last_publication_time) {
                        return (
                          <div className="meta">
                            Last published
                            <time datetime={feed.last_publication_time.toISOString()}>{feed.last_publication_time.toTPRString()}</time>
                          </div>
                        )
                      }
                    })()}
                    {(function() {
                      if(feed.failure_count > 0) {
                        return (
                          <div className="error">{feed.last_failure}</div>
                        )
                      }
                    })()}
                    <div className="actions">
                      <a href="#" onClick={this.unsubscribe.bind(null, feed)}>Unsubscribe</a>
                    </div>
                  </li>
                )
              }.bind(this))
            }
          </ul>
        </div>
      );
    },

    unsubscribe: function(feed, e) {
      e.preventDefault()
      if(confirm("Are you sure you want to unsubscribe from " + feed.name + "?")) {
        conn.deleteSubscription(feed.feed_id)
        this.fetch()
      }
    }
  })

  // App.Views.FeedsPage2 = function() {
  //   view.View.call(this, "div")
  //   this.el.className = "feeds"

  //   this.fetch = this.fetch.bind(this)

  //   this.header = this.createChild(App.Views.LoggedInHeader)
  //   this.header.render()

  //   this.subscribeForm = this.createChild(App.Views.SubscribeForm)
  //   this.subscribeForm.render()
  //   this.subscribeForm.subscribed.add(this.fetch)

  //   this.importForm = this.createChild(App.Views.ImportForm)
  //   this.importForm.render()
  //   this.importForm.imported.add(this.fetch)

  //   this.feedsListView = this.createChild(App.Views.FeedsList, {collection: []})

  //   this.fetch()
  // }

  // App.Views.FeedsPage2.prototype = Object.create(view.View.prototype)

  // var p = App.Views.FeedsPage2.prototype
  // p.template = JST["templates/feeds_page"]

  // p.fetch = function() {
  //   conn.getFeeds({
  //     succeeded: function(data) {
  //       this.feedsListView.collection = data
  //       this.feedsListView.render()
  //     }.bind(this)
  //   })
  // }

  // p.render = function() {
  //   this.el.innerHTML = ""
  //   this.el.appendChild(this.header.el)
  //   this.el.appendChild(this.subscribeForm.el)
  //   this.el.appendChild(this.importForm.el)
  //   this.el.appendChild(this.feedsListView.render())
  //   return this.el
  // }

  // App.Views.SubscribeForm = React.createClass({
  //   render: function() {
  //     return (
  //       <form class="subscribe">
  //         <App.Views.LoggedInHeaderR />
  //       </form>
  //     );
  //   },
  // })
  // App.Views.SubscribeForm.prototype = Object.create(view.View.prototype)

  // var p = App.Views.SubscribeForm.prototype
  // p.template = JST["templates/feeds/subscribe"]

  // p.render = function() {
  //   this.el.innerHTML = this.template()
  //   this.listen()
  //   return this.el
  // }

  // p.listen = function() {
  //   this.el.addEventListener("submit", this.subscribe)
  // }

  // p.subscribe = function(e) {
  //   e.preventDefault()

  //   conn.subscribe(this.el.elements.url.value, {
  //     succeeded: function() {
  //       this.el.elements.url.value = ""
  //       this.subscribed.dispatch()
  //     }.bind(this)
  //   })
  // }

  // App.Views.ImportForm = function() {
  //   view.View.call(this, "form")
  //   this.el.className = "import"

  //   this.imported = new signals.Signal()

  //   this.import = this.import.bind(this)
  // }

  // App.Views.ImportForm.prototype = Object.create(view.View.prototype)

  // var p = App.Views.ImportForm.prototype
  // p.template = JST["templates/feeds/import"]

  // p.render = function() {
  //   this.el.innerHTML = this.template()
  //   this.listen()
  //   return this.el
  // }

  // p.listen = function() {
  //   this.el.addEventListener("submit", this.import)
  // }

  // p.import = function(e) {
  //   e.preventDefault()
  //   var fd = new FormData(e.target)

  //   conn.importOPML(fd, {
  //     succeeded: function() {
  //       this.imported.dispatch()
  //       alert("import success")
  //     }.bind(this)
  //   })
  // }
})()
