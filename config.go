package main

type ConfigBlock struct {
	Listen struct {
		Host string
		Port string
	}
	Airbrake struct {
		Host string
		Protocol string
	}
}

var Config ConfigBlock
