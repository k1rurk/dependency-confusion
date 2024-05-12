package composer

import (
	"dependency-confusion/internal/models"
	"dependency-confusion/internal/parser/interfaces"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type ComposerJSON struct {
	Require    map[string]string `json:"require"`
	RequireDev map[string]string `json:"require-dev"`
}

type ComposerLookup struct {
	Packages map[string]string
	Verbose  bool
}

func NewComposerLookup(verbose bool) interfaces.PackageResolver {
	return &ComposerLookup{Packages: map[string]string{}, Verbose: verbose}
}

func (c *ComposerLookup) ReadPackagesFromFile(rawfile []byte) error {

	data := ComposerJSON{}
	err := json.Unmarshal([]byte(rawfile), &data)
	if err != nil {
		return err
	}

	for pkgname, version := range data.Require {
		if _, exists := c.Packages[pkgname]; !exists {
			c.Packages[pkgname] = version
		}
	}

	for pkgname, version := range data.RequireDev {
		if _, exists := c.Packages[pkgname]; !exists {
			c.Packages[pkgname] = version
		}
	}

	return nil
}

func (c *ComposerLookup) PackagesNotInPublic() []models.PackageManager {
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

func (c *ComposerLookup) isAvailableInPublic(pkgname string, retry int) bool {
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
		log.Warnf(" [W] Error when trying to request https://packagist.org/packages/%s : %s\n", pkgname, err)

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
