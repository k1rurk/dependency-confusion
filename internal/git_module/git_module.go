package gitmodule

import (
	"dependency-confusion/internal/models"
	"dependency-confusion/internal/parser"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	log "github.com/sirupsen/logrus"
)

var ManifestFiles = []string{
	".csproj",
	"packages.config",
	"build.gradle",
	"pom.xml",
	"composer.lock",
	"composer.json",
	"installed.json",
	"requirements.txt",
	"package-lock.json",
	"yarn.lock",
	"package.json",
	"pyproject.toml",
	"pdm.lock",
	"requirements.in",
	"Pipfile",
	"setup.cfg",
	"Gemfile",
	"Gemfile.lock",
	"Dockerfile",
}

type HttpClient struct {
	Client   http.Client
	ApiToken string
}

type Orgs struct {
	ExcludeRepos []string
	Name         string
	HttpClient   *HttpClient
	FileTree     map[string]TreeRepo
	Verbose      bool
}

type DefaultBranch struct {
	DefaultBrunch string `json:"default_branch"`
}

type ContentFile struct {
	ContentFile string `json:"content"`
}

type OrgRepos struct {
	OrgRepoName string `json:"full_name"`
}

type TreeRepo struct {
	Tree []TreeList `json:"tree"`
}

type TreeList struct {
	Path string `json:"path"`
}

// NewOrg constructs a `Orgs` struct and returns it
func NewOrg(excludeRepos []string, name, accessToken string, verbose bool) *Orgs {
	httpClient := &HttpClient{*http.DefaultClient, accessToken}
	var fileTree = map[string]TreeRepo{}
	return &Orgs{ExcludeRepos: excludeRepos, Name: name, HttpClient: httpClient, FileTree: fileTree, Verbose: verbose}
}

func (c *HttpClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.ApiToken))
	return c.Client.Do(req)

}

func (c *HttpClient) Get(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err

	}
	return c.Do(req)
}

func (o *Orgs) ScanAllRepos() ([]models.PackageManager, error) {
	orgRepos, err := o.GetOrgRepos()
	var dependencies []models.PackageManager
	if err != nil {
		return nil, err
	}
	for _, repo := range orgRepos {
		deps, err := o.ScanRepo(repo)
		if err != nil {
			return nil, err
		}
		dependencies = append(dependencies, deps...)
	}
	return dependencies, nil
}

// completed
func (o *Orgs) GetOrgRepos() ([]string, error) {
	page := 1
	var listOrgsRepos []string
	for {
		url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100&page=%v", o.Name, page)
		resp, err := o.HttpClient.Get(url)
		if err != nil {
			log.Errorln(err)
			return nil, err
		}
		defer resp.Body.Close()
		var orgRepos []OrgRepos
		if resp.StatusCode == http.StatusOK {
			err = json.NewDecoder(resp.Body).Decode(&orgRepos)
			if err != nil {
				log.Errorln(err)
				return nil, err
			}
			if len(orgRepos) == 0 {
				break
			}
			for _, repo := range orgRepos {
				if !slices.Contains(o.ExcludeRepos, repo.OrgRepoName) {
					listOrgsRepos = append(listOrgsRepos, repo.OrgRepoName)
				}
			}
		} else {
			return nil, fmt.Errorf("failed to fetch repositories. Status code: %s, Response: %s", resp.Status, resp.Body)
		}
		page += 1
	}
	return listOrgsRepos, nil
}

func (o *Orgs) ScanRepo(repoName string) ([]models.PackageManager, error) {
	log.Infoln("Scanning " + o.Name)
	var dependencies []models.PackageManager
	for _, manifestFileName := range ManifestFiles {
		arrayFilePaths, err := o.GetFilePaths(repoName, manifestFileName)
		if err != nil {
			return nil, err
		}
		for _, manifestPath := range arrayFilePaths {
			log.Infoln("Scanning " + manifestFileName)
			result, err := o.GetFileContent(repoName, manifestPath)
			if err != nil {
				return nil, err
			}
			depAfterParse := parser.Parse(manifestFileName, result, o.Verbose)
			if depAfterParse != nil {
				dependencies = append(dependencies, depAfterParse...)
			}
			// fmt.Println(string(result))
		}

	}
	return dependencies, nil
}

// completed
func (o *Orgs) GetDefaultRepoBranch(repoName string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s", repoName)
	resp, err := o.HttpClient.Get(url)
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	defer resp.Body.Close()
	var defaultBranch DefaultBranch
	if resp.StatusCode == http.StatusOK {
		err = json.NewDecoder(resp.Body).Decode(&defaultBranch)
		if err != nil {
			log.Errorln(err)
			return "", err
		}
		return defaultBranch.DefaultBrunch, nil
	} else {
		return "", fmt.Errorf("failed to fetch repository default branch. Status code: %s, Response: %s", resp.Status, resp.Body)
	}
}

// completed
func (o *Orgs) GetRepoFileTree(repoName string) error {
	defaultBranch, err := o.GetDefaultRepoBranch(repoName)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/trees/%s?recursive=1", repoName, defaultBranch)
	resp, err := o.HttpClient.Get(url)
	if err != nil {
		log.Errorln(err)
		return err
	}
	defer resp.Body.Close()
	var fileTree TreeRepo
	if resp.StatusCode == http.StatusOK {
		err := json.NewDecoder(resp.Body).Decode(&fileTree)
		if err != nil {
			log.Errorln(err)
			return err
		}
		o.FileTree[repoName] = fileTree
	} else {
		return fmt.Errorf("failed to fetch repository details. Status code: %s, Response: %s", resp.Status, resp.Body)
	}

	return nil
}

// completed
func (o *Orgs) GetFilePaths(repoName, fileName string) ([]string, error) {
	if _, exists := o.FileTree[repoName]; !exists {
		err := o.GetRepoFileTree(repoName)
		if err != nil {
			return nil, err
		}
	}
	var filePaths []string
	for _, file := range o.FileTree[repoName].Tree {
		if strings.HasSuffix(file.Path, fileName) || file.Path == fileName {
			filePaths = append(filePaths, file.Path)
		}
	}
	return filePaths, nil
}

// completed
func (o *Orgs) GetFileContent(repoName, filePath string) ([]byte, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", repoName, filePath)
	resp, err := o.HttpClient.Get(url)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	defer resp.Body.Close()
	var contentFile ContentFile
	if resp.StatusCode == http.StatusOK {
		err = json.NewDecoder(resp.Body).Decode(&contentFile)
		if err != nil {
			log.Errorln(err)
			return nil, err
		}
		return base64.StdEncoding.DecodeString(contentFile.ContentFile)
	} else {
		return nil, fmt.Errorf("failed to fetch file content. Status code: %s, Response: %s", resp.Status, resp.Body)
	}
}

