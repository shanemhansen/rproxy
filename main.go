package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/BurntSushi/toml"
)

var confflag = flag.String("conf", "", "proxy config file")
var dumpconfig = flag.Bool("dumpconfig", false, "dump config and exit")

type Config struct {
	Address      string
	TLS          bool
	KeyFile      string
	CertFile     string
	Host         []Host
	ApiKey       string
	HostHeader   string
	ApiKeyHeader string
	LogFile      string
}

func (c *Config) FromReader(rdr io.Reader) error {
	_, err := toml.DecodeReader(rdr, c)
	return err
}

type Host struct {
	URL string
}

func main() {
	flag.Parse()
	if *confflag == "" {
		log.Fatal("conf file is required")
	}
	f, err := os.Open(*confflag)
	if err != nil {
		log.Fatal(err)
	}
	var conf Config
	if err := conf.FromReader(f); err != nil {
		log.Fatal(err)
	}
	l, err := os.OpenFile(conf.LogFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	log.SetOutput(l)
	if *dumpconfig {
		data, err := json.MarshalIndent(conf, "", "\t")
		if err != nil {
			log.Fatal(err)
		}
		log.Fatal(string(data))
	}
	servefunc := http.ListenAndServe
	if conf.TLS {
		servefunc = func(addr string, handler http.Handler) error {
			return http.ListenAndServeTLS(addr, conf.CertFile, conf.KeyFile, handler)
		}
	}
	apiProxy, err := NewApiProxy(conf.Host, conf.ApiKey)
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(servefunc(conf.Address, Auth(apiProxy, &conf)))
}
func Auth(p *rproxy, conf *Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		clientKey := req.Header.Get(conf.ApiKeyHeader)
		if clientKey != conf.ApiKey {
			http.Error(w, "Bad rproxy key", http.StatusUnauthorized)
			return
		}
		intended := req.Header.Get(conf.HostHeader)
		uri, ok := p.proxies[intended]
		if !ok {
			http.Error(w, "Bad rproxy host", http.StatusBadRequest)
			return
		}
		// modify the request
		req.URL.Host = uri.Host
		req.URL.Scheme = uri.Scheme
		req.Host = uri.Host
		req.Header.Del(conf.ApiKeyHeader)
		req.Header.Del(conf.HostHeader)
		p.ServeHTTP(w, req)
	})
}
func NewApiProxy(urls []Host, apiKey string) (*rproxy, error) {
	p := &rproxy{
		proxies: make(map[string]*url.URL),
		proxy: &httputil.ReverseProxy{
			Director: func(r *http.Request) {
			},
		},
		apiKey: apiKey,
	}
	for _, address := range urls {
		uri, err := url.Parse(address.URL)
		if err != nil {
			return nil, err
		}
		p.proxies[uri.Host] = uri

	}
	return p, nil
}

type rproxy struct {
	proxies map[string]*url.URL
	proxy   *httputil.ReverseProxy
	apiKey  string
}

func (p *rproxy) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	p.proxy.ServeHTTP(resp, req)
}
