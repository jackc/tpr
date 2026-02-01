import React from 'react'
import { createRoot } from 'react-dom/client'
import { HashRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom'
import App from './components/App.jsx'
import LoginPage from './components/LoginPage.jsx'
import HomePage from './components/HomePage.jsx'
import ArchivePage from './components/ArchivePage.jsx'
import FeedsPage from './components/FeedsPage.jsx'
import AccountPage from './components/AccountPage.jsx'
import LostPasswordPage from './components/LostPasswordPage.jsx'
import ResetPasswordPage from './components/ResetPasswordPage.jsx'
import RegisterPage from './components/RegisterPage.jsx'
import Session from './session.js'

import styles from './main.scss'

// Protected Route Component - redirects to login if not authenticated
function ProtectedRoute({ children }) {
  const location = useLocation()

  if (!Session.isAuthenticated()) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }
  return children
}

const root = createRoot(document.getElementById('view'))
root.render(
  <React.StrictMode>
    <HashRouter>
      <Routes>
        <Route path="/" element={<App />}>
          <Route index element={<ProtectedRoute><HomePage /></ProtectedRoute>} />
          <Route path="login" element={<LoginPage />} />
          <Route path="home" element={<ProtectedRoute><HomePage /></ProtectedRoute>} />
          <Route path="archive" element={<ProtectedRoute><ArchivePage /></ProtectedRoute>} />
          <Route path="feeds" element={<ProtectedRoute><FeedsPage /></ProtectedRoute>} />
          <Route path="account" element={<ProtectedRoute><AccountPage /></ProtectedRoute>} />
          <Route path="register" element={<RegisterPage />} />
          <Route path="lostPassword" element={<LostPasswordPage />} />
          <Route path="resetPassword" element={<ResetPasswordPage />} />
        </Route>
      </Routes>
    </HashRouter>
  </React.StrictMode>
)
