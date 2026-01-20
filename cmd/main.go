package main

import (
	"flag"
	"os"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/config"
	"github.com/Amber-Gaze/OpsHub/pkg/logger"
)

func Exit(code int) {
	logger.Sync()
	time.Sleep(1 * time.Second)
	os.Exit(code)
}

func main() {
	var configFile string
	var showVersion bool
	flag.StringVar(&configFile, "c", "../conf/detector.yaml", "default config file")
	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.Parse()

	conf, err := config.LoadConfig(configFile)
	if err != nil {
		panic(err)
	}
	logger.InitLogger(conf.Logger)

	Exit(0)
}
