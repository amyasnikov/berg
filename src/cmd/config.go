package main

import (
	"github.com/osrg/gobgp/v3/pkg/config"
	"github.com/osrg/gobgp/v3/pkg/config/oc"
	flag "github.com/spf13/pflag"
)

type Config struct {
	GobgpConfig *oc.BgpConfigSet
	ConfigFile  string
	GrpcHosts   string
	LogLevel    string
}

func NewConfig() (cfg Config) {
	configFile := flag.StringP("config", "f", "", "Path to TOML config file")
	grpcHosts := flag.StringP("api-host", "a", ":50051", "gRPC API address:port to listen to.")
	logLevel := flag.StringP("log-level", "l", "info", "Log Level")

	flag.Parse()
	if *configFile == "" {
		panic("config file must be defined")
	}
	gobgpConfig, err := config.ReadConfigFile(*configFile, "toml")
	if err != nil {
		panic(err)
	}
	cfg.GobgpConfig = gobgpConfig
	cfg.ConfigFile = *configFile
	cfg.LogLevel = *logLevel
	cfg.GrpcHosts = *grpcHosts
	return
}
