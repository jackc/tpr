class App.Views.RegisterPage extends Backbone.View
  template: _.template($("#register_page_template").html())

  initialize: (options)->
    @registrationService = options.registrationService

  events:
    "submit form" : "register"

  register: (e)->
    e.preventDefault()
    $form = $(e.currentTarget)
    registration =
      name: $form.find("input[name='name']").val()
      password: $form.find("input[name='password']").val()
      passwordConfirmation: $form.find("input[name='passwordConfirmation']").val()
    @registrationService.register(registration)
      .success(@onRegistrationSuccess)
      .fail(@onRegistrationFailure)

  onRegistrationSuccess: ->
    Backbone.history.navigate('home', true)

  onRegistrationFailure: (response)->
    alert response.responseText

  render: ->
    @$el.html @template()
    @
