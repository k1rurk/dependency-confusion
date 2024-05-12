package javamvn

import (
	"dependency-confusion/internal/models"
	"dependency-confusion/internal/parser/interfaces"
	"dependency-confusion/internal/parser/nodejs"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// MVNLookup represents a collection of mvn packages to be tested for dependency confusion.
type MVNLookup struct {
	Packages    []MVNPackage
	Verbose     bool
	PackageFile string
}

type MVNPackage struct {
	Group    string
	Artifact string
	Version  string
}

// NewMVNLookup constructs an `MVNLookup` struct and returns it.
func NewMVNLookup(verbose bool, packageFile string) interfaces.PackageResolver {
	return &MVNLookup{Packages: []MVNPackage{}, Verbose: verbose, PackageFile: packageFile}
}

// ReadPackagesFromFile reads package information from an mvn pom.xml file
//
// Returns any errors encountered
func (n *MVNLookup) ReadPackagesFromFile(rawfile []byte) error {
	if n.PackageFile == "pom.xml" {
		return n.ReadPackagesFromPomFile(rawfile)
	} else {
		return n.ReadPackagesFromGradleFile(rawfile)
	}
}

func (n *MVNLookup) ReadPackagesFromGradleFile(rawfile []byte) error {

	//fileContentString := string(rawfile)
	// Regular expression to match dependencies
	pattern := regexp.MustCompile(`dependencies\s*{\s*([\s\S]*?)\s*}`)

	// Find the dependencies block
	match := pattern.Find(rawfile)

	if len(match) > 0 {
		// Regular expression to match dependencies
		dependencyPattern := regexp.MustCompile(`[\'\"]([\w\-.]+:[\w\-.]+:[\w\-.\$]+)`)
		dependencies := dependencyPattern.FindAll(match, -1)
		replacer := strings.NewReplacer("\\", "", "'", "", `"`, "")
		for _, dep := range dependencies {
			replacedStr := replacer.Replace(string(dep))
			depSplit := strings.Split(replacedStr, ":")
			version := depSplit[2]
			if strings.Contains(version, "$") {
				version = ""
			}
			n.Packages = append(n.Packages, MVNPackage{depSplit[0], depSplit[1], version})
		}
		dependencyPatternNoVersion := regexp.MustCompile(`[\'\"]([\w\-.]+:[\w\-.]+:[\w\-.\$]+)`)
		dependenciesNoVersion := dependencyPatternNoVersion.FindAll(match, -1)
		for _, dep := range dependenciesNoVersion {
			replacedStr := replacer.Replace(string(dep))
			depSplit := strings.Split(replacedStr, ":")
			mvnPackage := MVNPackage{depSplit[0], depSplit[1], ""}
			if !slices.Contains(n.Packages, mvnPackage) {
				n.Packages = append(n.Packages, mvnPackage)
			}
		}

	}

	return nil
}

// ReadPackagesFromFile reads package information from an mvn pom.xml file
//
// Returns any errors encountered
func (n *MVNLookup) ReadPackagesFromPomFile(rawfile []byte) error {

	var project MavenProject
	if err := xml.Unmarshal([]byte(rawfile), &project); err != nil {
		log.Fatalf("unable to unmarshal pom file. Reason: %s\n", err)
	}

	for _, dep := range project.Dependencies {
		n.Packages = append(n.Packages, MVNPackage{dep.GroupId, dep.ArtifactId, dep.Version})
	}

	for _, dep := range project.Build.Plugins {
		n.Packages = append(n.Packages, MVNPackage{dep.GroupId, dep.ArtifactId, dep.Version})
	}

	for _, build := range project.Profiles {
		for _, dep := range build.Build.Plugins {
			n.Packages = append(n.Packages, MVNPackage{dep.GroupId, dep.ArtifactId, dep.Version})
		}
	}

	return nil
}

// PackagesNotInPublic determines if an mvn package does not exist in the public mvn package repository.
//
// Returns a map of strings with any mvn packages not in the public mvn package repository
func (n *MVNLookup) PackagesNotInPublic() []models.PackageManager {
	var notAvail []models.PackageManager
	for _, pkg := range n.Packages {
		if !n.isAvailableInPublic(pkg, 0) {
			notAvail = append(notAvail, models.PackageManager{Name: "mvn", Package: pkg.Group + "/" + pkg.Artifact, Version: pkg.Version})
		}
	}
	return notAvail
}

// isAvailableInPublic determines if an mvn package exists in the public mvn package repository.
//
// Returns true if the package exists in the public mvn package repository.
func (n *MVNLookup) isAvailableInPublic(pkg MVNPackage, retry int) bool {
	if retry > 3 {
		log.Warnf(" [W] Maximum number of retries exhausted for package: %s\n", pkg.Group)
		return false
	}
	if pkg.Group == "" {
		return true
	}

	group := strings.Replace(pkg.Group, ".", "/", -1)
	if n.Verbose {
		log.Infoln("Checking: https://repo1.maven.org/maven2/" + group + "/ ")
	}
	resp, err := http.Get("https://repo1.maven.org/maven2/" + group + "/")
	if err != nil {
		log.Warnf(" [W] Error when trying to request https://repo1.maven.org/maven2/"+group+"/ : %s\n", err)
		return false
	}
	defer resp.Body.Close()
	if n.Verbose {
		log.Infof("%s\n", resp.Status)
	}
	if resp.StatusCode == http.StatusOK {
		npmResp := nodejs.NpmResponse{}
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &npmResp)
		if npmResp.NotAvailable() {
			if n.Verbose {
				log.Warnf("[W] Package %s was found, but all its versions are unpublished, making anyone able to takeover the namespace.\n", pkg.Group)
			}
			return false
		}
		return true
	} else if resp.StatusCode == 429 {
		log.Warnf(" [!] Server responded with 429 (Too many requests), throttling and retrying...\n")
		time.Sleep(10 * time.Second)
		retry = retry + 1
		return n.isAvailableInPublic(pkg, retry)
	}
	return false
}
