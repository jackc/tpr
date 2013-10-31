func: RenderLogin
imports: github.com/JackC/form
escape: html
parameters: f *form.Form
---
<html>
<head>
  <link type="text/css" rel="stylesheet" href="/css/application.css">
  <script src="/js/vendor/jquery-2.0.3.min.js"></script>
  <script src="/js/application.js"></script>
</head>
<body>
<div>
  <div>Login</div>
  <form action="/login" method="POST">
    <dl>
      <dt>
        <label for="name">User name</label>
        <% if f.Fields["name"].Error != nil { %>
          <span class="error"><%= f.Fields["name"].Error.Error() %></span>
        <% } %>
      </dt>
      <dd><input type="text" name="name" value="<%= f.Fields["name"].Unparsed %>" /></dd>

      <dt>
        <label for="password">Password</label>
        <% if f.Fields["password"].Error != nil { %>
          <span class="error"><%= f.Fields["password"].Error.Error() %></span>
        <% } %>
      </dt>
      <dd><input type="password" name="password" value="<%= f.Fields["password"].Unparsed %>"/></dd>
    </dl>

    <input type="submit" value="Login" />
  </form>
  <a href="/register">Create an account</a>
</div>
</body>
</html>
