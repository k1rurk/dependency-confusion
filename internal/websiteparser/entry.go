package websiteparser


func Run(url string, cookie []string, temDir string, apiKey string) {
	agent := New(cookie, apiKey, temDir)
	agent.Execute(url)
}
