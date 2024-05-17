package npm

import (
	"dependency-confusion/internal/upload_registry/types"
	"path/filepath"
	"runtime"
)


func RunNPMTemplate(name, version, dns string) error {

	npmPackageJSON := types.NPMPackage{Name: name, Version: version}
	npmIndexJS := types.NpmDNS{Name: name, DNS: dns}
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)

	err := types.ExecuteTemplateFile(npmIndexJS, filepath.Join(basepath, "template", "index.tmpl"), filepath.Join(basepath, "sources", "index.js"))
	if err != nil {
		return err
	}

	err = types.ExecuteTemplateFile(npmPackageJSON, filepath.Join(basepath, "template", "package.tmpl"), filepath.Join(basepath, "sources", "package.json"))
	if err != nil {
		return err
	}
	return nil
}
