package main

import (
	"fmt"
	"os"
	"time"

	"github.com/osrg/gobgp/v3/pkg/config"
	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"github.com/pelletier/go-toml/v2"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"golang.org/x/time/rate"

)

type Config struct {
	GobgpConfig *oc.BgpConfigSet
	ConfigFile  string
	GrpcHosts   string
	LogLevel    string
	logger *logrus.Logger
}

func NewConfig(logger *logrus.Logger) (cfg Config) {
	configFile := flag.StringP("config", "f", "", "Path to TOML config file")
	grpcHosts := flag.StringP("api-host", "a", ":50051", "gRPC API address:port to listen to.")
	logLevel := flag.StringP("log-level", "l", "info", "Log Level")

	flag.Parse()
	if *configFile == "" {
		panic("config file must be defined")
	}
	cfg.ConfigFile = *configFile
	cfg.LogLevel = *logLevel
	cfg.GrpcHosts = *grpcHosts
	cfg.logger = logger
	cfg.GobgpConfig = cfg.mustReadConfig()
	return
}

func (c *Config) mustReadConfig() *oc.BgpConfigSet {
	ensureVrfIdDefined(c.ConfigFile)
	gobgpConfig, err := config.ReadConfigFile(c.ConfigFile, "toml")
	if err != nil {
		c.logger.Fatalf("error reading config file: %w", err)
	}
	return gobgpConfig
}


func (c *Config)watchConfigChanges() <- chan *oc.BgpConfigSet {
	ch := make(chan *oc.BgpConfigSet)
	rateLimiter := rate.Sometimes{Interval: 1 * time.Second}
	config.WatchConfigFile(c.ConfigFile, "toml", func() {
		rateLimiter.Do(func() {
			c.logger.Info("Config changes detected, reloading configuration")
			newConfig := c.mustReadConfig()
			ch <- newConfig
		})
	})
	return ch
}


func ensureVrfIdDefined(fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	config := oc.BgpConfigSet{}
	err = toml.NewDecoder(file).Decode(&config)
	if err != nil {
		return err
	}
	erroredVrfs := []string{}
	for _, vrf := range config.Vrfs {
		if vrf.Config.Id == 0 {
			erroredVrfs = append(erroredVrfs, vrf.Config.Name)
		}
	}
	if len(erroredVrfs) > 0 {
		return fmt.Errorf("ID is mandatory for a VRF. The following VRFs have no ID: %v", erroredVrfs)
	}
	return nil
}