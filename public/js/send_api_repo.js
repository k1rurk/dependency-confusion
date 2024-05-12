const apiUrlRepo = 'http://127.0.0.1:9000/api/v1/github/repo'; //name

const contactFormRepos = document.getElementById('form-github-repos');
const responseMessage = document.getElementById('dataTable').getElementsByTagName('tbody')[0];
const errorText = document.getElementById('error-text');
const errorWindow = document.getElementById('error-window');


contactFormRepos.addEventListener('submit', function (event) {
  event.preventDefault();
  errorWindow.style.cssText = "display: none;";
  sendRequest(apiUrlRepo, 'name', contactFormRepos)
});