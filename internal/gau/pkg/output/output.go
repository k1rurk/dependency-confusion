package output

import (
	gitmodule "dependency-confusion/internal/git_module"
	"dependency-confusion/tools"
	"io"
	"net/url"
	"path"
	"slices"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/bytebufferpool"
)

type JSONResult struct {
	Url string `json:"url"`
}

func WriteURLs(writer io.Writer, results <-chan string, blacklistMap mapset.Set[string], RemoveParameters bool) error {
	lastURL := mapset.NewThreadUnsafeSet[string]()
	log.Infoln("Writing to file")
	for result := range results {
		buf := bytebufferpool.Get()
		u, err := url.Parse(result)
		if err != nil {
			continue
		}
		if path.Ext(u.Path) == "" || blacklistMap.Contains(strings.ToLower(path.Ext(u.Path))) {
			continue
		}
		filename := tools.URLFilename(u.Path)
		extension := tools.URLExtension(u.Path)
		if !(slices.Contains(gitmodule.ManifestFiles, filename) || extension == ".js" || extension == ".css") {
			continue
		}

		if RemoveParameters && !lastURL.Contains(u.Host+u.Path) {
			continue
		}

		lastURL.Add(u.Host + u.Path)

		buf.B = append(buf.B, []byte(result)...)
		buf.B = append(buf.B, "\n"...)
		_, err = writer.Write(buf.B)
		if err != nil {
			return err
		}
		bytebufferpool.Put(buf)
	}
	return nil
}

func WriteURLsJSON(writer io.Writer, results <-chan string, blacklistMap mapset.Set[string], RemoveParameters bool) {
	var jr JSONResult
	enc := jsoniter.NewEncoder(writer)
	for result := range results {
		u, err := url.Parse(result)
		if err != nil {
			continue
		}
		if blacklistMap.Contains(strings.ToLower(path.Ext(u.Path))) {
			continue
		}
		jr.Url = result
		if err := enc.Encode(jr); err != nil {
			// todo: handle this error
			continue
		}
	}
}
