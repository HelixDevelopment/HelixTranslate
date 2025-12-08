package utils

import (
	"net"
	"strconv"
	"sync"
)

var (
	portMutex   sync.Mutex
	usedPorts   = make(map[int]bool)
	nextPort    = 30000 // Start from high port range
)

// GetFreePort returns a free port number that can be used for testing
func GetFreePort() int {
	portMutex.Lock()
	defer portMutex.Unlock()

	// Find a free port
	for i := 0; i < 100; i++ {
		port := nextPort
		nextPort++
		
		if !isPortUsed(port) {
			if isPortAvailable(port) {
				usedPorts[port] = true
				return port
			}
		}
	}
	
	// If we can't find a port in our range, use system-assigned port
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	
	port := l.Addr().(*net.TCPAddr).Port
	usedPorts[port] = true
	return port
}

// ReleasePort marks a port as no longer in use
func ReleasePort(port int) {
	portMutex.Lock()
	defer portMutex.Unlock()
	delete(usedPorts, port)
}

// isPortUsed checks if a port is already marked as used
func isPortUsed(port int) bool {
	return usedPorts[port]
}

// isPortAvailable checks if a port is actually available for listening
func isPortAvailable(port int) bool {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return false
	}
	defer l.Close()
	
	return true
}

// GetLocalhostURL returns a localhost URL with the given port
func GetLocalhostURL(port int, scheme string) string {
	if scheme == "" {
		scheme = "http"
	}
	return scheme + "://localhost:" + strconv.Itoa(port)
}

// GetLocalhostWSURL returns a WebSocket URL for localhost
func GetLocalhostWSURL(port int, path string) string {
	if path == "" {
		path = "/ws"
	}
	if path[0] != '/' {
		path = "/" + path
	}
	return "ws://localhost:" + strconv.Itoa(port) + path
}