const apiUrlReport = 'http://127.0.0.1:9000/api/v1/report';
const generateReport = document.getElementById('generate-report');

generateReport.addEventListener('click', function (e) {
    e.preventDefault()
    fetch(apiUrlReport)
      .then(response => {
        if (response.status >= 400) {
            return response.json().then(response => {throw new Error(response.error)});
        } else {
          window.open(response.url, '_blank');
        }
      }).catch(error => {
        console.log(error);
      });
});