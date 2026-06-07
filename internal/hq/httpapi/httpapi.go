package httpapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func StaticHandler(webRoot string) http.HandlerFunc {
	fileServer := http.FileServer(http.Dir(webRoot))

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		path := filepath.Clean(r.URL.Path)
		if path == "." || path == string(filepath.Separator) {
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		fullPath := filepath.Join(webRoot, strings.TrimPrefix(path, string(filepath.Separator)))
		if _, err := os.Stat(fullPath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, filepath.Join(webRoot, "index.html"))
	}
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write json response: %v", err)
	}
}

func LogRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func LocalIPv4Addresses() []string {
	var addresses []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return addresses
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		ifaceAddresses, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, ifaceAddress := range ifaceAddresses {
			ipNet, ok := ifaceAddress.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}

			addresses = append(addresses, fmt.Sprintf("%s", ip))
		}
	}

	return addresses
}
