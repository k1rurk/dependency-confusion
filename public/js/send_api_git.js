const loadingWindow = document.getElementById('loading-window');

function sendRequest(api, nameForm, form) {
    const formData = new FormData(form);
    loadingWindow.style.cssText = "display: flex;";
  
    var jsonBody = {
      [nameForm]: formData.get("search")
    };
    const requestOptions = {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(jsonBody),
    };
  
    fetch(api, requestOptions)
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
  }