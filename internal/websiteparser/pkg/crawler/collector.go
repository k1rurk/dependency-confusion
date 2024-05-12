package crawler

import (
	"context"
	"dependency-confusion/tools"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
)

type FakeBrowserHeadersResponse struct {
	Result []map[string]string `json:"result"`
}

func RandomHeader(headersList []map[string]string) map[string]string {
	randomIndex := rand.Intn(len(headersList))
	return headersList[randomIndex]
}

func GetHeadersList(scrapeopsAPIKey string) []map[string]string {

	// ScrapeOps Browser Headers API Endpint
	scrapeopsAPIEndpoint := "https://headers.scrapeops.io/v1/browser-headers?api_key=" + scrapeopsAPIKey

	req, _ := http.NewRequest("GET", scrapeopsAPIEndpoint, nil)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make Request
	resp, err := client.Do(req)
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()

		// Convert Body To JSON
		var fakeBrowserHeadersResponse FakeBrowserHeadersResponse
		json.NewDecoder(resp.Body).Decode(&fakeBrowserHeadersResponse)
		return fakeBrowserHeadersResponse.Result
	}

	var emptySlice []map[string]string
	return emptySlice
}

// Collector searches for css, js, and images within a given link
// TODO improve for better performance
func Collector(ctx context.Context, url string, projectPath string, cookieJar *cookiejar.Jar, proxyString string, scrapeopsAPIKey string, dirCache string) error {
	// create a new collector
	c := colly.NewCollector(
		colly.Async(true),
		// Visit only domains by specified url
		colly.AllowedDomains(tools.GetDomain(url)),
		// Cache responses to prevent multiple download of pages
		// even if the collector is restarted
		colly.CacheDir(filepath.Join(dirCache, fmt.Sprintf("/%s_cache", tools.GetDomain(url)))),
	)
	setUpCollector(c, ctx, cookieJar, proxyString)

	// Get Fake Browser Headers From API
	headersList := GetHeadersList(scrapeopsAPIKey)

	// Set Random Fake Browser Headers
	if len(headersList) != 0 {
		c.OnRequest(func(r *colly.Request) {
			randomHeader := RandomHeader(headersList)
			for key, value := range randomHeader {
				r.Headers.Set(key, value)
			}
		})
	}

	// Visit next page
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		// Extract the linked URL from the anchor tag
		link := e.Attr("href")
		log.Infoln("Next page link found:", link)
		c.Visit(e.Request.AbsoluteURL(link))
	})

	// Limit the number of threads started by colly to two
	// when visiting links which domains' matches "*" glob
	c.Limit(&colly.LimitRule{
		Parallelism: 2,
		Delay:       5 * time.Second,
	})

	// Set up an error callback
	c.OnError(func(r *colly.Response, err error) {
		log.Infof("Request URL: %s failed with response: %v\n", r.Request.URL, r)
	})

	// search for all link tags that have a rel attribute that is equal to stylesheet - CSS
	c.OnHTML("link[rel='stylesheet']", func(e *colly.HTMLElement) {
		// hyperlink reference
		link := e.Attr("href")
		// Info css file was found
		log.Infoln("Css found", "-->", link)
		// extraction
		Extractor(e.Request.AbsoluteURL(link), projectPath)
	})

	// search for all script tags with src attribute -- JS
	c.OnHTML("script[src]", func(e *colly.HTMLElement) {
		// src attribute
		link := e.Attr("src")
		// Info link
		log.Infoln("Js found", "-->", link)
		// extraction
		Extractor(e.Request.AbsoluteURL(link), projectPath)
	})

	// search for all script tags with src attribute -- JS
	c.OnHTML("script[data-src]", func(e *colly.HTMLElement) {
		// src attribute
		link := e.Attr("data-src")
		// Info link
		log.Infoln("Js found", "-->", link)
		// extraction
		Extractor(e.Request.AbsoluteURL(link), projectPath)
	})

	// Visit each url and wait for stuff to load :)
	if err := c.Visit(url); err != nil {
		return err
	}
	c.Wait()
	return nil
}

type cancelableTransport struct {
	ctx       context.Context
	transport http.RoundTripper
}

func (t cancelableTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.ctx.Err(); err != nil {
		return nil, err
	}
	return t.transport.RoundTrip(req.WithContext(t.ctx))
}

func setUpCollector(c *colly.Collector, ctx context.Context, cookieJar *cookiejar.Jar, proxyString string) {
	if cookieJar != nil {
		c.SetCookieJar(cookieJar)
	}
	if proxyString != "" {
		c.SetProxy(proxyString)
	} else {
		c.WithTransport(cancelableTransport{ctx: ctx, transport: http.DefaultTransport})
	}
	extensions.Referer(c)

}
