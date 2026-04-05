// Handle HX-Redirect responses globally
document.addEventListener('htmx:afterRequest', function(evt) {
  var xhr = evt.detail.xhr;
  if (xhr) {
    var redirect = xhr.getResponseHeader('HX-Redirect');
    if (redirect) {
      window.location.href = redirect;
    }
  }
});
