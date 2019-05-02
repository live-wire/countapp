// config reader

package utils

import (
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "log"
    "time"
)

type conf struct {
    Workers []string `yaml:"workers"`
    Database string `yaml:"database"`
    WorkerLogs string `yaml:"worker_logs"`
    WorkerPersist float64 `yaml:"worker_persist"`
    ConfigCheck float64 `yaml:"config_check"`
}

// current fetched config
var currentConfig conf

// var configLastFetched int64
var lastFetched time.Time

// Reads the yaml config file
func (c *conf) getConf() *conf {
    // fmt.Println("Fetching settings from disk")
    yamlFile, err := ioutil.ReadFile("config.yaml")
    if err != nil {
        log.Printf("yamlFile.Get err   #%v ", err)
    }
    err = yaml.Unmarshal(yamlFile, c)
    if err != nil {
        log.Fatalf("Unmarshal: %v", err)
    }
    lastFetched = time.Now()
    return c
}

// Fetches the config file again if the time elapsed exceeds yaml config_check
func Config() *conf {
    elapsed := time.Now().Sub(lastFetched)
    if currentConfig.Database == "" || currentConfig.ConfigCheck <= elapsed.Seconds() {
        currentConfig.getConf()
    }
    return &currentConfig
}
