package main

import (
	"log"
	"flag"
	"time"
	"bytes"
	"net/http"
	"io/ioutil"
	"encoding/xml"
	"code.google.com/p/gcfg"
	"github.com/peterbourgon/g2s"
	"github.com/mreiferson/go-httpclient"
)

func main() {
	// Parse command line arguments
	var (
		config_file = flag.String("config", "", "Path to configuration file")
	)
	flag.Parse()

	// Load configuration into package variable Config
	config_error := gcfg.ReadFileInto(&Config, *config_file)
	if config_error != nil {
		log.Fatal("Could not load config file: " + config_error.Error())
	}

	// Instantiate StatsD connection
	if Config.Statsd.Host == "" {
		StatsD = g2s.Noop()
	} else {
		StatsD, _ = g2s.Dial(Config.Statsd.Protocol, Config.Statsd.Host + ":" + Config.Statsd.Port)
	}

	// Log startup
	log.Println("go-airbrake-proxy started")

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

	// Modify the HTTP transport with timeouts
	transport := &httpclient.Transport {
		ConnectTimeout: Config.Airbrake.Timeout * time.Second,
		RequestTimeout: Config.Airbrake.Timeout * time.Second,
		ResponseHeaderTimeout: Config.Airbrake.Timeout * time.Second,
	}
	defer transport.Close()

	// Create an HTTP client and make request to Airbrake
	airbrake_client := &http.Client {
		Transport: transport,
	}
	airbrake_request, _ := http.NewRequest("POST", airbrake_url, bytes.NewReader(body))

	// Set headers and send request to Airbrake
	airbrake_request.Header.Set("Content-Type", "text/xml")
	airbrake_request.Header.Set("Connection", "close")

	// Make request to Airbrake
	airbrake_response, airbrake_response_error := airbrake_client.Do(airbrake_request)

	// Record request timing in StatsD
	response_diff := time.Since(response_start)
	go StatsD.Timing(1.0, Config.Statsd.Prefix + ".airbrake.request", response_diff)

	// Decode response and record stats
	if airbrake_response_error != nil {
		go StatsD.Counter(1.0, Config.Statsd.Prefix + ".airbrake.request.fail.timeout", 1)
	} else {
		defer airbrake_response.Body.Close()
		if airbrake_response.StatusCode == 200 {
			airbrakeXML(ioutil.ReadAll(airbrake_response.Body))
		} else {
			go StatsD.Counter(1.0, Config.Statsd.Prefix + ".airbrake.request.fail.error", 1)
		}
	}
}

func airbrakeXML(xml_body []byte, xml_error error) {
	// Validate XML response from Airbrake (should have valid notice ID)
	var notice Notice
	xml.Unmarshal(xml_body, &notice)

	if notice.Id != 0 {
		go StatsD.Counter(1.0, Config.Statsd.Prefix + ".airbrake.request.success", 1)
	} else {
		go StatsD.Counter(1.0, Config.Statsd.Prefix + ".airbrake.request.fail.xml", 1)
	}
}
