package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/VictoriaMetrics/metricsql"
)

var vectors []string

func CheckRules(server Server) {
	for {
		vectorValidityCache := map[string]bool{}
		response := getRules(server)
		for _, group := range response.Data.Groups {
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
				for key, interval := range Conf.ValidityCheckIntervals {
					if key >= len(Conf.ValidityCheckIntervals)-1 {
						continue
					}

					zDictVector := zerolog.Dict()
					from := interval
					to := Conf.ValidityCheckIntervals[key+1]
					valids := checkValidity(rule, from, to, vectorValidityCache, server)

					labelsValid := append([]string{rule.Name, rule.Id, from, to, "valid"}, server.LabelValues...)
					labelsInvalid := append([]string{rule.Name, rule.Id, from, to, "invalid"}, server.LabelValues...)

					valid := true
					for k, v := range valids {
						if !v {
							valid = false
						}
						zDictVector.Bool(k, v)
					}

					if !valid {
						gauge.WithLabelValues(labelsValid...).Set(0)
						gauge.WithLabelValues(labelsInvalid...).Set(1)
						zDictValid.Bool(fmt.Sprintf("%s-%s", from, to), false)
					} else {
						gauge.WithLabelValues(labelsValid...).Set(1)
						gauge.WithLabelValues(labelsInvalid...).Set(0)
						zDictValid.Bool(fmt.Sprintf("%s-%s", from, to), true)
					}
					zDictVectorPresent.Dict(fmt.Sprintf("%s-%s", from, to), zDictVector)
				}
				log.Info().Str("alertname", rule.Name).Str("id", rule.Id).Dict("is_vector_present", zDictVectorPresent).Dict("is_valid", zDictValid).Send()
			}
		}
		dur, err := time.ParseDuration(Conf.ComputeInterval)
		if err != nil {
			log.Fatal().Err(err).Msg("Can't parse Compute Interval")
			os.Exit(0)
		}
		time.Sleep(dur)
	}
}

func checkValidity(rule Rule, from, to string, cache map[string]bool, server Server) map[string]bool {
	checkedVector := []string{}
	valids := map[string]bool{}
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
		if val, ok := cache[fmt.Sprintf("%s[%s-%s]", vector, from, to)]; ok {
			log.Debug().Str("alertname", rule.Name).Str("id", rule.Id).Str("vector", vector).Bool("incache", true).Send()
			existingM = val
		} else {
			log.Debug().Str("alertname", rule.Name).Str("id", rule.Id).Str("vector", vector).Bool("incache", false).Send()
			dur1, err := time.ParseDuration(to)
			if err != nil {
				log.Error().Err(err).Msg("Something is wrong with Validity Check Intervals")
				os.Exit(1)
			}
			dur2, err := time.ParseDuration(from)
			if err != nil {
				log.Error().Err(err).Msg("Something is wrong with Validity Check Intervals")
				os.Exit(1)
			}
			dur := time.Until(time.Now().Add(dur1).Add(-dur2)).Truncate(time.Minute)
			check := fmt.Sprintf("present_over_time(%s[%s] offset %s)", vector, dur.String(), from)
			_, err = metricsql.Parse(check)
			if err != nil {
				log.Error().Err(err).Str("check", check).Msg("Something is wrong with Validity Check Intervals")
				os.Exit(1)
			}
			log.Debug().Str("alertname", rule.Name).Str("id", rule.Id).Str("check", check).Send()
			existingM = existingMetric(check, server)
			// Add vector in cache to avoid checking multiple times the same thing
			cache[fmt.Sprintf("%s[%s-%s]", vector, from, to)] = existingM
		}
		valids[vector] = existingM
	}
	return valids
}

func tree(expr metricsql.Expr) {
	switch e := expr.(type) {
	case *metricsql.MetricExpr:
		vectors = append(vectors, string(e.AppendString(nil)))
	}
}
