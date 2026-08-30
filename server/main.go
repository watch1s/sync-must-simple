package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
)

func main() {
	localPort := 8787
	lanPort := 8788

	db, err := InitDB("sync.db")
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}

	hub := NewWSHub()
	peers := NewPeerManager(db, hub)
	handlers := &Handlers{db: db, hub: hub, peers: peers}

	// Startup initial pull
	go peers.PullFromAllPeers()

	isPrivateIP := func(r *http.Request) bool {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		ip := net.ParseIP(host)
		if ip == nil {
			return false
		}
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return true
		}
		if ip4 := ip.To4(); ip4 != nil {
			return (ip4[0] == 10) || (ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) || (ip4[0] == 192 && ip4[1] == 168) || (ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127)
		}
		if len(ip) == net.IPv6len && (ip[0]&0xfe == 0xfc || ip[0] == 0xfe && (ip[1]&0xc0) == 0x80) {
			return true
		}
		return false
	}

	corsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Access-Control-Request-Private-Network")
			w.Header().Set("Access-Control-Allow-Private-Network", "true")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next(w, r)
		}
	}

	lanMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !isPrivateIP(r) {
				log.Printf("Forbidden: Rejected non-LAN connection from %s", r.RemoteAddr)
				http.Error(w, "Forbidden: Only LAN access is allowed", http.StatusForbidden)
				return
			}
			next(w, r)
		}
	}

	// 1. localMux (127.0.0.1:8787)
	localMux := http.NewServeMux()
	localMux.HandleFunc("/state", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.GetState(w, r)
		} else if r.Method == http.MethodPut {
			handlers.PutState(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	localMux.HandleFunc("/ws", corsMiddleware(hub.HandleWS))

	// 2. lanMux (0.0.0.0:8788)
	lanMux := http.NewServeMux()
	lanMux.HandleFunc("/peer/sync", lanMiddleware(handlers.HandlePeerSync))
	lanMux.HandleFunc("/peer/state", lanMiddleware(handlers.HandlePeerState))
	lanMux.HandleFunc("/peer/request", lanMiddleware(handlers.HandlePeerRequest))
	lanMux.HandleFunc("/peer/ping", lanMiddleware(handlers.HandlePeerPing))
	lanMux.HandleFunc("/peers", lanMiddleware(handlers.HandlePeers))
	lanMux.HandleFunc("/manage", lanMiddleware(HandleManage))

	startServer := func() {
		localAddr := fmt.Sprintf("127.0.0.1:%d", localPort)
		lanAddr := fmt.Sprintf("0.0.0.0:%d", lanPort)
		
		localIP := getLocalIP()
		
		fmt.Printf("\n==============================================\n")
		fmt.Printf("sync-must-simple server started!\n")
		fmt.Printf("   Extension Address: http://127.0.0.1:%d/state\n", localPort)
		fmt.Printf("   Peer API Address:  http://%s:%d\n", localIP, lanPort)
		fmt.Printf("   Manage Peers:      http://127.0.0.1:%d/manage\n", lanPort)
		fmt.Printf("==============================================\n\n")
		
		go func() {
			if err := http.ListenAndServe(localAddr, localMux); err != nil {
				log.Fatalf("Local server failed to start: %v", err)
			}
		}()
		
		go func() {
			if err := http.ListenAndServe(lanAddr, lanMux); err != nil {
				log.Fatalf("LAN server failed to start: %v", err)
			}
		}()
	}

	RunApp(startServer)
}
