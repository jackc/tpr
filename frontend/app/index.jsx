import React from 'react'
import ReactDOM from 'react-dom'
import { Router, Route, IndexRoute, Link, hashHistory } from 'react-router'
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

require('./main.scss');


function requireAuth(nextState, replace) {
  if (!Session.isAuthenticated()) {
    replace({
      pathname: '/login',
      state: { nextPathname: nextState.location.pathname }
    })
  }
}

ReactDOM.render((
  <Router history={hashHistory}>
    <Route path="/" component={App}>
      <IndexRoute component={HomePage} onEnter={requireAuth} />
      <Route path="/login" component={LoginPage} />
      <Route path="/home" component={HomePage} onEnter={requireAuth} />
      <Route path="/archive" component={ArchivePage} onEnter={requireAuth} />
      <Route path="/feeds" component={FeedsPage} onEnter={requireAuth} />
      <Route path="/account" component={AccountPage} onEnter={requireAuth} />
      <Route path="/register" component={RegisterPage} />
      <Route path="/lostPassword" component={LostPasswordPage} />
      <Route path="/resetPassword" component={ResetPasswordPage} />
    </Route>
  </Router>
), document.getElementById('view'))

