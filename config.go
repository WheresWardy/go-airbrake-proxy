package main

import (
	"time"
	"github.com/peterbourgon/g2s"
)

type ConfigBlock struct {
	Listen struct {
		Host string
		Port string
	}
	Airbrake struct {
		Protocol string
		Host string
		Timeout time.Duration
	}
	Statsd struct {
		Protocol string
		Host string
		Port string
		Prefix string
	}
}

var Config ConfigBlock
var StatsD g2s.Statter
