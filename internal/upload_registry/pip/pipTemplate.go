package pip

import (
	"dependency-confusion/internal/upload_registry/types"
	"path/filepath"
	"runtime"
)

func RunPIPTemplate(name, version, dns string) error {
	pipPackage := types.PipPackage{Name: name, Version: version, DNS: dns}
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	err := types.ExecuteTemplateFile(pipPackage, filepath.Join(basepath,"template","setup.tmpl"), filepath.Join(basepath,"sources", "setup.py"))
	if err != nil {
		return err
	}
	return nil
}