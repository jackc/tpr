(function() {
  "use strict"

  App.Views.FeedsPage = React.createClass({
    getInitialState: function() {
      return {
        feeds: [],
        url: ""
      }
    },

    componentDidMount: function() {
      this.fetch()
    },

    handleChange: function(name, event) {
      var h = {}
      h[name] = event.target.value
      this.setState(h)
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
          <App.Views.LoggedInHeader />

          <form className="subscribe" onSubmit={this.subscribe}>
            <dl>
              <dt>
                <label htmlFor="feed_url">Feed URL</label>
              </dt>
              <dd><input type="text" id="feed_url" value={this.state.url} onChange={this.handleChange.bind(null, "url")} /></dd>
            </dl>
            <input type="submit" value="Subscribe" />
          </form>

          <form className="import" onSubmit={this.import}>
            <dl>
              <dt>
                <label htmlFor="opml_file">OPML File</label>
              </dt>
              <dd><input type="file" name="file" id="opml_file" /></dd>
            </dl>
            <input type="submit" value="Import" />
            {' '}
            <a href={"/api/feeds.xml?session="+State.Session.id}>Export</a>
          </form>

          <ul>
            {
              this.state.feeds.map(function(feed, index) {
                return (
                  <App.Views.FeedsItem key={feed.url} feed={feed} unsubscribeFn={this.unsubscribe.bind(null, feed)} />
                )
              }.bind(this))
            }
          </ul>
        </div>
      );
    },

    subscribe: function(e) {
      e.preventDefault()

      conn.subscribe(this.state.url, {
        succeeded: function() {
          this.setState({"url": ""})
          this.fetch()
        }.bind(this)
      })
    },

    unsubscribe: function(feed, e) {
      e.preventDefault()
      if(confirm("Are you sure you want to unsubscribe from " + feed.name + "?")) {
        conn.deleteSubscription(feed.feed_id)
        var feeds = this.state.feeds.slice(0)
        var idx = feeds.indexOf(feed)
        feeds.splice(idx, 1)
        this.setState({feeds: feeds})
      }
    },

    import: function(e) {
      e.preventDefault()
      var fd = new FormData(e.target)

      conn.importOPML(fd, {
        succeeded: function() {
          this.fetch()
          alert("import success")
        }.bind(this)
      })
    }
  })

  App.Views.FeedsItem = React.createClass({
    render: function() {
      var feed = this.props.feed
      var unsubscribeFn = this.props.unsubscribeFn

      return (
        <li>
          <div className="name"><a href={feed.url}>{feed.name}</a></div>
          {(function() {
            if(feed.last_publication_time) {
              return (
                <div className="meta">
                  Last published
                  {' '}
                  <time dateTime={feed.last_publication_time.toISOString()}>{feed.last_publication_time.toTPRString()}</time>
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
            <a href="#" onClick={unsubscribeFn}>Unsubscribe</a>
          </div>
        </li>
      )
    },
  })
})()
