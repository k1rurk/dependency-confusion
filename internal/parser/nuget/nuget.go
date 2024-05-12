package nuget

import (
	"dependency-confusion/internal/models"
	"dependency-confusion/internal/parser/interfaces"
	"encoding/xml"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

// NugetLookup represents a collection of .Net packages to be tested for dependency confusion.
type NugetLookup struct {
	Packages    []NugetPackage
	Verbose     bool
	PackageFile string
}

type NugetPackage struct {
	PackageId string
	Version   string
}

// NewNugetLookup constructs a `NugetLookup` struct and returns it
func NewNugetLookup(verbose bool, packageFile string) interfaces.PackageResolver {
	return &NugetLookup{Packages: []NugetPackage{}, Verbose: verbose, PackageFile: packageFile}
}

// ReadPackagesFromFile chooses a file parser based on the user-supplied .Net package manager.
//
// Returns any errors encountered
func (p *NugetLookup) ReadPackagesFromFile(rawfile []byte) error {
	if p.PackageFile == "packages.config" {
		return p.ReadPackagesFromPackageConfig(rawfile)
	} else {
		return p.ReadPackagesFromProjectConfig(rawfile)
	}
}

// PackagesNotInPublic determines if a .Net package does not exist in the nuget package repository.
//
// Returns a map of strings with any .Net packages not in the nuget package repository
func (p *NugetLookup) PackagesNotInPublic() []models.PackageManager {
	var notAvail []models.PackageManager
	for _, pkg := range p.Packages {
		if !p.isAvailableInPublic(strings.ToLower(pkg.PackageId)) {
			notAvail = append(notAvail, models.PackageManager{Name: "nuget", Package: pkg.PackageId, Version: pkg.Version})
		}
	}
	return notAvail
}

func (p *NugetLookup) ReadPackagesFromPackageConfig(rawfile []byte) error {
	var project NugetProjectPackages
	if err := xml.Unmarshal([]byte(rawfile), &project); err != nil {
		log.Fatalf("unable to unmarshal packages.config file. Reason: %s\n", err)
	}
	for _, dep := range project.Package {
		if dep.Id != "" {
			p.Packages = append(p.Packages, NugetPackage{dep.Id, dep.Version})
		}
	}
	return nil
}

func (p *NugetLookup) ReadPackagesFromProjectConfig(rawfile []byte) error {
	var project NugetProjectCsproj
	if err := xml.Unmarshal([]byte(rawfile), &project); err != nil {
		log.Fatalf("unable to unmarshal packages.config file. Reason: %s\n", err)
	}
	for _, item := range project.ItemGroup {
		for _, dep := range item.PackageReference {
			if dep.Include != "" && !strings.Contains(dep.Include, "..") {
				if dep.VersionTag.Value != "" {
					p.Packages = append(p.Packages, NugetPackage{dep.Include, dep.VersionTag.Value})
				} else {
					p.Packages = append(p.Packages, NugetPackage{dep.Include, dep.Version})
				}
			}
		}
	}
	return nil
}

// isAvailableInPublic determines if an nuget package exists in the public nuget package repository.
//
// Returns true if the package exists in the public nuget package repository.
func (p *NugetLookup) isAvailableInPublic(pkgname string) bool {
	if p.Verbose {
		log.Infoln("Checking: https://api.nuget.org/v3/registration5-semver1/" + pkgname + "/index.json: ")
	}
	resp, err := http.Get("https://api.nuget.org/v3/registration5-semver1/" + pkgname + "/index.json")
	if err != nil {
		log.Warnf(" [W] Error when trying to request https://api.nuget.org/v3/registration5-semver1/"+pkgname+"/index.json: %s\n", err)
		return false
	}
	if p.Verbose {
		log.Infof("%s\n", resp.Status)
	}
	if resp.StatusCode == http.StatusOK {
		return true
	}
	return false

}
