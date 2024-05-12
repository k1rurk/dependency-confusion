package runconfig

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"dependency-confusion/internal/gau/pkg/providers"
	"dependency-confusion/tools"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/lynxsecurity/viper"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

type URLScanConfig struct {
	Host   string `mapstructure:"host"`
	APIKey string `mapstructure:"apikey"`
}
type OTXConfig struct {
	Host   string `mapstructure:"host"`
	APIKey string `mapstructure:"apikey"`
}

type GitConfig struct {
	AccessToken string `mapstructure:"accesstoken"`
}

type ScrapeOpsConfig struct {
	ScrapeopsAPIKey string `mapstructure:"scrapeopsAPIKey"`
}

type DNSConfig struct {
	Domain       string `mapstructure:"domain"`
	PublicIP     string `mapstructure:"publicip"`
	Records      []RR   `mapstructure:"arecords"`
}

type RR struct {
	Hostname string `mapstructure:"hostname"`
	IP       string `mapstructure:"ip"`
}

type Config struct {
	Filters          providers.Filters `mapstructure:"filters"`
	Proxy            string            `mapstructure:"proxy"`
	Threads          uint              `mapstructure:"threads"`
	Timeout          uint              `mapstructure:"timeout"`
	Verbose          bool              `mapstructure:"verbose"`
	MaxRetries       uint              `mapstructure:"retries"`
	RemoveParameters bool              `mapstructure:"parameters"`
	Providers        []string          `mapstructure:"providers"`
	Blacklist        []string          `mapstructure:"blacklist"`
	JSON             bool              `mapstructure:"json"`
	URLScan          URLScanConfig     `mapstructure:"urlscan"`
	OTX              OTXConfig         `mapstructure:"otx"`
	Git              GitConfig         `mapstructure:"git"`
	ScrapeOps        ScrapeOpsConfig   `mapstructure:"scrapeops"`
	DNSConfig        DNSConfig         `mapstructure:"dns"`
}

func (c *Config) ProviderConfig() (*providers.Config, error) {
	var dialer fasthttp.DialFunc

	if c.Proxy != "" {
		parse, err := url.Parse(c.Proxy)
		if err != nil {
			return nil, fmt.Errorf("proxy url: %v", err)
		}
		switch parse.Scheme {
		case "http":
			dialer = fasthttpproxy.FasthttpHTTPDialer(strings.ReplaceAll(c.Proxy, "http://", ""))
		case "socks5":
			dialer = fasthttpproxy.FasthttpSocksDialer(c.Proxy)
		default:
			return nil, fmt.Errorf("unsupported proxy scheme: %s", parse.Scheme)
		}
	}

	pc := &providers.Config{
		Threads:          c.Threads,
		Timeout:          c.Timeout,
		MaxRetries:       c.MaxRetries,
		RemoveParameters: c.RemoveParameters,
		Client: &fasthttp.Client{
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			Dial: dialer,
		},
		Providers: c.Providers,
		JSON:      c.JSON,
		URLScan: providers.URLScan{
			Host:   c.URLScan.Host,
			APIKey: c.URLScan.APIKey,
		},
		OTX: providers.OTX{
			Host:   c.OTX.Host,
			APIKey: c.URLScan.APIKey,
		},
	}

	log.SetLevel(log.ErrorLevel)
	if c.Verbose {
		log.SetLevel(log.InfoLevel)
	}
	pc.Blacklist = mapset.NewThreadUnsafeSet(c.Blacklist...)
	pc.Blacklist.Add("")
	return pc, nil
}

type Options struct {
	viper *viper.Viper
}

func New() *Options {
	v := viper.New()

	return &Options{viper: v}
}

func (o *Options) ReadInConfig() (*Config, error) {
	projectDir := tools.GetDirectoryProject()

	confFile := filepath.Join(projectDir, "config", "config.toml")
	return o.ReadConfigFile(confFile)
}

func (o *Options) ReadConfigFile(name string) (*Config, error) {
	o.viper.SetConfigFile(name)

	if err := o.viper.ReadInConfig(); err != nil {
		return o.DefaultConfig(), err
	}

	var c Config

	if err := o.viper.Unmarshal(&c); err != nil {
		return o.DefaultConfig(), err
	}

	o.getFlagValues(&c)

	return &c, nil
}

func (o *Options) DefaultConfig() *Config {
	c := &Config{
		Filters:          providers.Filters{},
		Proxy:            "",
		Timeout:          45,
		Threads:          1,
		Verbose:          false,
		MaxRetries:       5,
		RemoveParameters: false,
		Providers:        []string{"wayback", "commoncrawl", "otx", "urlscan"},
		Blacklist:        []string{},
		JSON:             false,
	}

	o.getFlagValues(c)

	return c
}

func (o *Options) getFlagValues(c *Config) {
	verbose := o.viper.GetBool("verbose")
	json := o.viper.GetBool("json")
	retries := o.viper.GetUint("retries")
	proxy := o.viper.GetString("proxy")
	fetchers := o.viper.GetStringSlice("providers")
	threads := o.viper.GetUint("threads")
	blacklist := o.viper.GetStringSlice("blacklist")
	fp := o.viper.GetBool("fp")

	if proxy != "" {
		c.Proxy = proxy
	}

	// set if --threads flag is set, otherwise use default
	if threads > 1 {
		c.Threads = threads
	}

	// set if --blacklist flag is specified, otherwise use default
	if len(blacklist) > 0 {
		c.Blacklist = blacklist
	}

	// set if --providers flag is specified, otherwise use default
	if len(fetchers) > 0 {
		c.Providers = fetchers
	}

	if retries > 0 {
		c.MaxRetries = retries
	}

	if fp {
		c.RemoveParameters = fp
	}

	c.JSON = json
	c.Verbose = verbose

	// get filter flags
	mc := o.viper.GetStringSlice("mc")
	fc := o.viper.GetStringSlice("fc")
	mt := o.viper.GetStringSlice("mt")
	ft := o.viper.GetStringSlice("ft")
	from := o.viper.GetString("from")
	to := o.viper.GetString("to")

	var seenFilterFlag bool

	var filters providers.Filters
	if len(mc) > 0 {
		seenFilterFlag = true
		filters.MatchStatusCodes = mc
	}

	if len(fc) > 0 {
		seenFilterFlag = true
		filters.FilterStatusCodes = fc
	}

	if len(mt) > 0 {
		seenFilterFlag = true
		filters.MatchMimeTypes = mt
	}

	if len(ft) > 0 {
		seenFilterFlag = true
		filters.FilterMimeTypes = ft
	}

	if from != "" {
		seenFilterFlag = true
		if _, err := time.Parse("200601", from); err == nil {
			filters.From = from
		}
	}

	if to != "" {
		seenFilterFlag = true
		if _, err := time.Parse("200601", to); err == nil {
			filters.To = to
		}
	}

	if seenFilterFlag {
		c.Filters = filters
	}
}
