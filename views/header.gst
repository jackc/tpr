package: main
func: RenderHeader
parameters: e *environment
escape: html
---
<html>
<head>
  <link type="text/css" rel="stylesheet" href="/css/application.css">
  <script src="/js/vendor/jquery-2.0.3.min.js"></script>
  <script src="/js/application.js"></script>
</head>
<body>

<header>
  <h1>Reader</h1>

  <p>Welcome <%= e.CurrentAccount().name %></p>

  <form action="/logout" method="POST">
    <input type="submit" value="Logout" />
  </form>
</header>
