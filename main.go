package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
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

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Config struct {
	ListenAddress          string   `yaml:"listenAddress"`
	ListenPort             int      `yaml:"listenPort"`
	ComputeInterval        string   `yaml:"computeInterval"`
	ValidityCheckIntervals []string `yaml:"validityCheckIntervals"`
	Servers                []Server `yaml:"servers"`
	LabelKeys              []string `yaml:"labelKeys"`
}

type Server struct {
	LabelValues []string `yaml:"labelValues"`
	RuleURL     string   `yaml:"ruleUrl"`
	QueryURL    string   `yaml:"queryUrl"`
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
	Status    string `json:"status"`
	Error     string `json:"error"`
	ErrorType string `json:"errorTYpe"`
}

var (
	vectors []string
	Client  HTTPClient
	gauge   *prometheus.GaugeVec
	config  Config
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	confPath := flag.String("conf", "config.yaml", "Config path")
	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.Parse()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	config = *loadConf(*confPath)
	listenAddress := config.ListenAddress
	listenPort := config.ListenPort

	// Init http client
	Client = &http.Client{}

	// Build Gauges
	gauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "alertsvalidator_validity_range",
			Help: "Does the alert's metric contains data point in range",
		},
		append([]string{
			"alertname",
			"range_from",
			"range_to",
			"status",
		}, config.LabelKeys...),
	)

	for _, server := range config.Servers {
		go checkRules(server)
	}

	log.Info().Msgf("Listening on %s:%d/metrics", listenAddress, listenPort)
	http.Handle("/metrics", promhttp.Handler())
	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	err := http.ListenAndServe(listenAddress+":"+fmt.Sprint(listenPort), nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Cant launch metric server")
	}
}

func loadConf(confPath string) *Config {
	yamlFile, err := os.ReadFile(confPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Can't read config file")
		os.Exit(1)
	}
	var c Config
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		log.Fatal().Err(err).Msg("Unmarshal error")
		os.Exit(1)
	}
	return &c
}

func checkRules(server Server) {
	for {
		vectorValidityCache := map[string]bool{}
		groups := getRules(server.RuleURL)
		for _, group := range groups.Data.Groups {
			for _, rule := range group.Rules {
				if rule.Type != "alerting" {
					continue
				}
				vectors = []string{}
				expr, err := metricsql.Parse(rule.Expression)
				metricsql.VisitAll(expr, tree)

				if err != nil {
					continue
				}
				zDictVectorPresent := zerolog.Dict()
				zDictValid := zerolog.Dict()
				for key, interval := range config.ValidityCheckIntervals {
					zDictVector := zerolog.Dict()
					if key >= len(config.ValidityCheckIntervals)-1 {
						continue
					}
					checkedVector := []string{}
					valid := true
					for _, vector := range vectors {
						checked := false
						for _, v := range checkedVector {
							if v == vector {
								checked = true
							}
						}
						if checked {
							continue
						} else {
							checkedVector = append(checkedVector, vector)
						}
						existingM := false

						// Try to to see if we already checked a vector in this loop
						if val, ok := vectorValidityCache[fmt.Sprintf("%s[%s-%s]", vector, interval, config.ValidityCheckIntervals[key+1])]; ok {
							log.Debug().Str("alertname", rule.Name).Str("vector", vector).Bool("incache", true).Send()
							existingM = val
						} else {
							log.Debug().Str("alertname", rule.Name).Str("vector", vector).Bool("incache", false).Send()
							dur1, err := time.ParseDuration(config.ValidityCheckIntervals[key+1])
							if err != nil {
								log.Error().Err(err).Msg("Something is wrong with Validity Check Intervals")
								os.Exit(1)
							}
							dur2, err := time.ParseDuration(interval)
							if err != nil {
								log.Error().Err(err).Msg("Something is wrong with Validity Check Intervals")
								os.Exit(1)
							}
							dur := time.Until(time.Now().Add(dur1).Add(-dur2)).Truncate(time.Minute)
							check := fmt.Sprintf("present_over_time(%s[%s] offset %s)", vector, dur.String(), interval)
							_, err = metricsql.Parse(check)
							if err != nil {
								log.Error().Err(err).Str("check", check).Msg("Something is wrong with Validity Check Intervals")
								os.Exit(1)
							}
							log.Debug().Str("alertname", rule.Name).Str("check", check).Send()
							existingM = existingMetric(check, server.QueryURL)
							// Add vector in cache to avoid checking multiple times the same thing
							vectorValidityCache[fmt.Sprintf("%s[%s-%s]", vector, interval, config.ValidityCheckIntervals[key+1])] = existingM
						}
						if !existingM {
							zDictVector.Bool(vector, false)
							valid = false
						} else {
							zDictVector.Bool(vector, true)
						}
					}
					labelsValid := append([]string{rule.Name, interval, config.ValidityCheckIntervals[key+1], "valid"}, server.LabelValues...)
					labelsInvalid := append([]string{rule.Name, interval, config.ValidityCheckIntervals[key+1], "invalid"}, server.LabelValues...)
					if !valid {
						gauge.WithLabelValues(labelsValid...).Set(0)
						gauge.WithLabelValues(labelsInvalid...).Set(1)
						zDictValid.Bool(fmt.Sprintf("%s-%s", interval, config.ValidityCheckIntervals[key+1]), false)
					} else {
						gauge.WithLabelValues(labelsValid...).Set(1)
						gauge.WithLabelValues(labelsInvalid...).Set(0)
						zDictValid.Bool(fmt.Sprintf("%s-%s", interval, config.ValidityCheckIntervals[key+1]), true)
					}
					zDictVectorPresent.Dict(fmt.Sprintf("%s-%s", interval, config.ValidityCheckIntervals[key+1]), zDictVector)
				}
				log.Info().Str("alertname", rule.Name).Dict("is_vector_present", zDictVectorPresent).Dict("is_valid", zDictValid).Send()
			}
		}
		dur, err := time.ParseDuration(config.ComputeInterval)
		if err != nil {
			log.Fatal().Err(err).Msg("Can't parse Compute Interval")
			os.Exit(0)
		}
		time.Sleep(dur)
	}
}

func getRules(server string) Groups {
	url := server + `/api/v1/rules`
	req, _ := http.NewRequest("GET", url, nil)

	res, err := Client.Do(req)
	var gr Groups

	if err != nil {
		log.Error().Err(err)
		return gr
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatal().Err(err).Str("server", server).Msg("Can't get rules, and api response body can't be read")
		} else {
			log.Error().Str("server", server).Str("body", string(b)).Msg("Can't get rules")
		}
	} else {
		err := json.NewDecoder(res.Body).Decode(&gr)

		if err != nil {
			log.Error().Str("server", server).Err(err).Msg("Can't get rules")
		}
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
		log.Error().Err(err).Str("server", server).Msg("Can't query VM api")
		return false
	}
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&qu)

	if err != nil {
		log.Error().Err(err).Str("server", server).Msg("Can't decode VM api response")
		return false
	}

	if qu.Status == "error" {
		log.Error().Str("server", server).Str("error", qu.Error).Str("errorType", qu.ErrorType).Send()
	}

	return len(qu.Data.Result) > 0
}

func tree(expr metricsql.Expr) {
	switch e := expr.(type) {
	case *metricsql.MetricExpr:
		vectors = append(vectors, string(e.AppendString(nil)))
	}
}
