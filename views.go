package main

import (
	"github.com/JackC/form"
	"html"
	"io"
)

func RenderFeedsIndex(writer io.Writer, e *environment, feeds []feedIndexFeed) (err error) {
	RenderHeader(writer, e)
	io.WriteString(writer, `
<h1>Feeds</h1>
<ul>
  `)
	for _, feed := range feeds {
		io.WriteString(writer, `
    <li>`)
		io.WriteString(writer, html.EscapeString(feed.name))
		io.WriteString(writer, `</li>
  `)
	}
	io.WriteString(writer, `
</ul>
`)
	RenderFooter(writer)
	io.WriteString(writer, `
`)

	return
}
func RenderErrors(writer io.Writer, errors FieldErrors) (err error) {
	for _, e := range errors.Errors {
		io.WriteString(writer, `
  <span class="error">`)
		io.WriteString(writer, html.EscapeString(e.Error()))
		io.WriteString(writer, `</span>
`)
	}
	io.WriteString(writer, `
`)

	return
}
func RenderFooter(writer io.Writer) (err error) {
	io.WriteString(writer, `</body>
</html>
`)

	return
}
func RenderHeader(writer io.Writer, e *environment) (err error) {
	io.WriteString(writer, `<html>
<head>
  <link type="text/css" rel="stylesheet" href="/css/application.css">
  <script src="/js/vendor/jquery-2.0.3.min.js"></script>
  <script src="/js/application.js"></script>
</head>
<body>

<header>
  <h1>Reader</h1>

  <p>Welcome `)
	io.WriteString(writer, html.EscapeString(e.CurrentAccount().name))
	io.WriteString(writer, `</p>

  <form action="/logout" method="POST">
    <input type="submit" value="Logout" />
  </form>
</header>
`)

	return
}
func RenderLogin(writer io.Writer, f *form.Form) (err error) {
	io.WriteString(writer, `<html>
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
        `)
	if f.Fields["name"].Error != nil {
		io.WriteString(writer, `
          <span class="error">`)
		io.WriteString(writer, html.EscapeString(f.Fields["name"].Error.Error()))
		io.WriteString(writer, `</span>
        `)
	}
	io.WriteString(writer, `
      </dt>
      <dd><input type="text" id="name" name="name" value="`)
	io.WriteString(writer, html.EscapeString(f.Fields["name"].Unparsed))
	io.WriteString(writer, `" /></dd>

      <dt>
        <label for="password">Password</label>
        `)
	if f.Fields["password"].Error != nil {
		io.WriteString(writer, `
          <span class="error">`)
		io.WriteString(writer, html.EscapeString(f.Fields["password"].Error.Error()))
		io.WriteString(writer, `</span>
        `)
	}
	io.WriteString(writer, `
      </dt>
      <dd><input type="password" id="password" name="password" value="`)
	io.WriteString(writer, html.EscapeString(f.Fields["password"].Unparsed))
	io.WriteString(writer, `"/></dd>
    </dl>

    <input type="submit" value="Login" />
  </form>
  <a href="/register">Create an account</a>
</div>
</body>
</html>
`)

	return
}
func RenderRegister(writer io.Writer, f *form.Form) (err error) {
	io.WriteString(writer, `<html>
<head>
  <link type="text/css" rel="stylesheet" href="/css/application.css">
  <script src="/js/vendor/jquery-2.0.3.min.js"></script>
  <script src="/js/application.js"></script>
</head>
<body>
<h1>Registration</h1>
<form action="" method="POST">
  <dl>
    <dt>
      <label for="name">User name</label>
      `)
	if f.Fields["name"].Error != nil {
		io.WriteString(writer, `
        <span class="error">`)
		io.WriteString(writer, html.EscapeString(f.Fields["name"].Error.Error()))
		io.WriteString(writer, `</span>
      `)
	}
	io.WriteString(writer, `
    </dt>
    <dd><input type="text" id="name" name="name" value="`)
	io.WriteString(writer, html.EscapeString(f.Fields["name"].Unparsed))
	io.WriteString(writer, `" /></dd>

    <dt>
      <label for="password">Password</label>
      `)
	if f.Fields["password"].Error != nil {
		io.WriteString(writer, `
        <span class="error">`)
		io.WriteString(writer, html.EscapeString(f.Fields["password"].Error.Error()))
		io.WriteString(writer, `</span>
      `)
	}
	io.WriteString(writer, `
    </dt>
    <dd><input type="password" id="password" name="password" value="`)
	io.WriteString(writer, html.EscapeString(f.Fields["password"].Unparsed))
	io.WriteString(writer, `"/></dd>

    <dt>
      <label for="passwordConfirmation">Password Confirmation</label>
      `)
	if f.Fields["passwordConfirmation"].Error != nil {
		io.WriteString(writer, `
        <span class="error">`)
		io.WriteString(writer, html.EscapeString(f.Fields["passwordConfirmation"].Error.Error()))
		io.WriteString(writer, `</span>
      `)
	}
	io.WriteString(writer, `
    </dt>
    <dd><input type="password" id="passwordConfirmation" name="passwordConfirmation" value="`)
	io.WriteString(writer, html.EscapeString(f.Fields["passwordConfirmation"].Unparsed))
	io.WriteString(writer, `" /></dd>
  </dl>
  <input type="submit" value="Register" />
</form>
</body>
</html>
`)

	return
}
func RenderSubscribe(writer io.Writer, e *environment, f *form.Form) (err error) {
	RenderHeader(writer, e)
	io.WriteString(writer, `
<h1>New Subscription</h1>
<form action="" method="POST">
  <dl>
    <dt>
      <label for="url">Feed URL</label>
      `)
	if f.Fields["url"].Error != nil {
		io.WriteString(writer, `
        <span class="error">`)
		io.WriteString(writer, html.EscapeString(f.Fields["url"].Error.Error()))
		io.WriteString(writer, `</span>
      `)
	}
	io.WriteString(writer, `
    </dt>
    <dd><input type="text" name="url" value="`)
	io.WriteString(writer, html.EscapeString(f.Fields["url"].Unparsed))
	io.WriteString(writer, `" /></dd>
  </dl>
  <input type="submit" value="Subscribe" />
</form>
`)
	RenderFooter(writer)
	io.WriteString(writer, `
`)

	return
}
