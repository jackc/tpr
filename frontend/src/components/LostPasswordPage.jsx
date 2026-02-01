import React, { useState } from 'react'
import {conn} from '../connection.js'

export default function LostPasswordPage() {
  const [email, setEmail] = useState('')

  const handleChange = (e) => {
    setEmail(e.target.value)
  }

  const requestPasswordReset = (e) => {
    e.preventDefault()
    conn.requestPasswordReset(email, {
      succeeded: () => {
        alert("Please check your email for reset information")
      },
      failed: () => {
        alert("Failure resetting password")
      }
    })
  }

  return (
    <div className="lostPassword">
      <form onSubmit={requestPasswordReset}>
        <dl>
          <dt>
            <label htmlFor="email">Email</label>
          </dt>
          <dd>
            <input
              type="email"
              id="email"
              autoFocus
              value={email}
              onChange={handleChange}
            />
          </dd>
        </dl>

        <input type="submit" value="Reset Password" />
      </form>
    </div>
  )
}
