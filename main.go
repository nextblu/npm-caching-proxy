package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)


func downloadFile(filepath string, url string) {
	// Create the file
	cleanedPath := strings.Split(filepath, "/")
	filepath = "./cache/" + cleanedPath[1] +"/"+ cleanedPath[2] + "/" + cleanedPath[3]
	url = "http://registry.npmjs.org" + url
	log.Printf("Starting file download in the following directory: %s\n", filepath)
	log.Printf("Starting file download from the following url: %s\n", url)
	_ = os.MkdirAll("./cache/" + cleanedPath[1] +"/"+ cleanedPath[2] + "/", os.ModePerm)
	out, err := os.Create(filepath)
	if err != nil  {
		fmt.Printf("Out file error %s\n", err)
	}
	defer out.Close()
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	// Check server response
	if resp.StatusCode != http.StatusOK {
		fmt.Println(err)
	}
	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil  {
		fmt.Println(err)
	}
}

// Get the port to listen on
func getListenAddress() string {
	port := "1338"
	return ":" + port
}

// Get the url for a given proxy condition
func getProxyUrl(filename string) string {
	proxyCondition := filename
	internalHandler := "http://localhost:3000"
	defaultOrigin := "http://registry.npmjs.org"
	// Checking if file is already been cached
	log.Printf("Required file: %s\n", proxyCondition)
	if strings.Contains(filename, ".tgz") {
		if _, err := os.Stat("./cache/"+proxyCondition); err == nil {
			// path/to/whatever exists
			log.Printf("The required file is actually cached, serving\n")
			return internalHandler
		} else if os.IsNotExist(err) {
			// path/to/whatever does *not* exist
			log.Printf("The file is not cached, returning origin and starting cache collection\n")
			go downloadFile(proxyCondition, filename)
			return defaultOrigin
		} else {
			// Schrodinger: file may or may not exist. See err for details
			// Therefore, do *NOT* use !os.IsNotExist(err) to test for file existence
			log.Printf("The file is not cached, returning origin and starting cache collection\n")
			return defaultOrigin
		}
	}
	log.Printf("The requied url is not a file, returning origin\n")
	return defaultOrigin
}

/*
	Logging
*/

// Log the typeform payload and redirect url
func logRequestPayload(proxyUrl string) {
	log.Printf("proxy_url: %s\n", proxyUrl)
}

// Log the env variables required for a reverse proxy
func logSetup() {
	internalHandler := "http://localhost:3000"
	defaultOrigin := "http://registry.npmjs.org"

	log.Printf("Server will run on: %s\n", getListenAddress())
	log.Printf("Redirecting to internal url: %s\n", internalHandler)
	log.Printf("Redirecting to Default url: %s\n", defaultOrigin)
}

/*
	Reverse Proxy Logic
*/

// Serve a reverse proxy for a given url
func serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	// parse the OriginalUrl
	OriginalUrl, _ := url.Parse(target)

	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(OriginalUrl)

	// Update the headers to allow for SSL redirection
	req.URL.Host = OriginalUrl.Host
	req.URL.Scheme = OriginalUrl.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = OriginalUrl.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}


// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	proxyUrl := getProxyUrl(req.URL.Path)
	log.Printf("Received request for url: %v", proxyUrl)
	logRequestPayload(proxyUrl)
	serveReverseProxy(proxyUrl, res, req)
}

// Internal http server, to be defined
func internalHTTPServer(){
	fs := http.FileServer(http.Dir("./cache"))
	http.Handle("./cache/", http.StripPrefix("./cache/", fs))

	log.Println("Listening on :3000...")
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// Starting the http server on separate routine
	go internalHTTPServer()

	// Log setup values
	logSetup()

	// start server
	http.HandleFunc("/", handleRequestAndRedirect)
	if err := http.ListenAndServe(getListenAddress(), nil); err != nil {
		panic(err)
	}
}

