package cli

import (
	"os"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

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

var Conf Config

func LoadConf(confPath string) {
	yamlFile, err := os.ReadFile(confPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Can't read config file")
		os.Exit(1)
	}
	err = yaml.Unmarshal(yamlFile, &Conf)
	if err != nil {
		log.Fatal().Err(err).Msg("Unmarshal error")
		os.Exit(1)
	}
}
