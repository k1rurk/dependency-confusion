const loadingWindow = document.getElementById('loading-window');
const errorText = document.getElementById('error-text');
const errorWindow = document.getElementById('error-window');
const sendNpm = document.getElementById('send-npm');
const sendPiP = document.getElementById('send-pip');
const packagePiP = document.getElementById('package-pip');
const versionPiP = document.getElementById('version-pip');
const packageNpm = document.getElementById('package-npm');
const versionNpm = document.getElementById('version-npm');

function sendRequest(formData) {
    loadingWindow.style.cssText = "display: flex;";
    errorWindow.style.cssText = "display: none;";
  
    var jsonBody = {
      "name": formData.get("name"),
      "package": formData.get("package"),
      "version": formData.get("version"),
    };
    const requestOptions = {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(jsonBody),
    };
  
    fetch("http://127.0.0.1:9000/api/v1/registry", requestOptions)
      .then(response => {
        loadingWindow.style.cssText = "display: none;";
        if (!response.ok) {
            return response.json().then(response => {throw new Error(response.error)});
        }
        return response.json();
      })
      .then(data => {
        console.log(data)
      })
      .catch(error => {
        console.log(error);
        errorWindow.style.cssText = "display: flex;";
		    errorText.textContent = error.message;
      });
  }


sendNpm.addEventListener('click', function () {
    if (packageNpm.value.length !== 0 && versionNpm.value.length !== 0) {
        const formData = new FormData();
        formData.append("name", "npm");
        formData.append("package", packageNpm.value);
        formData.append("version", versionNpm.value);
        sendRequest(formData);
    }
});

sendPiP.addEventListener('click', function () {
    if (packagePiP.value.length !== 0 && versionPiP.value.length !== 0) {
        const formData = new FormData();
        formData.append("name", "pip");
        formData.append("package", packagePiP.value);
        formData.append("version", versionPiP.value);
        sendRequest(formData);
    }
});