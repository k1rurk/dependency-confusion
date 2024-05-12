package file

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// CreateProject initializes the project directory and returns the path to the project
func CreateProject(projectName, dirPath string) string {

	// define project path
	projectPath := filepath.Join(dirPath, projectName)

	// create CSS/JS/Image directories
	createCSS(projectPath)
	createJS(projectPath)

	// project path
	return projectPath
}

// createCSS create a css directory in the current path
func createCSS(path string) {
	// create css directory
	err := os.MkdirAll(filepath.Join(path, "css"), 0777)
	check(err)
}

// createJS create a JS directory in the current path
func createJS(path string) {
	err := os.MkdirAll(filepath.Join(path, "js"), 0777)
	check(err)
}

func check(err error) {
	if err != nil {
		log.Errorln(err)
	}
}
