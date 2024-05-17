package types

import (
	"text/template"
	"os"
)

type NPMPackage struct {
	Name    string
	Version string
}

type PipPackage struct {
	Name    string
	Version string
	DNS string
}

type NpmDNS struct {
	Name string
	DNS string
}

type PackageObject interface {
	NpmDNS | NPMPackage | PipPackage
}

func ExecuteTemplateFile[P PackageObject](pObject P, templateFile, createdFilename string) error {
	var tmplPackageFile = templateFile
	tmpl, err := template.ParseFiles(tmplPackageFile)
	if err != nil {
		return err
	}
	file, err := os.Create(createdFilename)
	if err != nil {
		return err
	}
	defer file.Close()

	err = tmpl.Execute(file, pObject)
	if err != nil {
		return err
	}
	return nil
}