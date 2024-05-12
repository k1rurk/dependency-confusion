package handlers

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"dependency-confusion/internal/database"
	"dependency-confusion/internal/gau"
	gitmodule "dependency-confusion/internal/git_module"
	"dependency-confusion/internal/models"
	"dependency-confusion/internal/parser"
	container "dependency-confusion/internal/upload_registry"
	"dependency-confusion/internal/upload_registry/npm"
	"dependency-confusion/internal/upload_registry/pip"
	"dependency-confusion/internal/websiteparser"
	"dependency-confusion/runconfig"
	"dependency-confusion/tools"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"dependency-confusion/internal/report"
	"github.com/alitto/pond"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type HandlerConfig struct {
	DB     *sql.DB
	Config *runconfig.Config
	Cache []models.PackageManager
}

func New(db *sql.DB, config *runconfig.Config) *HandlerConfig {
	return &HandlerConfig{db, config, []models.PackageManager{}}
}

// @BasePath /api/v1

// RootDir godoc
// @Summary redirect to root
// @Schemes
// @Description redirect to root
// @Tags root
// @Header  302              {string}  Location  "/site"
// @Router / [get]
func RootDir(g *gin.Context) {
	g.Redirect(http.StatusFound, "/site")
}

// FindDomainPackages godoc
// @Summary Get vulnerable packages from site and gau
// @Description Get a list of vulnerable packages founded in site and gau tool
// @Tags gau
// @Produce json
// @Accept  json
// @Param inputGau body models.GauStruct true "Object for search"
// @Success 200 {object} []models.PackageManager "Successfully retrieved list of vulnerable packages"
// @Failure 400 {string}  string    "Bad Request"
// @Failure 500 {string}  string	"Internal Server Error"
// @Router /domain [post]
func (h *HandlerConfig) FindDomainPackages(c *gin.Context) {
	//TODO make faster with workers pool
	var inputGau models.GauStruct

	if err := c.ShouldBindJSON(&inputGau); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// make temp dir
	dname, err := os.MkdirTemp("", "dependency_confusion_temp")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		log.Errorln(err)
		return
	}
	log.Infoln("Temp dir name:", dname)
	defer os.RemoveAll(dname)

	h.Config.Threads = inputGau.Threads
	h.Config.Timeout = inputGau.Timeout
	h.Config.MaxRetries = inputGau.Retries
	h.Config.Verbose = true

	// gau
	outputFile, err := gau.Gau(inputGau.Domain, dname, h.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		log.Errorln(err)
		return
	}
	log.Infoln("Reading gau output file")
	readFile, err := os.Open(outputFile)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	var fileLines []string

	for fileScanner.Scan() {
		fileLines = append(fileLines, fileScanner.Text())
	}

	readFile.Close()

	// Create an unbuffered (blocking) pool with a fixed
	// number of workers
	pool := pond.New(int(h.Config.Threads), 0, pond.MinWorkers(int(h.Config.Threads)))
	defer pool.StopAndWait()
	group, ctx := pool.GroupContext(context.Background())

	checkedJSCSS := false
	var dependencies []models.PackageManager
	for _, line := range fileLines {
		group.Submit(func() error {
			checkedJSCSS = false
			filename := tools.URLFilename(line)
			extension := tools.URLExtension(line)
			for _, manifestFile := range gitmodule.ManifestFiles {
				if (filename == manifestFile || manifestFile == filepath.Ext(filename)) || ((extension == ".css" || extension == ".js") && !checkedJSCSS) {
					log.Infof("Check %s\n", filename)
					req, _ := http.NewRequestWithContext(ctx, http.MethodGet, line, nil)
					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						return err
					}
					defer resp.Body.Close()
					dupBody := new(bytes.Buffer)
					_ = io.TeeReader(resp.Body, dupBody)
					parsedDeps := parser.Parse(filename, dupBody.Bytes(), true)
					if parsedDeps != nil {
						dependencies = append(dependencies, parsedDeps...)
					}
					if extension == ".css" || extension == ".js" {
						checkedJSCSS = true
					}
				}
			}
			return nil
		})
	}

	err = group.Wait()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// webparser
	websiteparser.Run(inputGau.Domain, []string{}, dname, h.Config.ScrapeOps.ScrapeopsAPIKey)

	// Create an unbuffered (blocking) pool with a fixed
	// number of workers
	poolTwo := pond.New(int(h.Config.Threads), 0, pond.MinWorkers(int(h.Config.Threads)))
	defer poolTwo.StopAndWait()
	groupTwo, _ := poolTwo.GroupContext(context.Background())

	err = filepath.Walk(dname, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			groupTwo.Submit(func() error {
				checkedJSCSS = false
				filename := tools.URLFilename(path)
				extension := tools.URLExtension(path)
				for _, manifestFile := range gitmodule.ManifestFiles {
					if (filename == manifestFile || manifestFile == filepath.Ext(filename)) || ((extension == ".css" || extension == ".js") && !checkedJSCSS) {
						log.Infof("Check %s\n", filename)
						data, err := os.ReadFile(path)
						if err != nil {
							return err
						}
						parsedDeps := parser.Parse(filename, data, true)
						if parsedDeps != nil {
							dependencies = append(dependencies, parsedDeps...)
						}
						if extension == ".css" || extension == ".js" {
							checkedJSCSS = true
						}
					}
				}
				return nil
			})
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		log.Errorln(err)
		return
	}

	// Wait for all HTTP requests to complete.
	err = groupTwo.Wait()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		log.Errorln(err)
		return
	}

	h.Cache = dependencies
	
	c.JSON(http.StatusOK, gin.H{"data": dependencies})
}

// GetOrgRepos godoc
// @Summary Get organization git repositories
// @Description Get dependency confusion packages that was founded in organization git repositories
// @Tags org
// @Accept  json
// @Produce  json
// @Param org body string true "Get org repository packages"
// @Success 200 {object} []models.PackageManager "Successfully extracted vulnerable packages"
// @Failure 400 {string}  string    "Bad Request"
// @Failure 500 {string}  string	"Internal Server Error"
// @Router /github/org [post]
func (h *HandlerConfig) GetOrgRepos(c *gin.Context) {
	var input models.GitOrg

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gitOrg := gitmodule.NewOrg([]string{}, input.Org, h.Config.Git.AccessToken, true)

	dependencies, err := gitOrg.ScanAllRepos()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		log.Errorln(err)
		return
	}

	h.Cache = dependencies

	c.JSON(http.StatusOK, gin.H{"data": dependencies})
}

// GetRepo godoc
// @Summary Get a git repository
// @Description Get dependency confusion packages that was founded in git repositories
// @Tags repo
// @Accept  json
// @Produce  json
// @Param   name     body   string   true   "Get repository packages"
// @Success 200 {object} []models.PackageManager "Successfully extracted vulnerable packages"
// @Failure 400 {string}  string    "Bad Request"
// @Failure 500 {string}  string	"Internal Server Error"
// @Router /github/repo [post]
func (h *HandlerConfig) GetRepo(c *gin.Context) {
	var input models.GitRepo

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gitRepo := gitmodule.NewOrg([]string{}, "", h.Config.Git.AccessToken, true)

	dependencies, err := gitRepo.ScanRepo(input.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		log.Errorln(err)
		return
	}

	h.Cache = dependencies

	c.JSON(http.StatusOK, gin.H{"data": dependencies})
}

// ParseFile godoc
// @Summary Parse a file for dependency confusion
// @Description Parse a file for dependency confusion
// @Tags file
// @Accept  mpfd
// @Produce  json
// @Param   file formData file true "File for scanning"
// @Success 200 {object} []models.PackageManager "Successfully extracted vulnerable packages"
// @Failure 400 {string}  string    "Bad Request"
// @Failure 500 {string}  string	"Internal Server Error"
// @Router /parse/file [post]
func (h *HandlerConfig) ParseFile(c *gin.Context) {

	// single file
	file, err := c.FormFile("file")

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Errorln(err)
		return
	}

	log.Infoln(file.Filename)

	// make temp dir
	dname, err := os.MkdirTemp("", "dependency_confusion_temp")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		log.Errorln(err)
		return
	}
	log.Infoln("Temp dir name:", dname)
	defer os.RemoveAll(dname)

	founded := false
	var dependencies []models.PackageManager
	for _, manifestFile := range gitmodule.ManifestFiles {
		if file.Filename == manifestFile || manifestFile == filepath.Ext(file.Filename) {
			err := c.SaveUploadedFile(file, filepath.Join(dname, file.Filename))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				log.Errorln(err)
				return
			}
			data, err := os.ReadFile(filepath.Join(dname, file.Filename))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				log.Errorln(err)
				return
			}
			parsedDeps := parser.Parse(file.Filename, data, true)
			if parsedDeps != nil {
				dependencies = append(dependencies, parsedDeps...)
			}
			founded = true
		}
	}

	if !founded {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manifest file not founded"})
		log.Errorln(err)
		return
	}

	h.Cache = dependencies

	c.JSON(http.StatusOK, gin.H{"data": dependencies})
}

// ParseDir godoc
// @Summary Parse a directory for dependency confusion
// @Description Parse a directory for dependency confusion
// @Tags dir
// @Produce  json
// @Accept  mpfd
// @Param   files[] formData file true "Files for scanning"
// @Success 200 {object} []models.PackageManager "Successfully extracted vulnerable packages"
// @Failure 400 {string}  string    "Bad Request"
// @Failure 500 {string}  string	"Internal Server Error"
// @Router /parse/directory [post]
func (h *HandlerConfig) ParseDir(c *gin.Context) {

	// Multipart form
	form, err := c.MultipartForm()
	if err != nil {
		c.String(http.StatusBadRequest, "get form err: %s", err.Error())
		return
	}
	files := form.File["files[]"]

	// make temp dir
	dname, err := os.MkdirTemp("", "dependency_confusion_temp")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		log.Errorln(err)
		return
	}
	log.Infoln("Temp dir name:", dname)
	defer os.RemoveAll(dname)

	founded := false
	var dependencies []models.PackageManager
	for _, file := range files {
		filename := tools.PathFilename(file.Filename)
		for _, manifestFile := range gitmodule.ManifestFiles {
			if filename == manifestFile || manifestFile == filepath.Ext(filename) {
				err := c.SaveUploadedFile(file, filepath.Join(dname, filename))
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					log.Errorln(err)
					return
				}

				data, err := os.ReadFile(filepath.Join(dname, filename))
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					log.Errorln(err)
					return
				}
				parsedDeps := parser.Parse(filename, data, true)
				if parsedDeps != nil {
					dependencies = append(dependencies, parsedDeps...)
				}
				founded = true
			}
		}
	}

	if !founded {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manifest file not founded"})
		log.Errorln(err)
		return
	}

	h.Cache = dependencies

	c.JSON(http.StatusOK, gin.H{"data": dependencies})
}

// GetDNSData godoc
// @Summary GET DNS data from database
// @Description GET all rows from SQLite for DNS data
// @Tags dns
// @Produce  json
// @Success 200 {object} []models.DbPackage "Successfully extracted vulnerable packages from DNS"
// @Failure 500 {string}  string	"Internal Server Error"
// @Router /dns [get]
func (h *HandlerConfig) GetDNSData(c *gin.Context) {
	dbPackage, err := database.GetData(h.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		log.Errorln(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": dbPackage})
}

// SendPackageRegistry godoc
// @Summary POST data to registry
// @Description POST vulnerable version of package to corresponding registry
// @Tags registry
// @Produce  json
// @Accept  json
// @Param   input	body models.PackageManager true "Package object"
// @Success 200 {object} string "ok"
// @Failure 400 {string}  string    "Bad Request"
// @Failure 500 {string}  string	"Internal Server Error"
// @Router /registry [post]
func (h *HandlerConfig) SendPackageRegistry(c *gin.Context) {
	var input models.PackageManager

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Name == "npm" {
		err := npm.RunNPMTemplate(input.Package, input.Version, h.Config.DNSConfig.Domain)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Errorln(err)
			return
		}
	} else if input.Name == "pip" {
		err := pip.RunPIPTemplate(input.Package, input.Version, h.Config.DNSConfig.Domain)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Errorln(err)
			return
		}
	} else {
		c.String(http.StatusBadRequest, "wrong package manager: %s", input.Name)
		return
	}

	err := container.BuildAndRun(input.Name, "dependency_confusion", "dependency_container")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		log.Errorln(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}


func (h *HandlerConfig)GenerateReport(c *gin.Context) {
	var generatedContent [][]string

	for _, value := range h.Cache {
		tempArray := make([]string, 0, 3)
		tempArray[0] = value.Name
		tempArray[1] = value.Package
		tempArray[2] = value.Version
		generatedContent = append(generatedContent, tempArray)
	}
	
	err := report.Generate(generatedContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		log.Errorln(err)
		return
	}

	c.Redirect(http.StatusFound, "/site/pdfs/report.pdf")
}