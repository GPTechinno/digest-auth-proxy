package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"

	dac "github.com/xinsnake/go-http-digest-auth-client"
)

var dr dac.DigestRequest
var user string
var pass string
var ip net.IP

type proxy struct {
}

func (p *proxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	log.Println(req.RemoteAddr, " ", req.Method, " ", req.URL)

	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}
	req.URL.Host = ip.String()
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		msg := "unsupported protocol scheme " + req.URL.Scheme
		http.Error(wr, msg, http.StatusBadRequest)
		log.Println(msg)
		return
	}

	defer req.Body.Close()
	body, _ := ioutil.ReadAll(req.Body)
	// update digest authentication request
	dr.UpdateRequest(user, pass, req.Method, req.URL.String(), string(body))
	resp, err := dr.Execute()
	if err != nil {
		http.Error(wr, "Server Error", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	log.Println(req.RemoteAddr, " ", resp.Status)
	for k, vv := range resp.Header {
		for _, v := range vv {
			wr.Header().Add(k, v)
		}
	}
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}

func main() {
	// Parse args
	pIP := flag.String("ip", "", "Server IP Address")
	pUser := flag.String("user", "", "Digest Auth User")
	pPassword := flag.String("pass", "", "Digest Auth Password")
	var port int
	flag.IntVar(&port, "port", 9999, "Proxy TCP Port")
	flag.Parse()

	// Check argument validity
	ip = net.ParseIP(*pIP)
	if ip == nil {
		log.Fatal("Error parsing ip : ", *pIP)
	}
	if *pUser == "" {
		log.Fatal("Error User cannot be empty")
	}
	if *pPassword == "" {
		log.Fatal("Error Password cannot be empty")
	}

	user = *pUser
	pass = *pPassword
	dr.CertVal = true

	handler := &proxy{}
	log.Println("Starting proxy server on port ", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), handler); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
