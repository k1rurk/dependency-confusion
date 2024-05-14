package python

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"

	"dependency-confusion/internal/models"
	"dependency-confusion/internal/parser/interfaces"
	"dependency-confusion/internal/parser/utils"

	"github.com/bigkevmcd/go-configparser"
	"github.com/pelletier/go-toml"
)

type PythonPackage struct {
	Name    string
	Version string
}

// PythonLookup represents a collection of python packages to be tested for dependency confusion.
type PythonLookup struct {
	Packages    []PythonPackage
	Verbose     bool
	PackageFile string
}

// NewPythonLookup constructs a `PythonLookup` struct and returns it
func NewPythonLookup(verbose bool, packageFile string) interfaces.PackageResolver {
	return &PythonLookup{Packages: []PythonPackage{}, Verbose: verbose, PackageFile: packageFile}
}

// ReadPackagesFromFile chooses a file parser based on the user-supplied python package manager.
//
// Returns any errors encountered
func (p *PythonLookup) ReadPackagesFromFile(rawfile []byte) error {
	switch p.PackageFile {
	case "requirements.txt", "requirements.in":
		return p.ReadPackagesFromRequirementsTxt(rawfile)
	case "Pipfile":
		return p.ReadPackagesFromPipfile(rawfile)
	case "pdm.lock":
		return p.ReadPackagesFromPdmlock(rawfile)
	case "pyproject.toml":
		return p.ReadPackagesFromPyproject(rawfile)
	case "setup.cfg":
		return p.ReadPackagesFromSetupcfg(rawfile)
	default:
		return fmt.Errorf("python package manager not implemented: %s", p.PackageFile)
	}
}

// ReadPackagesFromRequirementsTxt reads package information from a python `requirements.txt`.
//
// Returns any errors encountered
func (p *PythonLookup) ReadPackagesFromRequirementsTxt(rawfile []byte) error {
	line := ""
	for _, l := range strings.Split(string(rawfile), "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "#") {
			continue
		}
		if len(l) > 0 {
			// Support line continuation
			if strings.HasSuffix(l, "\\") {
				line += l[:len(l)-1]
				continue
			}
			line += l
			if strings.Contains(line, "#") {
				line = strings.Split(line, "#")[0]
			}
			//(([>=<~])+\s*\d+\.?\d*\.?\d*)
			//(([>=<~])+\s*\d+[\w.-]*)
			versionPattern := regexp.MustCompile(`(([>=<~])+\s*\d+[\w.-]*)`)
			version := versionPattern.FindAllString(line, -1)

			pkgrow := strings.FieldsFunc(line, p.pipSplit)

			if len(pkgrow) > 0 {
				p.Packages = append(p.Packages, PythonPackage{strings.TrimSpace(pkgrow[0]), strings.Join(version, " ")})
			}
			// reset the line variable
			line = ""
		}
	}
	return nil
}

// ReadPackagesFromPipfile reads package information from a python `Pipfile`.
//
// Returns any errors encountered
func (p *PythonLookup) ReadPackagesFromPipfile(rawfile []byte) error {
	config, err := toml.Load(string(rawfile))
	if err != nil {
		return err
	}
	packages := config.Get("packages")
	if packages != nil {
		for key, val := range packages.(*toml.Tree).ToMap() {
			switch v := val.(type) {
			case string:
				p.Packages = append(p.Packages, PythonPackage{key, v})
			default:
				tempVal := v.(map[string]interface{})
				if interf, ok := tempVal["version"]; ok {
					p.Packages = append(p.Packages, PythonPackage{key, interf.(string)})
				} else {
					p.Packages = append(p.Packages, PythonPackage{key, ""})
				}
			}
		}
	}
	devPackages := config.Get("dev-packages")
	if devPackages != nil {
		for key, val := range devPackages.(*toml.Tree).ToMap() {
			switch v := val.(type) {
			case string:
				p.Packages = append(p.Packages, PythonPackage{key, v})
			default:
				tempVal := v.(map[string]interface{})
				if interf, ok := tempVal["version"]; ok {
					p.Packages = append(p.Packages, PythonPackage{key, interf.(string)})
				} else {
					p.Packages = append(p.Packages, PythonPackage{key, ""})
				}
			}
		}
	}
	return nil
}

// ReadPackagesFromPdmlock reads package information from a python `pdm.lock`.
//
// Returns any errors encountered
func (p *PythonLookup) ReadPackagesFromPdmlock(rawfile []byte) error {
	config, err := toml.Load(string(rawfile))
	if err != nil {
		return err
	}
	configTree := config.Get("package").([]*toml.Tree)
	if configTree != nil {
		for _, k := range configTree {
			p.Packages = append(p.Packages, PythonPackage{k.Get("name").(string), k.Get("version").(string)})
		}
	}
	return nil
}

// ReadPackagesFromPyproject reads package information from a python `pyroject.toml`.
//
// Returns any errors encountered
func (p *PythonLookup) ReadPackagesFromPyproject(rawfile []byte) error {
	config, err := toml.Load(string(rawfile))
	if err != nil {
		return err
	}
	var aStringProject []string
	configTreeProject := config.Get("project")
	if configTreeProject != nil {
		configTreeProjectTree := configTreeProject.(*toml.Tree)
		aInterfaceProject := configTreeProjectTree.Get("dependencies").([]interface{})
		aStringProject = make([]string, len(aInterfaceProject))
		for i, v := range aInterfaceProject {
			aStringProject[i] = v.(string)
		}
	}

	configTreeOptional := config.Get("project.optional-dependencies")
	if configTreeOptional != nil {
		for _, val := range configTreeOptional.(*toml.Tree).ToMap() {
			aVal := val.([]interface{})
			for _, v := range aVal {
				aStringProject = append(aStringProject, v.(string))
			}
		}
	}
	if len(aStringProject) > 0 {
		for _, valVersion := range aStringProject {
			versionPattern := regexp.MustCompile(`(([>=<~])+\s*\d+[\w.-]*)`)
			version := versionPattern.FindAllString(valVersion, -1)

			pkgrow := strings.FieldsFunc(valVersion, p.pipSplit)

			p.Packages = append(p.Packages, PythonPackage{strings.TrimSpace(pkgrow[0]), strings.Join(version, " ")})

		}
	}
	configTreePoetry := config.Get("tool.poetry.dependencies")
	if configTreePoetry != nil {
		for key, val := range configTreePoetry.(*toml.Tree).ToMap() {
			switch v := val.(type) {
			case string:
				p.Packages = append(p.Packages, PythonPackage{key, v})
			case map[string]interface{}:
				val := v["version"].(string)
				p.Packages = append(p.Packages, PythonPackage{key, val})
			default:
				arrayInterfaces := v.([]interface{})
				for _, interfaceValue := range arrayInterfaces {
					convertedValue := interfaceValue.(map[string]interface{})
					if interf, ok := convertedValue["version"]; ok {
						p.Packages = append(p.Packages, PythonPackage{key, interf.(string)})
					} else {
						p.Packages = append(p.Packages, PythonPackage{key, ""})
					}
				}
				
			}
		}
	}
	configTreePoetryDev := config.Get("tool.poetry.dev-dependencies")
	if configTreePoetryDev != nil {
		for key, val := range configTreePoetryDev.(*toml.Tree).ToMap() {
			switch v := val.(type) {
			case string:
				p.Packages = append(p.Packages, PythonPackage{key, v})
			case map[string]interface{}:
				val := v["version"].(string)
				p.Packages = append(p.Packages, PythonPackage{key, val})
			default:
				arrayInterfaces := v.([]interface{})
				for _, interfaceValue := range arrayInterfaces {
					convertedValue := interfaceValue.(map[string]interface{})
					if interf, ok := convertedValue["version"]; ok {
						p.Packages = append(p.Packages, PythonPackage{key, interf.(string)})
					} else {
						p.Packages = append(p.Packages, PythonPackage{key, ""})
					}
				}
				
			}
		}
	}
	configTreePoetryGroup := config.Get("tool.poetry.group.dependencies")
	if configTreePoetryGroup != nil {
		for key, val := range configTreePoetryGroup.(*toml.Tree).ToMap() {
			switch v := val.(type) {
			case string:
				p.Packages = append(p.Packages, PythonPackage{key, v})
			case map[string]interface{}:
				val := v["version"].(string)
				p.Packages = append(p.Packages, PythonPackage{key, val})
			default:
				arrayInterfaces := v.([]interface{})
				for _, interfaceValue := range arrayInterfaces {
					convertedValue := interfaceValue.(map[string]interface{})
					if interf, ok := convertedValue["version"]; ok {
						p.Packages = append(p.Packages, PythonPackage{key, interf.(string)})
					} else {
						p.Packages = append(p.Packages, PythonPackage{key, ""})
					}
				}
				
			}
		}
	}
	configTreePoetryDevGroup := config.Get("tool.poetry.group.dev.dependencies")
	if configTreePoetryDevGroup != nil {
		for key, val := range configTreePoetryDevGroup.(*toml.Tree).ToMap() {
			switch v := val.(type) {
			case string:
				p.Packages = append(p.Packages, PythonPackage{key, v})
			case map[string]interface{}:
				val := v["version"].(string)
				p.Packages = append(p.Packages, PythonPackage{key, val})
			default:
				arrayInterfaces := v.([]interface{})
				for _, interfaceValue := range arrayInterfaces {
					convertedValue := interfaceValue.(map[string]interface{})
					if interf, ok := convertedValue["version"]; ok {
						p.Packages = append(p.Packages, PythonPackage{key, interf.(string)})
					} else {
						p.Packages = append(p.Packages, PythonPackage{key, ""})
					}
				}
				
			}
		}
	}
	return nil
}

// ReadPackagesFromSetupcfg reads package information from a python `setup.cfg`.
//
// Returns any errors encountered
func (p *PythonLookup) ReadPackagesFromSetupcfg(rawfile []byte) error {
	r := bytes.NewReader(rawfile)
	var dependencies []string
	inidata, err := configparser.ParseReader(r)
	if err != nil {
		log.Errorf("Fail to read file: %v", err)
		return err
	}

	options, err := inidata.Get("options", "install_requires")
	if err != nil {
		log.Errorf("Fail to read options->install_requires: %v", err)
	}
	
	if len(options) != 0 {
		dependencies = strings.Split(options, "\n")
	} 

	dict, err := inidata.Items("options.extras_require")
	if err != nil {
		log.Errorf("Fail to read options.extras_require: %v", err)
	}

	for _, val := range dict.Keys() {
		dependencies = append(dependencies, dict[val])
	}

	for _, val := range dependencies {
		versionPattern := regexp.MustCompile(`(([>=<~])+\s*\d+[\w.-]*)`)
		version := versionPattern.FindAllString(val, -1)

		pkgrow := strings.FieldsFunc(val, p.pipSplit)

		p.Packages = append(p.Packages, PythonPackage{strings.TrimSpace(pkgrow[0]), strings.Join(version, " ")})
	}

	return nil
}

// ReadPackagesFromSetuppy reads package information from a python `setup.py`.
//
// Returns any errors encountered
// func (p *PythonLookup) ReadPackagesFromSetuppy(rawfile []byte) error {
// 	// [,\s]+(['"]*)install_requires(['"]*)\s*(=|:)+\s*((\[([^\]]+)\])|['"][\w\d-.]+['"])
// 	return nil
// }

// PackagesNotInPublic determines if a python package does not exist in the pypi package repository.
//
// Returns a map of strings with any python packages not in the pypi package repository
func (p *PythonLookup) PackagesNotInPublic() []models.PackageManager {
	var notAvail []models.PackageManager
	for _, pkg := range p.Packages {
		if pkg.Name == "python" || strings.TrimSpace(pkg.Name) == "." || strings.Contains(pkg.Name, "//github.com") {
			continue
		}
		if !p.isAvailableInPublic(pkg.Name) {
			notAvail = append(notAvail, models.PackageManager{Name: "pip", Package: pkg.Name, Version: pkg.Version})
		}
	}
	return notAvail
}

func (p *PythonLookup) pipSplit(r rune) bool {
	delims := []rune{
		'=',
		'<',
		'>',
		'!',
		' ',
		'~',
		'#',
		'[',
		';',
	}
	return utils.InSlice(r, delims)
}

// isAvailableInPublic determines if a python package exists in the pypi package repository.
//
// Returns true if the package exists in the pypi package repository.
func (p *PythonLookup) isAvailableInPublic(pkgname string) bool {
	if p.Verbose {
		log.Infoln("Checking: https://pypi.org/project/" + pkgname + "/ : ")
	}
	resp, err := http.Get("https://pypi.org/project/" + pkgname + "/")
	if err != nil {
		log.Warnf(" [W] Error when trying to request https://pypi.org/project/"+pkgname+"/ : %s\n", err)
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
