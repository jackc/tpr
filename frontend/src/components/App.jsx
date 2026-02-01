import React from 'react'
import { Outlet } from 'react-router-dom'
import Header from './Header.jsx'
import WorkingNotice from './WorkingNotice.jsx'

export default function App() {
  return (
    <div>
      <WorkingNotice />
      <Header />
      <Outlet />
    </div>
  )
}
