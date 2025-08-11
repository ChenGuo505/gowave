package config

import (
	"flag"
	"github.com/BurntSushi/toml"
	gwlog "github.com/ChenGuo505/gowave/log"
	"os"
)

var RootConfig = &GWConfig{}

type GWConfig struct {
	Log map[string]any `toml:"log"`
}

func init() {
	loadConfig()
}

func loadConfig() {
	confFile := flag.String("conf", "config/gowave.toml", "config file path")
	flag.Parse()
	if _, err := os.Stat(*confFile); err != nil {
		gwlog.DefaultLogger().Info("config file not found, using default")
		return
	}
	_, err := toml.DecodeFile(*confFile, RootConfig)
	if err != nil {
		gwlog.DefaultLogger().Info("config file decode error")
		return
	}
}
