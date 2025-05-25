package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/amyasnikov/berg/internal/app"
	"github.com/osrg/gobgp/v3/pkg/config"
	"github.com/osrg/gobgp/v3/pkg/log"
	"github.com/osrg/gobgp/v3/pkg/server"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	var logger = logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.JSONFormatter{})
	opts := NewConfig()
	switch opts.LogLevel {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}
	maxSize := 256 << 20
	grpcOpts := []grpc.ServerOption{grpc.MaxRecvMsgSize(maxSize), grpc.MaxSendMsgSize(maxSize)}
	logger.Info("berg started")
	bgpLogger := log.NewDefaultLogger()
	bgpServer := server.NewBgpServer(
		server.GrpcListenAddress(opts.GrpcHosts),
		server.GrpcOption(grpcOpts),
		server.LoggerOption(bgpLogger))
	bufSize := 100000
	berg := app.NewApp(opts.GobgpConfig, bgpServer, uint64(bufSize), logger)
	ctx, stopBerg := context.WithCancel(context.Background())
	go bgpServer.Serve()
	_, err := config.InitialConfig(context.Background(), bgpServer, opts.GobgpConfig, false)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"Topic": "Config",
			"Error": err,
		}).Fatalf("Failed to apply initial configuration %s", opts.ConfigFile)
	}

	go berg.Serve(ctx)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	sig := <-sigCh
	logger.Info("Received %s — shutting down.", sig)
	stopBerg()
	bgpServer.Stop()
}
