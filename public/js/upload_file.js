var isAdvancedUpload = function() {
  var div = document.createElement('div');
  return (('draggable' in div) || ('ondragstart' in div && 'ondrop' in div)) && 'FormData' in window && 'FileReader' in window;
}();

let draggableFileArea = document.querySelector(".drag-file-area");
let browseFileText = document.querySelector(".browse-files");
let uploadIcon = document.querySelector(".upload-icon");
let dragDropText = document.querySelector(".dynamic-message");
let fileInput = document.querySelector(".default-file-input");
let cannotUploadMessage = document.querySelector(".cannot-upload-message");
let cancelAlertButton = document.querySelector(".cancel-alert-button");
let uploadedFile = document.querySelector(".file-block");
let fileName = document.querySelector(".file-name");
let fileSize = document.querySelector(".file-size");
let progressBar = document.querySelector(".progress-bar");
let removeFileButton = document.querySelector(".remove-file-icon");
let uploadButton = document.querySelector(".upload-button");
let responseMessage = document.getElementById('dataTable').getElementsByTagName('tbody')[0];
let errorText = document.getElementById('error-text');
let errorWindow = document.getElementById('error-window');
let loadingWindow = document.getElementById('loading-window');
let fileFlag = 0;

fileInput.addEventListener("click", () => {
	fileInput.value = '';
	console.log(fileInput.value);
});

fileInput.addEventListener("change", e => {
	console.log(" > " + fileInput.value)
    console.log(fileInput.files[0].name)
    console.log(document.querySelector(".default-file-input").value);
	uploadIcon.innerHTML = 'check_circle';
	dragDropText.innerHTML = 'File Dropped Successfully!';
	// document.querySelector(".label").innerHTML = `drag & drop or <span class="browse-files"> <input type="file" class="default-file-input" style=""/> <span class="browse-files-text" style="top: 0;"> browse file</span></span>`;
	uploadButton.innerHTML = `Upload`;
	errorWindow.style.cssText = "display: none;";
    
	fileName.innerHTML = fileInput.files[0].name;
	fileSize.innerHTML = (fileInput.files[0].size/1024).toFixed(1) + " KB";

	uploadedFile.style.cssText = "display: flex;";
	progressBar.style.width = 0;
	fileFlag = 0;
    
});

function sendFile(file) {
	const formData = new FormData();
	formData.append("file", file);
	const requestOptions = {
		method: 'POST',
		body: formData,
	};
	loadingWindow.style.cssText = "display: flex;";

	fetch("http://127.0.0.1:9000/api/v1/parser/file", requestOptions)
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
	// let isFileUploaded = fileInput.files[0];
	// const reader = new FileReader();
	// reader.onload = (e) => {
	// 	console.log(e.target.result);
	// };
	// reader.readAsText(isFileUploaded);
	if(fileInput.files.length != 0) {
		if (fileFlag == 0) {
			sendFile(fileInput.files[0]);
    		fileFlag = 1;
    		var width = 0;
    		var id = setInterval(frame, 50);
    		function frame() {
      			if (width >= 360) {
        			clearInterval(id);
					uploadButton.innerHTML = `<span class="material-icons-outlined upload-button-icon"> check_circle </span> Uploaded`;
      			} else {
        			width += 5;
        			progressBar.style.width = width + "px";
      			}
    		}
  		}
	} else {
		cannotUploadMessage.style.cssText = "display: flex; animation: fadeIn linear 1.5s;";
	}
});

cancelAlertButton.addEventListener("click", () => {
	cannotUploadMessage.style.cssText = "display: none;";
});

if(isAdvancedUpload) {
	["drag", "dragstart", "dragend", "dragover", "dragenter", "dragleave", "drop"].forEach( evt => 
		draggableFileArea.addEventListener(evt, e => {
			e.preventDefault();
			e.stopPropagation();
		})
	);

	["dragover", "dragenter"].forEach( evt => {
		draggableFileArea.addEventListener(evt, e => {
			e.preventDefault();
			e.stopPropagation();
			uploadIcon.innerHTML = 'file_download';
			dragDropText.innerHTML = 'Drop your file here!';
		});
	});

    ["dragleave"].forEach( evt => {
		draggableFileArea.addEventListener(evt, e => {
			e.preventDefault();
			e.stopPropagation();
			uploadIcon.innerHTML = 'file_upload';
			dragDropText.innerHTML = 'Drag & drop any file here';
		});
	});

    ["dragleave"].forEach( evt => {
		fileInput.addEventListener(evt, e => {
			e.preventDefault();
			e.stopPropagation();
			uploadIcon.innerHTML = 'file_upload';
			dragDropText.innerHTML = 'Drag & drop any file here';
		});
	});


	draggableFileArea.addEventListener("drop", e => {
		uploadIcon.innerHTML = 'check_circle';
		dragDropText.innerHTML = 'File Dropped Successfully!';
		// document.querySelector(".label").innerHTML = `drag & drop or <span class="browse-files"> <input type="file" class="default-file-input" style=""/> <span class="browse-files-text" style="top: -23px; left: -20px;"> browse file</span> </span>`;
		uploadButton.innerHTML = `Upload`;
		errorWindow.style.cssText = "display: none;";
		
		let files = e.dataTransfer.files;
		fileInput.files = files;
		console.log(files[0].name + " " + files[0].size);
		console.log(document.querySelector(".default-file-input").value);
		fileName.innerHTML = files[0].name;
		fileSize.innerHTML = (files[0].size/1024).toFixed(1) + " KB";
		uploadedFile.style.cssText = "display: flex;";
		progressBar.style.width = 0;
		fileFlag = 0;
	});
}

removeFileButton.addEventListener("click", () => {
    console.log("removeFileButton")
	uploadedFile.style.cssText = "display: none;";
	fileInput.value = '';
	uploadIcon.innerHTML = 'file_upload';
	dragDropText.innerHTML = 'Drag & drop any file here';
	// document.querySelector(".label").innerHTML = `or <span class="browse-files"> <input type="file" class="default-file-input"/> <span class="browse-files-text">browse file</span> <span>from device</span> </span>`;
	uploadButton.innerHTML = `Upload`;
	errorWindow.style.cssText = "display: none;";
});