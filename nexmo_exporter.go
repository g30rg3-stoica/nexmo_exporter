package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"gopkg.in/alecthomas/kingpin.v2"
)

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
func NewExporter(apiUrl, key, secret, namespace string, timeout time.Duration) (*Exporter, error) {
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

/* Prometheus ingerface implementation */

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

/* Nexmo API client implementation */

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

/* Misc */

/* Data structure to hold Nexmo API credentials */
type APICredentials struct {
	APIKey    string
	APISecret string
}

/*
* Retrieves API key - API secret tuple from file or error
 */
func readAPIAuthCredentials() (APICredentials, error) {
	jsonData, err := ioutil.ReadFile("/app/credentials/nexmo.json")

	if err != nil {
		log.Fatal("Failed to read API credentials: ", err)

		return APICredentials{}, err
	}

	var credentialsData APICredentials
	json.Unmarshal(jsonData, &credentialsData)

	return credentialsData, nil

}

func main() {

	var (
		telemetryPort = kingpin.Flag(
			"web.telemetry-port",
			"Port to listen on for web interface and telemetry.",
		).Default(":9100").String()

		metricsPath = kingpin.Flag(
			"web.telemetry-path",
			"Path under which to expose metrics.",
		).Default("/metrics").String()

		nexmoApiUrl = kingpin.Flag(
			"nexmo.url",
			"Nemo API URL",
		).Default("https://rest.nexmo.com").String()

		nexmoTimeout = kingpin.Flag(
			"nexmo.timeout",
			"Timeout for trying to get stats from Nexmo.",
		).Default("5s").Duration()

		nexmoNamespace = kingpin.Flag(
			"nexmo.namespace",
			"Prometheus namespace for Nexmo metrics",
		).Default("nexmo").String()
	)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version("0.1.0") // FIXME: parameterize this
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	// read API authentication credentials
	apiCredentials, err := readAPIAuthCredentials()

	if err != nil {
		panic(err)
	}

	exporter, err := NewExporter(*nexmoApiUrl,
		apiCredentials.APIKey,
		apiCredentials.APISecret,
		*nexmoNamespace,
		*nexmoTimeout,
	)

	if err != nil {
		log.Fatal(err)
	}

	prometheus.MustRegister(exporter)
	prometheus.MustRegister(prometheus.NewBuildInfoCollector())

	log.Infoln("Listening on", *telemetryPort)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Nexmo Exporter</title></head>
             <body>
             <h1>Nexmo Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	log.Fatal(http.ListenAndServe(*telemetryPort, nil))
}
