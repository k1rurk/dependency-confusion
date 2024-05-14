package nodejs

import (
	"dependency-confusion/internal/models"
	"dependency-confusion/internal/parser/interfaces"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// PackageJSON represents the dependencies of an npm package
type PackageJSON struct {
	Dependencies         map[string]string `json:"dependencies,omitempty"`
	DevDependencies      map[string]string `json:"devDependencies,omitempty"`
	PeerDependencies     map[string]string `json:"peerDependencies,omitempty"`
	BundledDependencies  []string          `json:"bundledDependencies,omitempty"`
	BundleDependencies   []string          `json:"bundleDependencies,omitempty"`
	OptionalDependencies map[string]string `json:"optionalDependencies,omitempty"`
}

type PackageLock struct {
	Packages map[string]Dependencies `json:"packages"`
}

type PackageLockDirect struct {
	Dependencies map[string]map[string]string `json:"dependencies"`
}

type Dependencies struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

type NpmResponse struct {
	ID   string `json:"_id"`
	Name string `json:"name"`
	Time struct {
		Unpublished NpmResponseUnpublished `json:"unpublished"`
	} `json:"time"`
}

type NpmResponseUnpublished struct {
	Maintainers []struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"maintainers"`
	Name string `json:"name"`
	Tags struct {
		Latest string `json:"latest"`
	} `json:"tags"`
	Time     time.Time `json:"time"`
	Versions []string  `json:"versions"`
}

var blacklistDepName = []string{
	"node_modules",
	"favicon.ico",
}

// NotAvailable returns true if the package has its all versions unpublished making it susceptible for takeover
func (n *NpmResponse) NotAvailable() bool {
	// Check if a known field has a value
	return len(n.Time.Unpublished.Name) > 0
}

// NPMLookup represents a collection of npm packages to be tested for dependency confusion.
type NPMLookup struct {
	Packages    []NPMPackage
	Verbose     bool
	PackageFile string
	Extension   string
}

type NPMPackage struct {
	Name    string
	Version string
}

// NewNPMLookup constructs an `NPMLookup` struct and returns it.
func NewNPMLookup(verbose bool, packageFile, extension string) interfaces.PackageResolver {
	return &NPMLookup{Packages: []NPMPackage{}, PackageFile: packageFile, Extension: extension, Verbose: verbose}
}

// ReadPackagesFromFile reads package information from an npm package.json file
//
// Returns any errors encountered
func (n *NPMLookup) ReadPackagesFromFile(rawfile []byte) error {
	if n.PackageFile == "package.json" {
		return n.ReadPackagesFromPackageFile(rawfile)
	} else if n.PackageFile == "package-lock.json" {
		return n.ReadPackagesFromLockFile(rawfile)
	} else if n.PackageFile == "yarn.lock" {
		return n.ReadPackagesFromYarnLockFile(rawfile)
	} else {
		if n.Extension == ".css" {
			return n.ReadPackagesFromNPMMoudles(rawfile)
		} else {
			if err := n.ReadPackagesFromJSFile(rawfile); err == nil {
				return n.ReadPackagesFromNPMMoudles(rawfile)
			}
		}
	}
	return nil
}

func (n *NPMLookup) ReadPackagesFromPackageFile(rawfile []byte) error {
	data := PackageJSON{}
	err := json.Unmarshal([]byte(rawfile), &data)
	if err != nil {
		log.Warnf("[W] Non-fatal issue encountered while parsing npm file: %s\n", err)
	}
	for pkgname, pkgversion := range data.Dependencies {
		n.Packages = append(n.Packages, NPMPackage{pkgname, pkgversion})
	}
	for pkgname, pkgversion := range data.DevDependencies {
		n.Packages = append(n.Packages, NPMPackage{pkgname, pkgversion})
	}
	for pkgname, pkgversion := range data.PeerDependencies {
		n.Packages = append(n.Packages, NPMPackage{pkgname, pkgversion})
	}
	for pkgname, pkgversion := range data.OptionalDependencies {
		n.Packages = append(n.Packages, NPMPackage{pkgname, pkgversion})
	}
	for _, pkgname := range data.BundledDependencies {
		n.Packages = append(n.Packages, NPMPackage{pkgname, ""})
	}
	for _, pkgname := range data.BundleDependencies {
		n.Packages = append(n.Packages, NPMPackage{pkgname, ""})
	}
	return nil
}

// ReadPackagesFromLockFile reads package information from an npm package-lock.json file
//
// Returns any errors encountered
func (n *NPMLookup) ReadPackagesFromLockFile(rawfile []byte) error {
	var dependencies PackageLock
	json.Unmarshal(rawfile, &dependencies)
	if len(dependencies.Packages[""].Dependencies) == 0 && len(dependencies.Packages[""].DevDependencies) == 0 {
		var dependenciesDirect PackageLockDirect
		json.Unmarshal(rawfile, &dependenciesDirect)
		for key, val := range dependenciesDirect.Dependencies {
			n.Packages = append(n.Packages, NPMPackage{key, val["version"]})
		}
	} else {
		if len(dependencies.Packages[""].Dependencies) != 0 {
			for key, val := range dependencies.Packages[""].Dependencies {
				n.Packages = append(n.Packages, NPMPackage{key, val})
			}
		}
		if len(dependencies.Packages[""].DevDependencies) != 0 {
			for key, val := range dependencies.Packages[""].DevDependencies {
				n.Packages = append(n.Packages, NPMPackage{key, val})
			}
		}
	}
	return nil
}

// ReadPackagesFromYarnLockFile reads package information from an npm yarn.lock file
//
// Returns any errors encountered
func (n *NPMLookup) ReadPackagesFromYarnLockFile(rawfile []byte) error {

	dependenciesPattern := regexp.MustCompile(`[\w@-_]*[/]*[\w-_.]+@[\d^~><=.]+[\s]*[-\w.\s<=]*`)
	dependencies := dependenciesPattern.FindAllString(string(rawfile), -1)
	var depNameUnique map[string]string = map[string]string{}
	for _, dep := range dependencies {
		lastInd := strings.LastIndex(dep, "@")
		depName := dep[:lastInd]
		depVersion := dep[lastInd+1:]
		if _, exist := depNameUnique[depName]; !exist {
			depNameUnique[depName] = depVersion
			n.Packages = append(n.Packages, NPMPackage{depName, depVersion})
		}
	}

	return nil
}

func isNameValid(name string) bool {
	if name == "" {
		return false
	}
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || strings.ToLower(name) != name || strings.ContainsAny(name, "~\\'!()*\")") || strings.TrimSpace(name) != name || len(name) == 0 || len(name) > 214 {
		return false
	}
	for _, blacklist := range blacklistDepName {
		if name == blacklist {
			return false
		}
	}

	return true
}

// ReadPackagesFromJSFile reads package information from a js file
//
// Returns any errors encountered
func (n *NPMLookup) ReadPackagesFromJSFile(rawfile []byte) error {
	spacePattern := regexp.MustCompile(`\s`)
	tabPattern := regexp.MustCompile(`\t`)
	crPattern := regexp.MustCompile(`\r`)
	lfPattern := regexp.MustCompile(`\n`)

	fileLines := spacePattern.ReplaceAllString(string(rawfile), "")
	fileLines = tabPattern.ReplaceAllString(fileLines, "")
	fileLines = crPattern.ReplaceAllString(fileLines, "")
	fileLines = lfPattern.ReplaceAllString(fileLines, "")

	dependenciesPattern := regexp.MustCompile(`(?im)dependencies([a-z-_0-9])*['"]:\{(?P<dep>.*?)}`)
	dependencies := dependenciesPattern.FindAllStringSubmatch(fileLines, -1)
	var depNameUnique map[string]string = map[string]string{}
	for _, dep := range dependencies {
		dependencyList := dep[dependenciesPattern.SubexpIndex("dep")]
		dependencyListArray := strings.Split(dependencyList, ",")
		for _, dep := range dependencyListArray {
			nameAndVersionMatcher := regexp.MustCompile(`['"](.*)['"]:['"](.*)['"]`)
			if depParsed := nameAndVersionMatcher.FindStringSubmatch(dep); len(depParsed) == 3 {
				name := depParsed[1]
				version := depParsed[2]
				if isNameValid(name) {
					if _, exists := depNameUnique[name]; !exists {
						depNameUnique[name] = version
						n.Packages = append(n.Packages, NPMPackage{name, version})
					}
				}
			}
		}
	}

	return nil
}

// ReadPackagesFromNPMMoudles reads package information from js and css files
//
// Returns any errors encountered
func (n *NPMLookup) ReadPackagesFromNPMMoudles(rawfile []byte) error {
	spacePattern := regexp.MustCompile(`\s`)
	tabPattern := regexp.MustCompile(`\t`)
	crPattern := regexp.MustCompile(`\r`)
	lfPattern := regexp.MustCompile(`\n`)

	fileLines := spacePattern.ReplaceAllString(string(rawfile), "")
	fileLines = tabPattern.ReplaceAllString(fileLines, "")
	fileLines = crPattern.ReplaceAllString(fileLines, "")
	fileLines = lfPattern.ReplaceAllString(fileLines, "")

	extractFromNodeModules := regexp.MustCompile(`(?im)/node_modules/(?P<dep>@?[a-z-_.0-9]+)/`)
	fromNodeModulesPathMatcher := extractFromNodeModules.FindAllStringSubmatch(fileLines, -1)
	var depNameUnique map[string]string = map[string]string{}
	for _, dep := range fromNodeModulesPathMatcher {
		dependency := dep[extractFromNodeModules.SubexpIndex("dep")]
		if isNameValid(dependency) {
			if _, exists := depNameUnique[dependency]; !exists {
				depNameUnique[dependency] = ""
				n.Packages = append(n.Packages, NPMPackage{dependency, ""})
			}
		}
	}

	return nil
}

// PackagesNotInPublic determines if an npm package does not exist in the public npm package repository.
//
// Returns a map of strings with any npm packages not in the public npm package repository
func (n *NPMLookup) PackagesNotInPublic() []models.PackageManager {
	var notAvail []models.PackageManager
	for _, pkg := range n.Packages {
		if n.localReference(pkg.Version) || n.urlReference(pkg.Version) || n.gitReference(pkg.Version) {
			continue
		}
		if n.gitHubReference(pkg.Version) {
			if !n.gitHubOrgExists(pkg.Version) {
				notAvail = append(notAvail, models.PackageManager{Name: "npm", Package: pkg.Name, Version: pkg.Version})
				continue
			} else {
				continue
			}
		}
		if !n.isAvailableInPublic(pkg.Name, 0) {
			notAvail = append(notAvail, models.PackageManager{Name: "npm", Package: pkg.Name, Version: pkg.Version})
		}
	}
	return notAvail
}

// isAvailableInPublic determines if an npm package exists in the public npm package repository.
//
// Returns true if the package exists in the public npm package repository.
func (n *NPMLookup) isAvailableInPublic(pkgname string, retry int) bool {
	if retry > 3 {
		log.Warnf(" [W] Maximum number of retries exhausted for package: %s\n", pkgname)
		return false
	}
	if n.Verbose {
		log.Infoln("Checking: https://registry.npmjs.org/" + pkgname + "/ : ")
	}
	resp, err := http.Get("https://registry.npmjs.org/" + pkgname + "/")
	if err != nil {
		log.Warnf(" [W] Error when trying to request https://registry.npmjs.org/"+pkgname+"/ : %s\n", err)
		return false
	}
	defer resp.Body.Close()
	if n.Verbose {
		log.Infof("%s\n", resp.Status)
	}
	if resp.StatusCode == http.StatusOK {
		npmResp := NpmResponse{}
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &npmResp)
		if npmResp.NotAvailable() {
			if n.Verbose {
				log.Warnf("[W] Package %s was found, but all its versions are unpublished, making anyone able to takeover the namespace.\n", pkgname)
			}
			return false
		}
		return true
	} else if resp.StatusCode == 429 {
		log.Warnf(" [!] Server responded with 429 (Too many requests), throttling and retrying...\n")
		time.Sleep(10 * time.Second)
		retry = retry + 1
		return n.isAvailableInPublic(pkgname, retry)
	}
	return false
}

// localReference checks if the package version is in fact a reference to filesystem
func (n *NPMLookup) localReference(pkgversion string) bool {
	return strings.HasPrefix(strings.ToLower(pkgversion), "file:")
}

// urlReference checks if the package version is in fact a reference to a direct URL
func (n *NPMLookup) urlReference(pkgversion string) bool {
	pkgversion = strings.ToLower(pkgversion)
	return strings.HasPrefix(pkgversion, "http:") || strings.HasPrefix(pkgversion, "https:")
}

// gitReference checks if the package version is in fact a reference to a remote git repository
func (n *NPMLookup) gitReference(pkgversion string) bool {
	pkgversion = strings.ToLower(pkgversion)
	gitResources := []string{"git+ssh:", "git+http:", "git+https:", "git:", "github:"}
	for _, r := range gitResources {
		if strings.HasPrefix(pkgversion, r) {
			return true
		}
	}
	return false
}

// gitHubReference checks if the package version refers to a GitHub repository
func (n *NPMLookup) gitHubReference(pkgversion string) bool {
	return !strings.HasPrefix(pkgversion, "@") && strings.Contains(pkgversion, "/")
}

// gitHubOrgExists returns true if GitHub organization / user exists
func (n NPMLookup) gitHubOrgExists(pkgversion string) bool {
	orgName := strings.Split(pkgversion, "/")[0]
	if len(orgName) > 0 {
		org := strings.Split(orgName, ":")
		if len(org) > 2 {
			orgName = org[1]
		}
		if n.Verbose {
			log.Infoln("Checking: https://github.com/" + orgName + " : ")
		}
		resp, err := http.Get("https://github.com/" + orgName)
		if err != nil {
			log.Warnf(" [W] Error while trying to request https://github.com/"+orgName+" : %s\n", err)
			return false
		}
		defer resp.Body.Close()
		if n.Verbose {
			log.Infof("%d\n", resp.StatusCode)
		}
		return resp.StatusCode == 200
	}
	return false
}
