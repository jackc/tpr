import React from 'react';
import {conn} from '../connection.js'
import Session from '../session.js'
import{toTPRString} from '../date.js'

export default class FeedsPage extends React.Component {
  constructor(props, context) {
    super(props, context)

    this.state = {
      feeds: [],
      url: ""
    }

    this.handleChange = this.handleChange.bind(this)
    this.fetch = this.fetch.bind(this)
    this.subscribe = this.subscribe.bind(this)
    this.unsubscribe = this.unsubscribe.bind(this)
    this.import = this.import.bind(this)
  }

  componentDidMount() {
    this.fetch()
  }

  handleChange(name, event) {
    var h = {}
    h[name] = event.target.value
    this.setState(h)
  }

  fetch() {
    conn.getFeeds({
      succeeded: function(data) {
        this.setState({feeds: data})
      }.bind(this)
    })
  }

  render() {
    return (
      <div className="feeds">
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
          <a href={"/api/feeds.xml?session="+Session.id}>Export</a>
        </form>

        <ul>
          {
            this.state.feeds.map(function(feed, index) {
              return (
                <FeedsItem key={feed.url} feed={feed} unsubscribeFn={this.unsubscribe.bind(null, feed)} />
              )
            }.bind(this))
          }
        </ul>
      </div>
    )
  }

  subscribe(e) {
    e.preventDefault()

    conn.subscribe(this.state.url, {
      succeeded: function() {
        this.setState({"url": ""})
        this.fetch()
      }.bind(this)
    })
  }

  unsubscribe(feed, e) {
    e.preventDefault()
    if(confirm("Are you sure you want to unsubscribe from " + feed.name + "?")) {
      conn.deleteSubscription(feed.feed_id)
      var feeds = this.state.feeds.slice(0)
      var idx = feeds.indexOf(feed)
      feeds.splice(idx, 1)
      this.setState({feeds: feeds})
    }
  }

  import(e) {
    e.preventDefault()
    var fd = new FormData(e.target)

    conn.importOPML(fd, {
      succeeded: function() {
        this.fetch()
        alert("import success")
      }.bind(this)
    })
  }
}


class FeedsItem extends React.Component {
  constructor(props, context) {
    super(props, context)
  }

  render() {
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
                <time dateTime={feed.last_publication_time.toISOString()}>{toTPRString(feed.last_publication_time)}</time>
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
  }
}
