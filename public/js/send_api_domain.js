const apiUrlDomain = 'http://127.0.0.1:9000/api/v1/domain';

const inputFormRetries = document.getElementById('retries');
const inputFormThreads = document.getElementById('threads');
const inputFormTimeout = document.getElementById('timeout');
const contactFormDomain = document.getElementById('search-domain');
const responseMessage = document.getElementById('dataTable').getElementsByTagName('tbody')[0];
const errorText = document.getElementById('error-text');
const errorWindow = document.getElementById('error-window');
const loadingWindow = document.getElementById('loading-window');


contactFormDomain.addEventListener('submit', function (event) {
    event.preventDefault();
    errorWindow.style.cssText = "display: none;";
    const formData = new FormData(contactFormDomain);
    loadingWindow.style.cssText = "display: flex;";
    var jsonBody = {
      "domain": formData.get("search"),
      "retries": parseInt(inputFormRetries.value),
      "threads": parseInt(inputFormThreads.value),
      "timeout": parseInt(inputFormTimeout.value),
    };

    const requestOptions = {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(jsonBody),
    };
  
    fetch(apiUrlDomain, requestOptions)
      .then(response => {
        loadingWindow.style.cssText = "display: none;";
        if (!response.ok) {
            return response.json().then(response => {throw new Error(response.error)});
        }
        return response.json();
      })
      .then(data => {
        if (data.data == null) {
            console.log("Proved")
            throw new Error("Not vulnerable");
        }
        for (let index = 0; index < data.data.length; index++){
          //insert Row
          responseMessage.insertRow().innerHTML =
          "<td>" +data.data[index].name+ "</td>"+
          "<td>" +data.data[index].package+ "</td>"+
          "<td>" +data.data[index].version+ "</td>";
        }
      })
      .catch(error => {
        console.log(error);
        errorWindow.style.cssText = "display: flex;";
		errorText.textContent = error.message;
      });
  });