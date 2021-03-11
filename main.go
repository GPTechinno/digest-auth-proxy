package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

var user string
var pass string
var ip net.IP
var client *http.Client
var jar *Jar

type proxy struct {
}

func (p *proxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	log.Println("<", req.RemoteAddr, req.Method, req.URL)

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
	req2, err := http.NewRequest(req.Method, req.URL.String(), req.Body)
	log.Println(">", req2.Method, req2.URL)
	// TODO : copy some headers (maybe)
	resp, err := client.Do(req2)
	if err != nil {
		http.Error(wr, "Server Error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	log.Println("<", resp.Status)
	if resp.StatusCode == 401 {
		wa := newWwwAuthenticate(resp.Header.Get("WWW-Authenticate"))
		body, _ := ioutil.ReadAll(req.Body)
		auth, err := newAuthorization(wa, user, pass, req.RequestURI, req2.Method, string(body))
		if err != nil {
			http.Error(wr, "Server Error", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		req2.Header.Add("Authorization", auth.toString())
		log.Println(">", req2.Method, req2.URL)
		resp, err = client.Do(req2)
		if err != nil {
			http.Error(wr, "Server Error", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		log.Println("<", resp.Status)
	}
	defer resp.Body.Close()
	for k, vv := range resp.Header {
		for _, v := range vv {
			wr.Header().Add(k, v)
		}
	}
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
	log.Println(">", resp.Status)
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
	if *pIP != "" {
		ip = net.ParseIP(*pIP)
	} else {
		if os.Getenv("DAP_SERVER") != "" {
			ip = net.ParseIP(os.Getenv("DAP_SERVER"))
		}
	}
	if ip == nil {
		log.Fatal("Error parsing ip : ", *pIP)
	}
	if *pUser != "" {
		user = *pUser
	} else {
		if os.Getenv("DAP_USER") != "" {
			user = os.Getenv("DAP_USER")
		} else {
			log.Fatal("Error User cannot be empty")
		}
	}
	if *pPassword != "" {
		pass = *pPassword
	} else {
		if os.Getenv("DAP_PASS") != "" {
			pass = os.Getenv("DAP_PASS")
		} else {
			log.Fatal("Error Password cannot be empty")
		}
	}

	// Init Cookies Jar
	jar = NewJar()
	// Init HTTP Client
	client = &http.Client{
		Jar:     jar,
		Timeout: 5 * time.Second,
	}

	handler := &proxy{}
	log.Println("Starting proxy server on port ", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), handler); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
