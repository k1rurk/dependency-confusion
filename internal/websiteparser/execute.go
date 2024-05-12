package websiteparser

import (
	"context"
	"dependency-confusion/internal/websiteparser/pkg/crawler"
	"dependency-confusion/internal/websiteparser/pkg/file"
	"dependency-confusion/tools"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

type CloneAgent struct {
	ScrapeopsAPIKey string
	Cookies         []string
	ProxyString     string
	TempDir         string
}

var packageFiles = []string{
	"/composer.json",
	"/composer.lock",
	"/.composer/composer.json",
	"/vendor/composer/installed.json",
	"/env.js",
	"/env.development.js",
	"/env.production.js",
	"/env.test.js",
	"/env.dev.js",
	"/env.prod.js",
	"/pyproject.toml",
	"/webpack.config.js",
	"/package.json",
	"/package-lock.json",
	"/yarn.lock",
	"/webpack.mix.js",
	"/packages.config",
	"/.nuget/packages.config",
	"/npm-shrinkwrap.json",
	"/Pipfile",
	"/.dockerfile",
	"/.Dockerfile",
	"/Dockerfile",
}

func New(cookies []string, scrapeopsAPIKey string, tempDir string) *CloneAgent {
	return &CloneAgent{Cookies: cookies, ScrapeopsAPIKey: scrapeopsAPIKey, TempDir: tempDir}
}

// Execute the clone command
func (c *CloneAgent) Execute(url string) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := c.CloneSite(ctx, url); err != nil {
		log.Fatalf("%+v", err)
	}
}

func (c *CloneAgent) CheckPackageFiles(url, projectPath string) error {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	
	for _, pathFile := range packageFiles {
		uri := url + pathFile
		resp, err := client.Get(uri)
		if err != nil {
			log.Warnf("[W] Error when trying to request %s: %s\n", uri, err)
			return err
		}
		defer resp.Body.Close()
		log.Infof("%s\n", resp.Status)
		if resp.StatusCode == http.StatusOK {
			//check response content type
			ctype := resp.Header.Get("Content-Type")
			if strings.HasPrefix(ctype, "text/html") {
				log.Warnf("response content type was %s text/html\n", ctype)
				continue
			}
			// get the raw extension from the file
			ext := path.Ext(pathFile)
			var localFile string
			if ext == ".js" {
				dirPath := crawler.ExtensionDir[ext]
				localFile = filepath.Join(dirPath, pathFile)
			} else {
				localFile = pathFile
			}
			f, err := os.OpenFile(filepath.Join(projectPath, localFile), os.O_RDWR|os.O_CREATE, 0777)
			if err != nil {
				return err
			}
			defer f.Close()
			data, err := io.ReadAll(resp.Body)

			if err != nil {
				return err
			}
			f.Write(data)
		}
	}
	return nil
}

func (c *CloneAgent) CloneSite(ctx context.Context, uriString string) error {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		return err
	}
	var cs []*http.Cookie
	if len(c.Cookies) != 0 {
		cs = make([]*http.Cookie, 0, len(c.Cookies))
		for _, c := range c.Cookies {
			ff := strings.Fields(c)
			for _, f := range ff {
				var k, v string
				if i := strings.IndexByte(f, '='); i >= 0 {
					k, v = f[:i], strings.TrimRight(f[i+1:], ";")
				} else {
					return fmt.Errorf("no = in cookie %q", c)
				}
				cs = append(cs, &http.Cookie{Name: k, Value: v})
			}
		}
		u, err := url.Parse(uriString)
		if err != nil {
			return fmt.Errorf("%q: %w", uriString, err)
		}
		jar.SetCookies(&url.URL{Scheme: u.Scheme, User: u.User, Host: u.Host}, cs)

	}

	isValid, isValidDomain := tools.ValidateURL(uriString), tools.ValidateDomain(uriString)
	if !isValid && !isValidDomain {
		return fmt.Errorf("%q is not valid", uriString)
	}
	name := uriString
	if isValidDomain {
		uriString = tools.CreateURL(name)
	} else {
		name = tools.GetDomain(uriString)
	}
	projectPath := file.CreateProject(name, c.TempDir)

	err = c.CheckPackageFiles(uriString, projectPath)
	if err != nil {
		return err
	}
	// make temp dir
	dirCache, err := os.MkdirTemp("", "dependency_confusion_collector_cache")
	if err != nil {
		return err
	}
	log.Infoln("Temp dir name for cache:", dirCache)
	defer os.RemoveAll(dirCache)
	if err := crawler.Crawl(ctx, uriString, projectPath, jar, c.ProxyString, c.ScrapeopsAPIKey, dirCache); err != nil {
		return fmt.Errorf("%q: %w", uriString, err)
	}

	return nil
}
