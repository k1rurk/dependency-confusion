package docker

import (
	"dependency-confusion/internal/models"
	"dependency-confusion/internal/parser/interfaces"
	"net/http"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

type DockerLookup struct {
	Packages []DockerPackage
	Verbose  bool
}

type DockerPackage struct {
	Namespace string
	Name      string
	Version   string
}

// NewDockerLookup constructs a `DockerLookup` struct and returns it
func NewDockerLookup(verbose bool) interfaces.PackageResolver {
	return &DockerLookup{Packages: []DockerPackage{}, Verbose: verbose}
}

// ReadPackagesFromFile chooses a file parser
//
// Returns any errors encountered
func (d *DockerLookup) ReadPackagesFromFile(rawfile []byte) error {
	return d.ReadPackagesFromDockerFile(rawfile)
}

func (d *DockerLookup) ReadPackagesFromDockerFile(rawfile []byte) error {

	dependenciesPattern := regexp.MustCompile(`FROM\s+(--platform=[${}\w]*)*\s*[\w.\-_]+\/*[\w.\-_]*:*[\d\w.\-_]*[\s:]+`)
	dependencies := dependenciesPattern.FindAllString(string(rawfile), -1)
	depNameUnique := map[string]string{}
	if len(dependencies) != 0 {
		for _, dep := range dependencies {
			trimmedDep := strings.TrimSpace(dep)
			depSplit := strings.Split(trimmedDep, "/")
			depSplitSpace := strings.Split(depSplit[0], " ")
			var packageName string
			var version string
			if len(depSplit) == 2 {
				org := strings.TrimSpace(depSplitSpace[len(depSplitSpace)-1])
				packageNameVersionSplit := strings.Split(depSplit[1], ":")
				packageName = packageNameVersionSplit[0]
				if len(packageNameVersionSplit) == 1 {
					version = "latest"
				} else if packageNameVersionSplit[1] == "" {
					version = "[unknown]"
				} else {
					version = packageNameVersionSplit[1]
				}
				if _, exists := depNameUnique[org+"/"+packageName]; !exists {
					depNameUnique[org+"/"+packageName] = version
					d.Packages = append(d.Packages, DockerPackage{org, packageName, version})
				}
			}
		}
	}

	return nil
}

// PackagesNotInPublic determines if a Dockerfile package does not exist in the Docker hub .
//
// Returns a map of strings with any Docker packages not in the Docker hub repository
func (d *DockerLookup) PackagesNotInPublic() []models.PackageManager {
	var notAvail []models.PackageManager
	for _, pkg := range d.Packages {
		if !d.isAvailableInPublic(strings.ToLower(pkg.Namespace), strings.ToLower(pkg.Name)) {
			notAvail = append(notAvail, models.PackageManager{Name: "docker", Package: pkg.Namespace + "/" + pkg.Name, Version: pkg.Version})
		}
	}
	return notAvail
}

// isAvailableInPublic determines if an docker package exists in the public docker package repository.
//
// Returns true if the package exists in the public docker package repository.
func (d *DockerLookup) isAvailableInPublic(pkgNamespace, pkgName string) bool {
	if d.Verbose {
		log.Infoln("Checking: https://hub.docker.com/v2/namespaces/" + pkgNamespace + "/repositories/" + pkgName)
	}
	resp, err := http.Get("https://hub.docker.com/v2/namespaces/" + pkgNamespace + "/repositories/" + pkgName)
	if err != nil {
		log.Warnf(" [W] Error when trying to request https://hub.docker.com/v2/namespaces/"+pkgNamespace+"/repositories/"+pkgName+": %s\n", err)
		return false
	}
	if d.Verbose {
		log.Infof("%s\n", resp.Status)
	}
	if resp.StatusCode == http.StatusOK {
		return true
	}
	return false

}
