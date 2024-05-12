package gau

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"dependency-confusion/internal/gau/pkg/output"
	"dependency-confusion/internal/gau/runner"
	"dependency-confusion/runconfig"

	log "github.com/sirupsen/logrus"
)

func Gau(domain string, tempDir string, cfg *runconfig.Config) (string, error) {

	config, err := cfg.ProviderConfig()
	if err != nil {
		log.Errorln(err)
		return "", err
	}

	gau := new(runner.Runner)

	if err = gau.Init(config, cfg.Providers, cfg.Filters); err != nil {
		log.Warn(err)
	}

	results := make(chan string)

	var out = os.Stdout
	var outputFile string
	if config.JSON {
		outputFile = filepath.Join(tempDir, "parse_files.js")
	} else {
		outputFile = filepath.Join(tempDir, "parse_files.txt")
	}
	out, err = os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Errorf("Could not open output file: %v\n", err)
		return "", err
	}
	defer out.Close()
	// }

	var writeWg sync.WaitGroup
	writeWg.Add(1)
	go func(out io.Writer, JSON bool) {
		defer writeWg.Done()
		if JSON {
			output.WriteURLsJSON(out, results, config.Blacklist, config.RemoveParameters)
		} else if err = output.WriteURLs(out, results, config.Blacklist, config.RemoveParameters); err != nil {
			log.Errorf("error writing results: %v\n", err)
		}
	}(out, config.JSON)

	workChan := make(chan runner.Work)
	gau.Start(workChan, results)

	for _, provider := range gau.Providers {
		workChan <- runner.NewWork(domain, provider)
	}

	close(workChan)

	// wait for providers to fetch URLS
	gau.Wait()

	// close results channel
	close(results)

	// wait for writer to finish output
	writeWg.Wait()

	return outputFile, nil
}
