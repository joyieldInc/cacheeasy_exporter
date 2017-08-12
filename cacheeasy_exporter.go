/*
 * cacheeasy_exporter - scrapes redis/predixy/machine stats and
 * exports for prometheus.
 * Copyright (C) 2017 Joyield, Inc. <joyield.com@gmail.com>
 * All rights reserved.
 */
package main

import (
	"flag"
	machine_exporter "github.com/joyieldInc/machine_exporter/exporter"
	predixy_exporter "github.com/joyieldInc/predixy_exporter/exporter"
	redis_exporter "github.com/joyieldInc/redis_exporter/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Bind    string   `yaml:"bind"`
	Redis   []string `yaml:"redis"`
	Predixy []string `yaml:"predixy"`
}

const (
	RedisType   = 0
	PredixyType = 1
)

type Exporter struct {
	addr     string
	name     string
	etype    int
	exporter interface{}
}

type CacheEasyCollector struct {
	mtx       sync.Mutex
	exporters map[string]Exporter
}

func loadConfig(configfile string) (*Config, error) {
	s, err := ioutil.ReadFile(configfile)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	err = yaml.Unmarshal(s, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *CacheEasyCollector) init(cfg *Config) error {
	e, err := machine_exporter.NewExporter()
	if err != nil {
		return err
	}
	prometheus.MustRegister(e)
	c.load(cfg)
	return nil
}

func (c *CacheEasyCollector) load(cfg *Config) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	addrs := map[string]bool{}
	for _, serv := range cfg.Redis {
		ss := strings.Fields(serv)
		if len(ss) != 2 {
			log.Printf("redis server \"%s\" invalid, ignore it", serv)
			continue
		}
		addrs[ss[0]] = true
		exp, ok := c.exporters[ss[0]]
		if ok {
			if exp.etype == RedisType && exp.name == ss[1] {
				continue
			}
			if exp.etype == RedisType {
				e := exp.exporter.(*redis_exporter.Exporter)
				prometheus.Unregister(e)
			} else if exp.etype == PredixyType {
				e := exp.exporter.(*predixy_exporter.Exporter)
				prometheus.Unregister(e)
			}
			log.Printf("del exporter %s %s\n", exp.addr, exp.name)
		}
		e, err := redis_exporter.NewExporter(ss[0], ss[1])
		if err != nil {
			return err
		}
		prometheus.MustRegister(e)
		c.exporters[ss[0]] = Exporter{
			addr:     ss[0],
			name:     ss[1],
			etype:    RedisType,
			exporter: e,
		}
		log.Printf("add redis exporter %s\n", serv)
	}
	for _, serv := range cfg.Predixy {
		ss := strings.Fields(serv)
		if len(ss) != 2 {
			log.Printf("predixy server \"%s\" invalid, ignore it", serv)
			continue
		}
		addrs[ss[0]] = true
		exp, ok := c.exporters[ss[0]]
		if ok {
			if exp.etype == PredixyType && exp.name == ss[1] {
				continue
			}
			if exp.etype == RedisType {
				e := exp.exporter.(*redis_exporter.Exporter)
				prometheus.Unregister(e)
			} else if exp.etype == PredixyType {
				e := exp.exporter.(*predixy_exporter.Exporter)
				prometheus.Unregister(e)
			}
			log.Printf("del exporter %s %s\n", exp.addr, exp.name)
		}
		e, err := predixy_exporter.NewExporter(ss[0], ss[1])
		if err != nil {
			return err
		}
		prometheus.MustRegister(e)
		c.exporters[ss[0]] = Exporter{
			addr:     ss[0],
			name:     ss[1],
			etype:    PredixyType,
			exporter: e,
		}
		log.Printf("add predixy exporter %s\n", serv)
	}
	for addr, _ := range c.exporters {
		_, ok := addrs[addr]
		if !ok {
			addrs[addr] = false
		}
	}
	for addr, ok := range addrs {
		if !ok {
			exp, _ := c.exporters[addr]
			if exp.etype == RedisType {
				e := exp.exporter.(*redis_exporter.Exporter)
				prometheus.Unregister(e)
			} else if exp.etype == PredixyType {
				e := exp.exporter.(*predixy_exporter.Exporter)
				prometheus.Unregister(e)
			}
			delete(c.exporters, addr)
			log.Printf("del exporter %s %s\n", exp.addr, exp.name)
		}
	}
	return nil
}

func refreshConfig(c *CacheEasyCollector, configfile string) {
	for true {
		time.Sleep(10 * time.Second)
		cfg, err := loadConfig(configfile)
		if err != nil {
			log.Printf("Refresh config file error:%v", err)
			continue
		}
		err = c.load(cfg)
		if err != nil {
			log.Printf("Refresh config load error:%v", err)
		}
	}
}

func main() {
	var (
		config = flag.String("config", "cacheeasy_exporter.yml", "Config file")
		bind   = flag.String("bind", "", "Listen address")
	)
	flag.Parse()
	cfg, err := loadConfig(*config)
	if err != nil {
		log.Fatal(err)
	}
	if len(*bind) > 0 {
		cfg.Bind = *bind
	}
	if len(cfg.Bind) == 0 {
		cfg.Bind = ":9123"
	}
	c := &CacheEasyCollector{
		exporters: make(map[string]Exporter),
	}
	err = c.init(cfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("CacheEasyExporter listen at %s\n", cfg.Bind)
	go refreshConfig(c, *config)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(cfg.Bind, nil))
}
