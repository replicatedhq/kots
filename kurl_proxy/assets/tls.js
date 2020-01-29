if (document.readyState !== "loading") {
  ready();
} else {
  document.addEventListener('DOMContentLoaded', ready);
}

function ready() {
  var form = document.getElementById("upload-form");
  if (form) {
    form.addEventListener("submit", uploadAndWait);
  }

  var skip = document.getElementById("skip-button");
  if (skip) {
    skip.addEventListener("click", skipAndWait);
  }
}

function uploadAndWait(e) {
  e.preventDefault();

  var formData = new FormData();

  var certInput = document.getElementById("cert");
  var keyInput = document.getElementById("key");
  var hostnameInput = document.getElementById("hostname");

  formData.append("cert", certInput.files[0]);
  formData.append("key", keyInput.files[0]);

  var xhr = new XMLHttpRequest();


  xhr.onerror = function() {
    showError();
    enableForm();
  }

  xhr.onloadend = function() {
    if (xhr.status === 200) {
      redirectAfterRestart(hostnameInput.value, 10);
      return;
    }

    showError();
    enableForm();
  }

  xhr.open("POST", "/tls");
  xhr.send(formData);
  hideError();
  disableForm();
}

function skipAndWait(e) {
  e.stopPropagation();
  e.preventDefault();

  var hostnameInput = document.getElementById("hostname");

  var xhr = new XMLHttpRequest();

  xhr.onloadend = function() {
    if (xhr.status === 200) {
      redirectAfterRestart(hostnameInput.value, 10);
      return;
    }
    console.log("POST /tls/skip returned status code ", xhr.status);
  };

  xhr.open("POST", "/tls/skip");
  xhr.send();
  hideError();
  disableForm();
}

function redirectAfterRestart(hostname, n) {
  var url = window.location.origin;

  if (hostname) {
    url = "https://" + hostname + ":" + window.location.port;
  }

  // Errors are expected because the server is restarting, but the errors could also be due to the
  // user uploading a certificate that the browser does not trust. It's not possible to detect the
  // cause of the error, so proceed with redirect after some time
  if (n === 0) {
    window.location = url;
    return;
  }

  setTimeout(function() {
    var xhr = new XMLHttpRequest();

    xhr.open("GET", "/tls/meta");
    xhr.send();

    xhr.onloadend = function() {
      if (xhr.status !== 200) {
        redirectAfterRestart(hostname, n-1);
        return;
      }

      var resp = JSON.parse(xhr.response);

      if (resp.acceptAnonymousUploads) {
        redirectAfterRestart(hostname, n-1);
        return;
      }

      window.location = url;
    };
  }, 400);
}

function hideError() {
  document.getElementById("error").style.display = 'none';
}

function showError() {
  document.getElementById("error").style.display = '';
}

function disableForm() {
  document.querySelectorAll("#upload-form input,#upload-form button").forEach(function(el) {
    el.disabled = true;
  });
}

function enableForm() {
  document.querySelectorAll("#upload-form input,#upload-form button").forEach(function(el) {
    el.disabled = false;
  });
}
