package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

type LoadBalancer struct {
	port			string
	servers			[]Server
	roundRobinCount	int
}

func newSimpleServer(addr string) *simpleServer {
	serverURL, err := url.Parse(addr)
	handleErr(err)

	return &simpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverURL),
	}
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		servers:         servers,
		roundRobinCount: 0,
	}
}

func (s *simpleServer) Serve(rw http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(rw, r)
}

func (s *simpleServer) Address() string {
	return s.addr
}

func (s *simpleServer) IsAlive() bool {
	return true
}

func (lb *LoadBalancer) nextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	for !server.IsAlive() {
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	lb.roundRobinCount++
	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, r *http.Request) {
	targerServer := lb.nextAvailableServer()
	fmt.Printf("forwarding request to addr: %q\n", targerServer.Address())
	targerServer.Serve(rw, r)
}

func main() {
	serverList := []Server{
		newSimpleServer("https://dzen.ru"),
		newSimpleServer("http://ya.ru"),
		newSimpleServer("https://mail.ru"),
	}

	loadBalancer := NewLoadBalancer("8080", serverList)

	handleRedirect := func(rw http.ResponseWriter, r *http.Request) {
		loadBalancer.serveProxy(rw, r)
	}
	http.HandleFunc("/", handleRedirect)
	fmt.Printf("Serving requests on 'localhost: %s'\n", loadBalancer.port)
	http.ListenAndServe("0.0.0.0:" + loadBalancer.port, nil)
}

func handleErr(err error) {
	if err != nil {
		fmt.Printf("error: %v", err)
		// os.Exit(1)
	}
}
