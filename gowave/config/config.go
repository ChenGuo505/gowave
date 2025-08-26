package config

import (
	"flag"
	"github.com/BurntSushi/toml"
	gwlog "github.com/ChenGuo505/gowave/log"
	"os"
)

var RootConfig = &GWConfig{}

type GWConfig struct {
	Server     map[string]any `toml:"server"`
	Log        map[string]any `toml:"log"`
	DataSource map[string]any `toml:"datasource"`
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
