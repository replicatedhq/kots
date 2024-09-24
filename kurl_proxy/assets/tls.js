function startTLS() {
  let useSelfSigned = true;

  let selfSignedLabels;
  let customCertLabels;
  let certInput;
  let keyInput;

  if (document.readyState !== "loading") {
    ready();
  } else {
    document.addEventListener("DOMContentLoaded", ready);
  }

  function ready() {
    selfSignedLabels = document.getElementsByClassName("self-signed-visible");
    customCertLabels = document.getElementsByClassName("custom-cert-visible");

    document
      .getElementById("self-signed")
      .addEventListener("change", function () {
        const certTypeBox = this.closest(".cert-type-box");
        const allCertBoxes = document.querySelectorAll(".cert-type-box");
        if (this.checked) {
          allCertBoxes.forEach((box) => {
            box.classList.remove("checked-background");
            if (box === certTypeBox) {
              box.classList.add("checked-background");
            }
          });
        }
      });
    document
      .getElementById("custom-cert")
      .addEventListener("change", function () {
        const certTypeBox = this.closest(".cert-type-box");
        const allCertBoxes = document.querySelectorAll(".cert-type-box");
        if (this.checked) {
          allCertBoxes.forEach((box) => {
            box.classList.remove("checked-background");
            if (box === certTypeBox) {
              box.classList.add("checked-background");
            }
          });
        }
      });

    function handleSubmit(e) {
      if (useSelfSigned) {
        skipAndWait(e);
        return;
      }

      uploadAndWait(e);
    }

    var form = document.getElementById("upload-form");
    if (form) {
      form.addEventListener("submit", handleSubmit);
    }

    var skip = document.getElementById("skip-button");
    if (skip) {
      skip.addEventListener("click", skipAndWait);
    }

    const typeToggle = document.getElementsByName("type");

    typeToggle.forEach((el) => {
      el.addEventListener("change", handleTypeToggle);
    });

    keyInput = document.getElementById("key");
    keyLabel = document.getElementById("key-label");

    keyInput.onchange = (e) => {
      keyLabel.innerHTML = e.target.files[0].name;
    };

    certInput = document.getElementById("cert");
    certLabel = document.getElementById("cert-label");

    certInput.onchange = (e) => {
      certLabel.innerHTML = e.target.files[0].name;
    };
  }

  function uploadAndWait(e) {
    e.preventDefault();

    var formData = new FormData();

    var hostnameInput = document.getElementById("hostname");

    formData.append("cert", certInput.files[0]);
    formData.append("key", keyInput.files[0]);
    formData.append("hostname", hostnameInput.value);
    var xhr = new XMLHttpRequest();

    xhr.onerror = function () {
      showError();
      enableForm();
    };

    xhr.onloadend = function () {
      if (xhr.status === 200) {
        redirectAfterRestart(hostnameInput.value, 10);
        return;
      }

      var resp = JSON.parse(xhr.response);
      setErrorMsg(resp.error);

      showError();
      enableForm();
    };

    xhr.open("POST", "/tls");
    xhr.send(formData);
    hideError();
    disableForm();
  }

  function skipAndWait(e) {
    e.stopPropagation();
    e.preventDefault();

    var hostnameInput = document.getElementById("hostname");

    var formData = new FormData();
    formData.append("hostname", hostnameInput.value);

    var xhr = new XMLHttpRequest();

    xhr.onloadend = function () {
      if (xhr.status === 200) {
        redirectAfterRestart(hostnameInput.value, 10);
        return;
      }
      console.log("POST /tls/skip returned status code ", xhr.status);
    };

    xhr.open("POST", "/tls/skip");
    xhr.send(formData);
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

    setTimeout(function () {
      var xhr = new XMLHttpRequest();

      xhr.open("GET", "/tls/meta");
      xhr.send();

      xhr.onloadend = function () {
        if (xhr.status !== 200) {
          redirectAfterRestart(hostname, n - 1);
          return;
        }

        var resp = JSON.parse(xhr.response);

        if (resp.acceptAnonymousUploads) {
          redirectAfterRestart(hostname, n - 1);
          return;
        }

        window.location = url;
      };
    }, 400);
  }

  function setErrorMsg(errorMsg) {
    document.getElementById("tls-error-msg").innerHTML = errorMsg;
  }

  function hideError() {
    document.getElementById("error").style.display = "none";
  }

  function showError() {
    document.getElementById("error").style.display = "";
  }

  function disableForm() {
    document
      .querySelectorAll("#upload-form input,#upload-form button")
      .forEach(function (el) {
        el.disabled = true;
      });
  }

  function enableForm() {
    document
      .querySelectorAll("#upload-form input,#upload-form button")
      .forEach(function (el) {
        el.disabled = false;
      });
  }

  function toggleLabels() {
    useSelfSigned = !useSelfSigned;
    Array.from(selfSignedLabels).forEach(function (el) {
      el.classList.toggle("hidden");
    });
    Array.from(customCertLabels).forEach(function (el) {
      el.classList.toggle("hidden");
    });
  }

  function handleTypeToggle(e) {
    if (e && e.target && e.target.value) {
      if (
        e.target.value === "self-signed" ||
        e.target.value === "custom-cert"
      ) {
        toggleLabels();
      }
    }
  }
}
startTLS();
