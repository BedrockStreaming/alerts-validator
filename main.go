package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/BedrockStreaming/alerts-validator/cli"
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

	cli.LoadConf(*confPath)
	listenAddress := cli.Conf.ListenAddress
	listenPort := cli.Conf.ListenPort

	// Init http client
	cli.Client = &http.Client{}

	// Build Gauges
	cli.BuildGauge()

	for _, server := range cli.Conf.Servers {
		go cli.CheckRules(server)
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
