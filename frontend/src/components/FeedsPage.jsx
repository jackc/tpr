import React, { useState, useEffect } from 'react'
import {conn} from '../connection.js'
import Session from '../session.js'
import {toTPRString} from '../date.js'

export default function FeedsPage() {
  const [feeds, setFeeds] = useState([])
  const [url, setUrl] = useState("")

  useEffect(() => {
    fetchFeeds()
  }, [])

  const fetchFeeds = () => {
    conn.getFeeds({
      succeeded: (data) => {
        setFeeds(data)
      }
    })
  }

  const handleChange = (e) => {
    setUrl(e.target.value)
  }

  const subscribe = (e) => {
    e.preventDefault()

    conn.subscribe(url, {
      succeeded: () => {
        setUrl("")
        fetchFeeds()
      }
    })
  }

  const unsubscribe = (feed, e) => {
    e.preventDefault()
    if(confirm("Are you sure you want to unsubscribe from " + feed.name + "?")) {
      conn.deleteSubscription(feed.feed_id)
      setFeeds(prevFeeds => prevFeeds.filter(f => f !== feed))
    }
  }

  const importOPML = (e) => {
    e.preventDefault()
    const fd = new FormData(e.target)

    conn.importOPML(fd, {
      succeeded: () => {
        fetchFeeds()
        alert("import success")
      }
    })
  }

  return (
    <div className="feeds">
      <form className="subscribe" onSubmit={subscribe}>
        <dl>
          <dt>
            <label htmlFor="feed_url">Feed URL</label>
          </dt>
          <dd>
            <input
              type="text"
              id="feed_url"
              value={url}
              onChange={handleChange}
            />
          </dd>
        </dl>
        <input type="submit" value="Subscribe" />
      </form>

      <form className="import" onSubmit={importOPML}>
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
        {feeds.map((feed) => (
          <FeedsItem
            key={feed.url}
            feed={feed}
            unsubscribeFn={(e) => unsubscribe(feed, e)}
          />
        ))}
      </ul>
    </div>
  )
}

function FeedsItem({ feed, unsubscribeFn }) {
  return (
    <li>
      <div className="name"><a href={feed.url}>{feed.name}</a></div>
      {feed.last_publication_time && (
        <div className="meta">
          Last published
          {' '}
          <time dateTime={feed.last_publication_time.toISOString()}>
            {toTPRString(feed.last_publication_time)}
          </time>
        </div>
      )}
      {feed.failure_count > 0 && (
        <div className="error">{feed.last_failure}</div>
      )}
      <div className="actions">
        <a href="#" onClick={unsubscribeFn}>Unsubscribe</a>
      </div>
    </li>
  )
}
