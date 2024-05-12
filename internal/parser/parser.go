/*
Package main implements an automated Dependency Confusion scanner.

Original research provided by Alex Birsan.

Original blog post detailing Dependency Confusion : https://medium.com/@alex.birsan/dependency-confusion-4a5d60fec610 .
*/
package parser

import (
	"dependency-confusion/internal/models"
	"dependency-confusion/internal/parser/composer"
	"dependency-confusion/internal/parser/docker"
	"dependency-confusion/internal/parser/interfaces"
	"dependency-confusion/internal/parser/javamvn"
	"dependency-confusion/internal/parser/nodejs"
	"dependency-confusion/internal/parser/nuget"
	"dependency-confusion/internal/parser/python"
	"dependency-confusion/internal/parser/ruby"
	"dependency-confusion/tools"
	"strings"

	log "github.com/sirupsen/logrus"
)

func Parse(fileName string, file []byte, verbose bool) []models.PackageManager {
	var resolver interfaces.PackageResolver

	if strings.HasSuffix(fileName, ".csproj") {
		resolver = nuget.NewNugetLookup(verbose, fileName)
	} else if ext := tools.URLExtension(fileName); ext == ".js" || ext == ".css" {
		resolver = nodejs.NewNPMLookup(verbose, fileName, ext)
	} else {
		switch fileName {
		case "pyproject.toml", "pdm.lock", "requirements.txt", "requirements.in", "Pipfile", "setup.cfg":
			resolver = python.NewPythonLookup(verbose, fileName)
		case "package.json", "package-lock.json", "yarn.lock":
			resolver = nodejs.NewNPMLookup(verbose, fileName, "")
		case "composer.json":
			resolver = composer.NewComposerLookup(verbose)
		case "installed.json", "composer.lock":
			resolver = composer.NewComposerInstalledLookup(verbose)
		case "pom.xml", "build.gradle":
			resolver = javamvn.NewMVNLookup(verbose, fileName)
		case "Gemfile", "Gemfile.lock":
			resolver = ruby.NewRubyGemsLookup(verbose, fileName)
		case "Dockerfile":
			resolver = docker.NewDockerLookup(verbose)
		case "packages.config":
			resolver = nuget.NewNugetLookup(verbose, fileName)
		default:
			log.Errorf("Unknown standart package manager file: %s\n", fileName)
			return nil
		}
	}

	err := resolver.ReadPackagesFromFile(file)
	if err != nil {
		log.Errorf("Encountered an error while trying to read packages from file: %s\n", err)
	}
	// outputPackages := removeSafe(resolver.PackagesNotInPublic(), safespaces)
	return resolver.PackagesNotInPublic()
}

// removeSafe removes known-safe package names from the slice
// func removeSafe(packages []string, safespaces string) []string {
// 	retSlice := []string{}
// 	safeNamespaces := []string{}
// 	var ignored bool
// 	safeTmp := strings.Split(safespaces, ",")
// 	for _, s := range safeTmp {
// 		safeNamespaces = append(safeNamespaces, strings.TrimSpace(s))
// 	}
// 	for _, p := range packages {
// 		ignored = false
// 		for _, s := range safeNamespaces {
// 			ok, err := filepath.Match(s, p)
// 			if err != nil {
// 				fmt.Printf(" [W] Encountered an error while trying to match a known-safe namespace %s : %s\n", s, err)
// 				continue
// 			}
// 			if ok {
// 				ignored = true
// 			}
// 		}
// 		if !ignored {
// 			retSlice = append(retSlice, p)
// 		}
// 	}
// 	return retSlice
// }
