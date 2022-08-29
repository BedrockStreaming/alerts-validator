package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

type Response struct {
	Data      Data   `json:"data"`
	Status    string `json:"status"`
	Error     string `json:"error"`
	ErrorType string `json:"errorTYpe"`
}

type Data struct {
	Groups  []Groups `json:"groups"`
	Results []Result `json:"result"`
}

type Result struct {
	Metric interface{} `json:"metric"`
	Value  interface{} `json:"value"`
}

type Groups struct {
	Rules []Rule `json:"rules"`
}

type Rule struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Expression string `json:"query"`
	Type       string `json:"type"`
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var (
	Client HTTPClient
)

func getRules(server Server) Response {
	labels := append([]string{"rule", server.RuleURL}, server.LabelValues...)
	url := server.RuleURL + `/api/v1/rules`
	req, _ := http.NewRequest("GET", url, nil)

	res, err := Client.Do(req)
	var response Response

	if err != nil {
		apiErrorCounter.WithLabelValues(labels...).Inc()
		log.Error().Err(err).Str("server", server.RuleURL).Msg("Can't query api")
		return response
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			apiErrorCounter.WithLabelValues(labels...).Inc()
			log.Error().Err(err).Str("server", server.RuleURL).Msg("Can't get rules, and api response body can't be read")
		} else {
			apiErrorCounter.WithLabelValues(labels...).Inc()
			log.Error().Str("server", server.RuleURL).Str("body", string(b)).Msg("Can't get rules")
		}
	} else {
		err := json.NewDecoder(res.Body).Decode(&response)

		if err != nil {
			apiErrorCounter.WithLabelValues(labels...).Inc()
			log.Error().Err(err).Str("server", server.RuleURL).Msg("Can't decode rules")
		} else if response.Status == "error" {
			apiErrorCounter.WithLabelValues(labels...).Inc()
			log.Error().Str("server", server.RuleURL).Str("error", response.Error).Str("errorType", response.ErrorType).Send()
		}
	}

	return response
}

func existingMetric(query, server Server) bool {
	labels := append([]string{"query", server.QueryURL}, server.LabelValues...)
	url := server.QueryURL + `/api/v1/query`
	payload := `query=` + query
	req, _ := http.NewRequest("POST", url, strings.NewReader(payload))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := Client.Do(req)

	var response Response

	if err != nil {
		apiErrorCounter.WithLabelValues(labels...).Inc()
		log.Error().Err(err).Str("server", server.QueryURL).Msg("Can't query api")
		return false
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			apiErrorCounter.WithLabelValues(labels...).Inc()
			log.Error().Err(err).Str("server", server.QueryURL).Msg("Can't get rules, and api response body can't be read")
		} else {
			apiErrorCounter.WithLabelValues(labels...).Inc()
			log.Error().Str("server", server.QueryURL).Str("body", string(b)).Msg("Can't get rules")
		}
	} else {
		err = json.NewDecoder(res.Body).Decode(&response)

		if err != nil {
			apiErrorCounter.WithLabelValues(labels...).Inc()
			log.Error().Err(err).Str("server", server.QueryURL).Msg("Can't decode api response")
			return false
		} else if response.Status == "error" {
			apiErrorCounter.WithLabelValues(labels...).Inc()
			log.Error().Str("server", server.QueryURL).Str("error", response.Error).Str("errorType", response.ErrorType).Send()
		}
	}

	return len(response.Data.Results) > 0
}
