package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(res http.ResponseWriter, req *http.Request)
}

type SimpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

func newServer(addr string) *SimpleServer {
	serverUrl, err := url.Parse(addr)
	handleErr(err)

	return &SimpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

type LoadBalancer struct {
	port            string
	roundRobinCount int
	servers         []Server
}

func newLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		roundRobinCount: 0,
		servers:         servers,
	}
}

func (s *SimpleServer) Address() string {
	return s.addr
}

func (s *SimpleServer) IsAlive() bool {
	return true
}

func (s *SimpleServer) Serve(res http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(res, req)
}

func (lb *LoadBalancer) GetNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]

	for !server.IsAlive() {
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	lb.roundRobinCount++
	return server
}

func (lb *LoadBalancer) ServeProxy(res http.ResponseWriter, req *http.Request) {
	destinationServer := lb.GetNextAvailableServer()
	fmt.Printf("Redirecting to %s\n", destinationServer.Address())
	destinationServer.Serve(res, req)

}

func handleErr(err error) {
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	servers := []Server{
		newServer("https://twitter.com"),
		newServer("https://facebook.com"),
		newServer("https://google.com"),
	}

	lb := newLoadBalancer(":8080", servers)

	handleRedirect := func(res http.ResponseWriter, req *http.Request) {
		lb.ServeProxy(res, req)
	}
	http.HandleFunc("/", handleRedirect)

	fmt.Printf("Load Balancer listening on port %s\n", lb.port)
	err := http.ListenAndServe(lb.port, nil)
	handleErr(err)
}
