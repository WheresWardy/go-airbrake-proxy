package main

import (
	"bytes"
	"net/http"
	"io/ioutil"
	"code.google.com/p/gcfg"
)

func main() {
	// Load configuration into package variable Config
	gcfg.ReadFileInto(&Config, "config/config.ini")	

	// Fire up an HTTP server and handle it
	http.HandleFunc("/", httpHandler)
	http.ListenAndServe(Config.Listen.Host + ":" + Config.Listen.Port, nil)
}

func httpHandler(response http.ResponseWriter, request *http.Request) {
	// Proxy the request
	if request.Method == "POST" {
		body, _ := ioutil.ReadAll(request.Body)

		// Hijack so as to force close the connection
		hijack, _ := response.(http.Hijacker)
		connection, _, _ := hijack.Hijack()
		defer connection.Close()

		// Use a goroutine to make the Airbrake request
		go airbrakeRequest(request.RequestURI, body)
	}
}


func airbrakeRequest(requestURI string, body []byte) {
	// Create the airbrake URL from configuration settings
	airbrake_url := Config.Airbrake.Protocol + "://" + Config.Airbrake.Host + requestURI

	// Create an HTTP client and make request to Airbrake
	airbrake_client := &http.Client{}
	airbrake_request, _ := http.NewRequest("POST", airbrake_url, bytes.NewReader(body))

	// Set headers and send request to Airbrake
	airbrake_request.Header.Set("Content-Type", "text/xml")
	airbrake_request.Header.Set("Connection", "close")
	airbrake_client.Do(airbrake_request)
}
