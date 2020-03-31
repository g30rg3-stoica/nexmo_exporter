package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

const namespace = "nexmo"

// Exporter collects Nexmo stats and exports them using
// the prometheus metrics package.
type Exporter struct {
	URI    string
	client http.Client

	mutex        sync.RWMutex
	up           prometheus.Gauge
	totalScrapes prometheus.Counter
	balance      prometheus.Gauge
}

// NewExporter returns an initialized Exporter.
func NewExporter(apiUrl, key, secret string, timeout time.Duration) (*Exporter, error) {
	uri := apiUrl + "/account/get-balance/" + key + "/" + secret

	return &Exporter{
		URI:    uri,
		client: http.Client{Timeout: timeout},
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Was the last scrape of nexmo successful.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_total_scrapes",
			Help:      "Current total nexmo scrapes.",
		}),
		balance: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "balance",
			Help:      "Nexmo balance in euros.",
		}),
	}, nil
}

// Describe describes all the metrics. Implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.up.Desc()
	ch <- e.totalScrapes.Desc()
	ch <- e.balance.Desc()
}

// Collect fetches the stats and delivers them as Prometheus metrics.
// It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()

	e.scrape()

	ch <- e.up
	ch <- e.totalScrapes
	ch <- e.balance
}

func (e *Exporter) scrape() {
	e.totalScrapes.Inc()

	balance, err := e.getBalance()
	if err != nil {
		e.up.Set(0)
		log.Errorf("Can't get balance: %v", err)
		return
	}

	e.balance.Set(balance)
	e.up.Set(1)
}

type balanceResp struct {
	Value      float64 `json:"value"`
	AutoReload bool    `json:"autoReload"`
}

func (e *Exporter) getBalance() (float64, error) {
	req, err := http.NewRequest("GET", e.URI, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Add("Accept", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var balance balanceResp
	if err := json.Unmarshal(body, &balance); err != nil {
		return 0, err
	}
	return balance.Value, nil
}

func main() {

	var (
		listenAddress = kingpin.Flag(
			"web.listen-address",
			"Address to listen on for web interface and telemetry.",
		).Default(":9101").String()

		metricsPath = kingpin.Flag(
			"web.telemetry-path",
			"Path under which to expose metrics.",
		).Default("/metrics").String()

		nexmoApiUrl = kingpin.Flag(
			"nexmo.url",
			"Nemo API URL",
		).Default("https://rest.nexmo.com").String()

		nexmoAPIKey = kingpin.Flag(
			"nexmo.api-key",
			"Path under which to expose metrics.",
		).Default("").String()

		nexmoAPISecret = kingpin.Flag(
			"nexmo.api-secret",
			"Path under which to expose metrics.",
		).Default("").String()

		nexmoTimeout = kingpin.Flag(
			"nexmo.timeout",
			"Timeout for trying to get stats from Nexmo.",
		).Default("5s").Duration()
	)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("haproxy_exporter")) // FIXME: whats up with this
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	exporter, err := NewExporter(*nexmoApiUrl, *nexmoAPIKey, *nexmoAPISecret, *nexmoTimeout)
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("nexmo_exporter"))

	log.Infoln("Listening on", *listenAddress)
	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Nexmo Exporter</title></head>
             <body>
             <h1>Nexmo Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
