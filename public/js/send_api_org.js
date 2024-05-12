const apiUrlOrg = 'http://127.0.0.1:9000/api/v1/github/org'; //org

const contactFormOrg = document.getElementById('form-github-org');
const responseMessage = document.getElementById('dataTable').getElementsByTagName('tbody')[0];
const errorText = document.getElementById('error-text');
const errorWindow = document.getElementById('error-window');

contactFormOrg.addEventListener('submit', function (event) {
  event.preventDefault();
  errorWindow.style.cssText = "display: none;";
  sendRequest(apiUrlOrg, 'org', contactFormOrg)
});
