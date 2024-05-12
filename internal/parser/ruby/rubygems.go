package ruby

import (
	"bufio"
	"dependency-confusion/internal/models"
	"dependency-confusion/internal/parser/interfaces"
	"dependency-confusion/internal/parser/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	log "github.com/sirupsen/logrus"
)

type Gem struct {
	Remote       string
	IsLocal      bool
	IsRubyGems   bool
	IsTransitive bool
	Name         string
	Version      string
}

type RubyGemsResponse struct {
	Name      string `json:"name"`
	Downloads int64  `json:"downloads"`
	Version   string `json:"version"`
}

// RubyGemsLookup represents a collection of rubygems packages to be tested for dependency confusion.
type RubyGemsLookup struct {
	Packages    []Gem
	Verbose     bool
	PackageFile string
}

// NewRubyGemsLookup constructs an `RubyGemsLookup` struct and returns it.
func NewRubyGemsLookup(verbose bool, packageFile string) interfaces.PackageResolver {
	return &RubyGemsLookup{Packages: []Gem{}, Verbose: verbose, PackageFile: packageFile}
}

// ReadPackagesFromFile reads package information from a Gemfile.lock and Gemfile files
//
// Returns any errors encountered
func (r *RubyGemsLookup) ReadPackagesFromFile(rawfile []byte) error {

	if r.PackageFile == "Gemfile" {
		return r.ReadPackagesFromGemfile(rawfile)
	} else {
		return r.ReadPackagesFromGemlock(rawfile)
	}
}

func (r *RubyGemsLookup) ReadPackagesFromGemlock(rawfile []byte) error {
	scanner := bufio.NewScanner(strings.NewReader(string(rawfile)))
	depNameUnique := map[string]string{}
	var remote string
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "remote:") {
			remote = strings.TrimSpace(strings.SplitN(trimmedLine, ":", 2)[1])
		} else if trimmedLine == "revision:" {
			continue
		} else if trimmedLine == "branch:" {
			continue
		} else if trimmedLine == "GIT" {
			continue
		} else if trimmedLine == "GEM" {
			continue
		} else if trimmedLine == "PATH" {
			continue
		} else if trimmedLine == "PLATFORMS" {
			break
		} else if trimmedLine == "specs:" {
			continue
		} else if len(trimmedLine) > 0 {
			parts := strings.SplitN(trimmedLine, " ", 2)
			name := strings.TrimSpace(parts[0])
			var version string
			if len(parts) > 1 {
				version = strings.TrimRight(strings.TrimLeft(parts[1], "("), ")")
			} else {
				version = ""
			}
			if _, exists := depNameUnique[name]; !exists {
				depNameUnique[name] = version
				r.Packages = append(r.Packages, Gem{
					Remote:  remote,
					IsLocal: !strings.HasPrefix(remote, "http"),
					// what is this logic? only caring if it's on ruby gems? what about other URLs? https://github.com/omnilaboratory/api.doc/blob/40f21ef6765ec5d7d9029446109141669a71a57d/Gemfile.lock
					IsRubyGems:   strings.HasPrefix(remote, "https://rubygems.org"),
					IsTransitive: utils.CountLeadingSpaces(line) == 6,
					Name:         name,
					Version:      version,
				})
			}
		} else {
			continue
		}
	}
	return nil
}

func (r *RubyGemsLookup) ReadPackagesFromGemfile(rawfile []byte) error {

	pattern := regexp.MustCompile(`(gem\s*\"|gem\s*\').+`)
	dependencies := pattern.FindAll(rawfile, -1)

	for _, dep := range dependencies {
		trimmedDep := strings.TrimSpace(string(dep))
		trimmedDep = strings.Trim(trimmedDep, "gem")
		depSplit := strings.Split(trimmedDep, ",")
		name := strings.TrimFunc(depSplit[0], func(r rune) bool {
			return unicode.IsPunct(r) || unicode.IsSpace(r)
		})
		versionPattern := regexp.MustCompile(`(["'])([<>~=]*\s*\d.*)(["'])`)
		versions := versionPattern.FindAllString(trimmedDep, -1)
		var version string
		if len(versions) == 0 {
			version = ""
		} else if len(versions) == 1 {
			version = strings.TrimFunc(versions[0], func(r rune) bool {
				return unicode.IsPunct(r) || unicode.IsSpace(r)
			})
		} else {
			subVersionStart := strings.TrimFunc(versions[0], func(r rune) bool {
				return unicode.IsPunct(r) || unicode.IsSpace(r)
			})
			subVersionEnd := strings.TrimFunc(versions[1], func(r rune) bool {
				return unicode.IsPunct(r) || unicode.IsSpace(r)
			})
			version = fmt.Sprintf("%s %s", subVersionStart, subVersionEnd)
		}

		r.Packages = append(r.Packages, Gem{
			Remote:       "",
			IsLocal:      false,
			IsRubyGems:   true,
			IsTransitive: false,
			Name:         name,
			Version:      version,
		})
	}
	return nil
}

// PackagesNotInPublic determines if a rubygems package does not exist in the public rubygems package repository.
//
// Returns a map of strings with any rubygem packages not in the public rubygems package repository
func (r *RubyGemsLookup) PackagesNotInPublic() []models.PackageManager {
	var notAvail []models.PackageManager
	for _, pkg := range r.Packages {
		if pkg.IsLocal || !pkg.IsRubyGems {
			continue
		}
		if !r.isAvailableInPublic(pkg.Name, 0) {
			notAvail = append(notAvail, models.PackageManager{Name: "gem", Package: pkg.Name, Version: pkg.Version})
		}
	}
	return notAvail
}

// isAvailableInPublic determines if a rubygems package exists in the public rubygems.org package repository.
//
// Returns true if the package exists in the public rubygems package repository.
func (r *RubyGemsLookup) isAvailableInPublic(pkgname string, retry int) bool {
	if retry > 3 {
		log.Warnf(" [W] Maximum number of retries exhausted for package: %s\n", pkgname)
		return false
	}

	url := fmt.Sprintf("https://rubygems.org/api/v1/gems/%s.json", pkgname)
	if r.Verbose {
		log.Infof("Checking: %s : \n", url)
	}
	resp, err := http.Get(url)
	if err != nil {
		log.Warnf(" [W] Error when trying to request %s: %s\n", url, err)
		return false
	}
	defer resp.Body.Close()
	if r.Verbose {
		log.Infof("%s\n", resp.Status)
	}
	if resp.StatusCode == http.StatusOK {
		rubygemsResp := RubyGemsResponse{}
		body, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(body, &rubygemsResp)
		if err != nil {
			// This shouldn't ever happen because if it doesn't return JSON, it likely has returned
			// a non-200 status code.
			log.Warnf(" [W] Error when trying to unmarshal response from %s: %s\n", url, err)
			return false
		}
		return true
	} else if resp.StatusCode == 429 {
		log.Warnf(" [!] Server responded with 429 (Too many requests), throttling and retrying...\n")
		time.Sleep(10 * time.Second)
		retry = retry + 1
		return r.isAvailableInPublic(pkgname, retry)
	}
	return false
}
