
let draggableFileArea = document.querySelector(".drag-file-area");
let uploadIcon = document.querySelector(".upload-icon");
let dragDropText = document.querySelector(".browse-files-text");
let fileInput = document.querySelector(".default-file-input");
let cannotUploadMessage = document.querySelector(".cannot-upload-message");
let uploadButton = document.querySelector(".upload-button");
let responseMessage = document.getElementById('dataTable').getElementsByTagName('tbody')[0];
let errorText = document.getElementById('error-text');
let errorWindow = document.getElementById('error-window');
let cancelAlertButton = document.querySelector(".cancel-alert-button");
let uploadedFile = document.querySelector(".file-block");
let loadingWindow = document.getElementById('loading-window');
let fileFlag = 0;

fileInput.addEventListener("click", () => {
	fileInput.value = '';
	console.log(fileInput.value);
});

fileInput.addEventListener("change", e => {
    e.preventDefault()
	// console.log(" > " + fileInput.value)
    console.log(fileInput.files.length)
    console.log(document.querySelector(".default-file-input").value);
	uploadIcon.innerHTML = 'check_circle';
	dragDropText.textContent = 'File Dropped Successfully!';
	uploadButton.innerHTML = `Upload`;
	errorWindow.style.cssText = "display: none;";
    
	fileFlag = 0;
});

function sendFile(formData) {
	loadingWindow.style.cssText = "display: flex;";
	const requestOptions = {
		method: 'POST',
		body: formData,
	};

	fetch("http://127.0.0.1:9000/api/v1/parser/directory", requestOptions)
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
    })
	
}

uploadButton.addEventListener("click", () => {
	if(fileInput.files.length != 0) {
		if (fileFlag == 0) {
            var fd = new FormData();
            for (var x = 0; x < fileInput.files.length; x++) {
                fd.append("files[]", fileInput.files[x]);
            }
			sendFile(fd);
    		fileFlag = 1;
            uploadButton.innerHTML = `<span class="material-icons-outlined upload-button-icon"> check_circle </span> Uploaded`;
  		}
	} else {
		cannotUploadMessage.style.cssText = "display: flex; animation: fadeIn linear 1.5s;";
	}
});

cancelAlertButton.addEventListener("click", () => {
	cannotUploadMessage.style.cssText = "display: none;";
});