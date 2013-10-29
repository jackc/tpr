package: main
func: RenderErrors
parameters: errors FieldErrors
escape: html
---
<% for _, e := range errors.Errors { %>
  <span class="error"><%= e.Error() %></span>
<% } %>
