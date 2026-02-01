import React, { useState, useEffect, useRef, useCallback } from 'react'
import UnreadItems from '../UnreadItems.js'
import {toTPRString} from '../date.js'

const viewItem = (item, e) => {
  if(e) {
    e.preventDefault()
  }
  item.markRead()
  window.open(item.url)
}

export default function HomePage() {
  const [items, setItems] = useState([])
  const [selected, setSelected] = useState(null)
  const itemRefs = useRef([])
  const collectionRef = useRef(new UnreadItems())
  const prevSelectedRef = useRef(null)

  // Effect 1: Collection subscription (runs once)
  useEffect(() => {
    const collection = collectionRef.current

    const handleChange = () => {
      setItems(collection.items)
      setSelected(collection.items[0])
    }

    collection.changed.add(handleChange)
    collection.fetch()

    return () => {
      collection.changed.remove(handleChange)
    }
  }, [])

  // Effect 2: Keyboard navigation (depends on items/selected)
  useEffect(() => {
    const handleKeyDown = (e) => {
      switch(e.which) {
        case 74: // j
          selectNext()
          break
        case 75: // k
          selectPrevious()
          break
        case 86: // v
          viewSelected()
          break
        case 65: // a
          if(e.shiftKey) {
            collectionRef.current.markAllRead()
          }
          break
      }
    }

    document.addEventListener("keydown", handleKeyDown)
    return () => document.removeEventListener("keydown", handleKeyDown)
  }, [items, selected])

  // Effect 3: Selection change side effects
  useEffect(() => {
    if (prevSelectedRef.current && prevSelectedRef.current !== selected) {
      prevSelectedRef.current.markRead()
      ensureSelectedItemVisible()
    }
    prevSelectedRef.current = selected
  }, [selected])

  const markAllRead = (e) => {
    e.preventDefault()
    collectionRef.current.markAllRead()
  }

  const refresh = (e) => {
    e.preventDefault()
    collectionRef.current.fetch()
  }

  const selectNext = useCallback(() => {
    if(items.length === 0) return

    const idx = items.indexOf(selected) + 1
    if(idx >= items.length) return

    setSelected(items[idx])
  }, [items, selected])

  const selectPrevious = useCallback(() => {
    if(items.length === 0) return

    const idx = items.indexOf(selected) - 1
    if(idx < 0) return

    setSelected(items[idx])
  }, [items, selected])

  const viewSelected = useCallback(() => {
    if(selected) {
      viewItem(selected)
    }
  }, [selected])

  const ensureSelectedItemVisible = useCallback(() => {
    if(!selected) return

    const idx = items.indexOf(selected)
    const el = itemRefs.current[idx]

    if (!el) return

    const rect = el.getBoundingClientRect()
    const entirelyVisible = rect.top >= 0 &&
                           rect.left >= 0 &&
                           rect.bottom <= window.innerHeight &&
                           rect.right <= window.innerWidth

    if(!entirelyVisible) {
      el.scrollIntoView()
    }
  }, [items, selected])

  return (
    <div className="home">
      <Actions
        items={items}
        markAllReadFn={markAllRead}
        refreshFn={refresh}
      />

      <ul className="unreadItems">
        {items.map((item, index) => (
          <UnreadItem
            key={item.id}
            ref={(el) => itemRefs.current[index] = el}
            item={item}
            selected={item === selected}
          />
        ))}
      </ul>

      {items.length > 15 && (
        <Actions
          items={items}
          markAllReadFn={markAllRead}
          refreshFn={refresh}
        />
      )}
    </div>
  )
}

function Actions({ items, markAllReadFn, refreshFn }) {
  if(items.length > 0) {
    return (
      <div className="pageActions">
        <a href="#" className="markAllRead button" onClick={markAllReadFn}>
          Mark All Read
        </a>
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
        <a href="#" className="refresh button" onClick={refreshFn}>Refresh</a>
        <p className="noUnread">No unread items as of {toTPRString(new Date())}.</p>
      </div>
    )
  }
}

const UnreadItem = React.memo(React.forwardRef(({ item, selected }, ref) => {
  return (
    <li ref={ref} className={selected ? "selected" : ""}>
      <div className="title">
        <a href={item.url} onClick={(e) => viewItem(item, e)}>{item.title}</a>
      </div>
      <span className="meta">
        <span className="feedName">{item.feed_name}</span>
        {' '}
        on
        {' '}
        <time dateTime={item.publication_time.toISOString()} className="publication">
          {toTPRString(item.publication_time)}
        </time>
      </span>
    </li>
  )
}))
