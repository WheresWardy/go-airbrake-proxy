package main

import (
	"log"
	"time"
	"bytes"
	"net/http"
	"io/ioutil"
	"code.google.com/p/gcfg"
	"github.com/peterbourgon/g2s"
)

func main() {
	// Load configuration into package variable Config
	config_error := gcfg.ReadFileInto(&Config, "config/config.ini")
	if config_error != nil {
		log.Fatal("Could not load config file: " + config_error.Error())
	}

	// Instantiate StatsD connection
	if Config.Statsd.Host == "" {
		StatsD = g2s.Noop()
	} else {
		StatsD, _ = g2s.Dial(Config.Statsd.Protocol, Config.Statsd.Host + ":" + Config.Statsd.Port)
	}

	// Fire up an HTTP server and handle it
	http.HandleFunc("/", httpHandler)
	http.ListenAndServe(Config.Listen.Host + ":" + Config.Listen.Port, nil)
}

func httpHandler(response http.ResponseWriter, request *http.Request) {
	request_start := time.Now()

	// Proxy the request
	if request.Method == "POST" {
		body, _ := ioutil.ReadAll(request.Body)

		// Use a goroutine to make the Airbrake request
		go airbrakeRequest(request.RequestURI, body)
	}

	request_diff := time.Since(request_start);
	go StatsD.Timing(1.0, Config.Statsd.Prefix + ".http.request", request_diff)
}

func airbrakeRequest(requestURI string, body []byte) {
	response_start := time.Now()

	// Create the airbrake URL from configuration settings
	airbrake_url := Config.Airbrake.Protocol + "://" + Config.Airbrake.Host + requestURI

	// Create an HTTP client and make request to Airbrake
	airbrake_client := &http.Client{}
	airbrake_request, _ := http.NewRequest("POST", airbrake_url, bytes.NewReader(body))

	// Set headers and send request to Airbrake
	airbrake_request.Header.Set("Content-Type", "text/xml")
	airbrake_request.Header.Set("Connection", "close")

	// Deal with Airbrake response
	airbrake_client.Do(airbrake_request)

	response_diff := time.Since(response_start)
	go StatsD.Timing(1.0, Config.Statsd.Prefix + ".airbrake.request", response_diff)
}
