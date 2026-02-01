import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {conn} from '../connection.js'
import Session from '../session.js'

export default function ResetPasswordPage() {
  const navigate = useNavigate()
  const [token] = useState(window.location.hash.split("=")[1])
  const [password, setPassword] = useState('')

  const handleChange = (e) => {
    setPassword(e.target.value)
  }

  const resetPassword = (e) => {
    e.preventDefault()
    const reset = {
      token: token,
      password: password
    }

    conn.resetPassword(reset, {
      succeeded: (data) => {
        alert("Successfully reset password")
        Session.id = data.sessionID
        Session.name = data.name
        navigate('/home')
      },
      failed: () => {
        alert("Failure resetting password")
      }
    })
  }

  return (
    <div className="lostPassword">
      <header>
        <h1>The Pithy Reader</h1>
      </header>
      <form onSubmit={resetPassword}>
        <dl>
          <dt>
            <label htmlFor="password">Password</label>
          </dt>
          <dd>
            <input
              type="password"
              id="password"
              autoFocus
              value={password}
              onChange={handleChange}
            />
          </dd>
        </dl>

        <input type="submit" value="Reset Password" />
      </form>
    </div>
  )
}
