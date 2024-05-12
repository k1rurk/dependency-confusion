package providers

import (
	"context"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/valyala/fasthttp"
)


// Provider is a generic interface for all archive fetchers
type Provider interface {
	Fetch(ctx context.Context, domain string, results chan string) error
	Name() string
}

type URLScan struct {
	Host   string
	APIKey string
}

type OTX struct {
	Host   string
	APIKey string
}

type Config struct {
	Threads           uint
	Timeout           uint
	MaxRetries        uint
	RemoveParameters  bool
	Client            *fasthttp.Client
	Providers         []string
	Blacklist         mapset.Set[string]
	Output            string
	JSON              bool
	URLScan           URLScan
	OTX               OTX
}
