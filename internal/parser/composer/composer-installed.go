package composer

import (
	"dependency-confusion/internal/models"
	"dependency-confusion/internal/parser/interfaces"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/perimeterx/marshmallow"
	log "github.com/sirupsen/logrus"
)

type ComposerInstalledJSON []struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Require    map[string]string `json:"require"`
	RequireDev map[string]string `json:"require-dev"`
}

type ComposerInstalled struct {
	Packages    []Dependencies `json:"packages"`
	PackagesDev []Dependencies `json:"packages-dev"`
}

type Dependencies struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ComposerInstalledLookup struct {
	Packages map[string]string
	Verbose  bool
}

func NewComposerInstalledLookup(verbose bool) interfaces.PackageResolver {
	return &ComposerInstalledLookup{Packages: map[string]string{}, Verbose: verbose}
}

func (c *ComposerInstalledLookup) ReadPackagesFromFile(rawfile []byte) error {

	data := ComposerInstalledJSON{}
	dataTwo := ComposerInstalled{}
	err := json.Unmarshal([]byte(rawfile), &data)
	if err != nil {
		_, err = marshmallow.Unmarshal([]byte(rawfile), &dataTwo)
		if err != nil {
			return err
		}
	}

	for i := 0; i < len(data); i++ {
		if _, exists := c.Packages[data[i].Name]; !exists {
			c.Packages[data[i].Name] = data[i].Version
		}

		for pkgname, version := range data[i].Require {
			if _, exists := c.Packages[pkgname]; !exists {
				c.Packages[pkgname] = version
			}
		}

		for pkgname, version := range data[i].RequireDev {
			if _, exists := c.Packages[pkgname]; !exists {
				c.Packages[pkgname] = version
			}
		}

	}

	for i := 0; i < len(dataTwo.Packages); i++ {
		if _, exists := c.Packages[dataTwo.Packages[i].Name]; !exists {
			c.Packages[dataTwo.Packages[i].Name] = dataTwo.Packages[i].Version
		}
	}

	for i := 0; i < len(dataTwo.PackagesDev); i++ {
		if _, exists := c.Packages[dataTwo.PackagesDev[i].Name]; !exists {
			c.Packages[dataTwo.PackagesDev[i].Name] = dataTwo.PackagesDev[i].Version
		}
	}

	return nil
}

func (c *ComposerInstalledLookup) PackagesNotInPublic() []models.PackageManager {
	var notAvail []models.PackageManager
	for pkg, version := range c.Packages {
		if pkg == "php" {
			continue
		}

		if !c.isAvailableInPublic(pkg, 0) {
			notAvail = append(notAvail, models.PackageManager{Name: "composer", Package: pkg, Version: version})
		}
	}

	return notAvail
}

func (c *ComposerInstalledLookup) isAvailableInPublic(pkgname string, retry int) bool {
	if retry > 3 {
		log.Warnf(" [W] Maximum number of retries exhausted for package %s\n", pkgname)

		return false
	}

	// check if the package is specifically a platform package https://getcomposer.org/doc/01-basic-usage.md#platform-packages
	if strings.HasPrefix(pkgname, "ext-") {
		return true
	}

	if c.Verbose {
		log.Infof("Checking: https://packagist.org/packages/%s : ", pkgname)
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get("https://packagist.org/packages/" + pkgname)
	if err != nil {
		log.Errorf(" [W] Error when trying to request https://packagist.org/packages/%s : %s\n", pkgname, err)

		return false
	}
	defer resp.Body.Close()

	if c.Verbose {
		log.Infof("%s\n", resp.Status)
	}

	if resp.StatusCode == http.StatusOK {
		return true
	}

	if resp.StatusCode == 429 {
		log.Warnf(" [!] Server responded with 429 (Too many requests), throttling and retrying..\n")
		time.Sleep(10 * time.Second)
		retry = retry + 1

		c.isAvailableInPublic(pkgname, retry)
	}

	return false
}
