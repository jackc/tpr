package: main
imports: github.com/JackC/form
func: RenderRegister
parameters: f *form.Form
escape: html
---
<% RenderHeader(writer) %>
<h1>Registration</h1>
<form action="" method="POST">
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

    <dt>
      <label for="passwordConfirmation">Password Confirmation</label>
      <% if f.Fields["passwordConfirmation"].Error != nil { %>
        <span class="error"><%= f.Fields["passwordConfirmation"].Error.Error() %></span>
      <% } %>
    </dt>
    <dd><input type="password" name="passwordConfirmation" value="<%= f.Fields["passwordConfirmation"].Unparsed %>" /></dd>
  </dl>
  <input type="submit" value="Register" />
</form>
<% RenderFooter(writer) %>