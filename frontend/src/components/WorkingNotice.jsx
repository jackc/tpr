import React, { useState, useEffect } from 'react'
import {conn} from '../connection.js'

export default function WorkingNotice() {
  const [display, setDisplay] = useState("none")

  useEffect(() => {
    const handleStart = () => setDisplay("")
    const handleFinish = () => setDisplay("none")

    conn.firstAjaxStarted.add(handleStart)
    conn.lastAjaxFinished.add(handleFinish)

    // CRITICAL: Clean up signal listeners to prevent memory leaks
    return () => {
      conn.firstAjaxStarted.remove(handleStart)
      conn.lastAjaxFinished.remove(handleFinish)
    }
  }, [])

  return (
    <div id="working_notice" style={{display}}>
      <div>Working...</div>
    </div>
  )
}
