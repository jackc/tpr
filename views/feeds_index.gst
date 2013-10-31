package: main
func: RenderFeedsIndex
parameters: e *environment, feeds []feedIndexFeed
escape: html
---
<% RenderHeader(writer, e) %>
<h1>Feeds</h1>
<ul>
  <% for _, feed := range feeds { %>
    <li><%= feed.name %></li>
  <% } %>
</ul>
<% RenderFooter(writer) %>
