package: main
imports: github.com/JackC/form
func: RenderSubscribe
parameters: e *environment, f *form.Form
escape: html
---
<% RenderHeader(writer, e) %>
<h1>New Subscription</h1>
<form action="" method="POST">
  <dl>
    <dt>
      <label for="url">Feed URL</label>
      <% if f.Fields["url"].Error != nil { %>
        <span class="error"><%= f.Fields["url"].Error.Error() %></span>
      <% } %>
    </dt>
    <dd><input type="text" name="url" value="<%= f.Fields["url"].Unparsed %>" /></dd>
  </dl>
  <input type="submit" value="Subscribe" />
</form>
<% RenderFooter(writer) %>
