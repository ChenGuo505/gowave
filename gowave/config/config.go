package config

import (
	"flag"
	"os"

	gwlog "github.com/ChenGuo505/gowave/log"
	"gopkg.in/yaml.v3"
)

var RootConfig = &GWConfig{}

type GWConfig struct {
	Http           HttpConfig           `yaml:"http"`
	Rpc            RpcConfig            `yaml:"rpc"`
	Log            LogConfig            `yaml:"log"`
	DataSource     DataSourceConfig     `yaml:"datasource"`
	RegisterCenter RegisterCenterConfig `yaml:"registerCenter"`
}

type HttpConfig struct {
	Name string `yaml:"name"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type RpcConfig struct {
	Name string `yaml:"name"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type LogConfig struct {
	Level string `yaml:"level"`
	Path  string `yaml:"path"`
}

type DataSourceConfig struct {
	Driver   string `yaml:"driver"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type RegisterCenterConfig struct {
	Type      string     `yaml:"type"`
	Endpoints []Endpoint `yaml:"endpoints"`
}

type Endpoint struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

func init() {
	loadConfig()
}

func loadConfig() {
	confFile := flag.String("conf", "config/gowave.yaml", "config file path")
	flag.Parse()
	if _, err := os.Stat(*confFile); err != nil {
		gwlog.DefaultLogger().Info("config file not found, using default")
		return
	}
	conf, err := os.ReadFile(*confFile)
	if err != nil {
		gwlog.DefaultLogger().Info("config file read error")
		return
	}
	if err := yaml.Unmarshal(conf, RootConfig); err != nil {
		gwlog.DefaultLogger().Info("config file unmarshal error")
		return
	}
}
