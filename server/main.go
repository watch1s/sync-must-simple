package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

func main() {
	port := 8787

	db, err := InitDB("sync.db")
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}

	hub := NewWSHub()
	handlers := &Handlers{db: db, hub: hub}

	isPrivateIP := func(r *http.Request) bool {
		ipStr := r.RemoteAddr
		if colon := strings.LastIndex(ipStr, ":"); colon != -1 {
			ipStr = ipStr[:colon]
		}
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return false
		}
		if ip.IsLoopback() {
			return true
		}
		if ip4 := ip.To4(); ip4 != nil {
			return (ip4[0] == 10) || (ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) || (ip4[0] == 192 && ip4[1] == 168)
		}
		if len(ip) == net.IPv6len && ip[0]&0xfe == 0xfc {
			return true
		}
		return false
	}

	privateOnly := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !isPrivateIP(r) {
				http.Error(w, "Forbidden: Only LAN access is allowed", http.StatusForbidden)
				return
			}
			next(w, r)
		}
	}

	http.HandleFunc("/state", privateOnly(func(w http.ResponseWriter, r *http.Request) {
		// Enable CORS for development/LAN
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodGet {
			handlers.GetState(w, r)
		} else if r.Method == http.MethodPut {
			handlers.PutState(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	http.HandleFunc("/ws", privateOnly(hub.HandleWS))

	startServer := func() {
		addr := fmt.Sprintf("0.0.0.0:%d", port)
		fmt.Printf("Starting sync-must-simple server on %s...\n", addr)
		go func() {
			if err := http.ListenAndServe(addr, nil); err != nil {
				log.Fatalf("Server failed to start: %v", err)
			}
		}()
	}

	// RunApp handles blocking main thread and setting up systray (if tagged)
	RunApp(startServer)
}
