package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/VictoriaMetrics/metricsql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Config struct {
	ListenAddress string   `yaml:"listenAddress"`
	ListenPort    int      `yaml:"listenPort"`
	Interval      int      `yaml:"interval"`
	Servers       []Server `yaml:"servers"`
}

type Server struct {
	Tenant    string `yaml:"tenant"`
	AlertURL  string `yaml:"alertUrl"`
	SelectURL string `yaml:"selectUrl"`
}

type Groups struct {
	Data struct {
		Groups []struct {
			Rules []struct {
				Name       string `json:"name"`
				Expression string `json:"query"`
				Type       string `json:"type"`
			} `json:"rules"`
		} `json:"groups"`
	} `json:"data"`
}

type Query struct {
	Data struct {
		Result []struct {
			Metric interface{} `json:"metric"`
			Value  interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

var (
	vectors []string
	Client  HTTPClient

	metric_present_1_day = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "alertsvalidator_present_over_1day",
			Help: "does the alert's metrics contains data point over 1 day",
		},
		[]string{
			"tenant",
			"alertname",
		},
	)
	metric_present_30_days = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "alertsvalidator_present_over_30days",
			Help: "does the alert's metric contains data point over the last 30 days",
		},
		[]string{
			"tenant",
			"alertname",
		},
	)
	metric_present_90_days = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "alertsvalidator_present_over_90days",
			Help: "does the alert's metric contains data point over the last 90 days",
		},
		[]string{
			"tenant",
			"alertname",
		},
	)
)

func main() {
	confPath := flag.String("conf", "config.yaml", "Config path")
	flag.Parse()

	config := loadConf(*confPath)
	listenAddress := config.ListenAddress
	listenPort := config.ListenPort
	interval := config.Interval

	// Init http client
	Client = &http.Client{}

	for _, server := range config.Servers {
		go checkRules(server, interval)
	}

	log.Println("Listening on " + listenAddress + ":" + fmt.Sprint(listenPort) + "/metrics")
	http.Handle("/metrics", promhttp.Handler())
	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	err := http.ListenAndServe(listenAddress+":"+fmt.Sprint(listenPort), nil)
	if err != nil {
		log.Fatalln(err)
	}
}

func loadConf(confPath string) *Config {
	yamlFile, err := ioutil.ReadFile(confPath)
	if err != nil {
		log.Printf("yamlFile.Get err #%v ", err)
		os.Exit(1)
	}
	var c Config
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
		os.Exit(1)
	}
	return &c
}

func checkRules(server Server, interval int) {
	for {
		groups := getRules(server.AlertURL)
		for _, group := range groups.Data.Groups {
			for _, rule := range group.Rules {
				if rule.Type != "alerting" {
					continue
				}
				vectors = make([]string, 0)
				expr, err := metricsql.Parse(rule.Expression)
				metricsql.VisitAll(expr, tree)

				if err != nil {
					continue
				}
				for _, vector := range vectors {
					metric_present_1_day.WithLabelValues(server.Tenant, rule.Name).Set(1)
					metric_present_30_days.WithLabelValues(server.Tenant, rule.Name).Set(1)
					metric_present_90_days.WithLabelValues(server.Tenant, rule.Name).Set(1)
					if !existingMetric(vector, server.SelectURL) {
						if !existingMetric("present_over_time("+vector+"[30d])", server.SelectURL) {
							if !existingMetric("present_over_time("+vector+"[90d])", server.SelectURL) {
								metric_present_90_days.WithLabelValues(server.Tenant, rule.Name).Set(0)
							}
							metric_present_30_days.WithLabelValues(server.Tenant, rule.Name).Set(0)
						}
						metric_present_1_day.WithLabelValues(server.Tenant, rule.Name).Set(0)
					}
				}
			}
		}
		time.Sleep(time.Duration(interval) * time.Minute)
	}
}

func getRules(server string) Groups {
	url := server + `/api/v1/rules`
	req, _ := http.NewRequest("GET", url, nil)

	res, err := Client.Do(req)
	var gr Groups

	if err != nil {
		fmt.Println(err)
		return gr
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(string(b))
	} else {
		json.NewDecoder(res.Body).Decode(&gr)
	}

	return gr
}

func existingMetric(query, server string) bool {
	url := server + `/api/v1/query`
	payload := `query=` + query
	req, _ := http.NewRequest("POST", url, strings.NewReader(payload))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := Client.Do(req)

	var qu Query

	if err != nil {
		return false
	}
	defer res.Body.Close()
	json.NewDecoder(res.Body).Decode(&qu)

	return len(qu.Data.Result) > 0
}

func tree(expr metricsql.Expr) {
	switch e := expr.(type) {
	case *metricsql.MetricExpr:
		vectors = append(vectors, string(e.AppendString(nil)))
	}
}
