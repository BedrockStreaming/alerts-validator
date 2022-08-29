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

func getRules(server string) Response {
	url := server + `/api/v1/rules`
	req, _ := http.NewRequest("GET", url, nil)

	res, err := Client.Do(req)
	var response Response

	if err != nil {
		log.Error().Err(err)
		return response
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatal().Err(err).Str("server", server).Msg("Can't get rules, and api response body can't be read")
		} else {
			log.Fatal().Str("server", server).Str("body", string(b)).Msg("Can't get rules")
		}
	} else {
		err := json.NewDecoder(res.Body).Decode(&response)

		if err != nil {
			log.Fatal().Err(err).Str("server", server).Msg("Can't get rules")
		}
	}

	return response
}

func existingMetric(query, server string) bool {
	url := server + `/api/v1/query`
	payload := `query=` + query
	req, _ := http.NewRequest("POST", url, strings.NewReader(payload))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := Client.Do(req)

	var response Response

	if err != nil {
		log.Error().Err(err).Str("server", server).Msg("Can't query VM api")
		return false
	}
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&response)

	if err != nil {
		log.Fatal().Err(err).Str("server", server).Msg("Can't decode VM api response")
		return false
	}

	if response.Status == "error" {
		log.Fatal().Str("server", server).Str("error", response.Error).Str("errorType", response.ErrorType).Send()
	}

	return len(response.Data.Results) > 0
}
