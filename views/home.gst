func: RenderHome
parameters: e *environment, items []homeUnreadItem
escape: html
---
<% RenderHeader(writer, e) %>
<div>
  <a href="/subscribe">Subscribe</a>
</div>
<h1>Unread Items</h1>
<ul>
  <% for _, item := range items { %>
    <li>
      <div class="feedName"><%= item.feedName %></div>
      <div class="title">
        <a href="<%= item.url %>"><%= item.title %></a>
      </div>
      <div class="publicationTime"><%= item.publicationTime.String() %></div>
    </li>
  <% } %>
</ul>
<% RenderFooter(writer) %>
