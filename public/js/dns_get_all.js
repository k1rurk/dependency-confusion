const responseMessage = document.getElementById('dataTable').getElementsByTagName('tbody')[0];
const dataTable = document.getElementById('dataTable');

fetch("http://127.0.0.1:9000/api/v1/dns")
      .then(response => {
        if (!response.ok) {
            return response.json().then(response => {throw new Error(response.error)});
        }
        return response.json();
      })
      .then(data => {
        for (let index = 0; index < data.data.length; index++){
            //insert Row
            responseMessage.insertRow().innerHTML =
            "<td>" +data.data[index].DataExfiltrated.p+ "</td>"+
            "<td>" +data.data[index].Timestamp+ "</td>"+
            "<td>" +data.data[index].SourceIP+ "</td>"+
            "<td>" +data.data[index].DataExfiltrated.h+ "</td>"+
            "<td>" +data.data[index].DataExfiltrated.d+ "</td>"+
            "<td>" +data.data[index].DataExfiltrated.c+ "</td>";
          }
          $('#dataTable').DataTable();
      })
      .catch(error => {
        console.log(error);
      });