package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Handlers struct {
	db    *DB
	hub   *WSHub
	peers *PeerManager
}

func (h *Handlers) GetState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	states, err := h.db.GetAllStates()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if states == nil {
		states = []SyncState{}
	}
	json.NewEncoder(w).Encode(states)
}

func (h *Handlers) PutState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var state SyncState
	if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if state.Site == "" || state.URL == "" {
		http.Error(w, "site and url are required", http.StatusBadRequest)
		return
	}

	if err := h.db.SaveState(state); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	log.Printf("[Local Extension] Updated state for %s: scroll=%.2f%%", state.URL, state.ScrollPercent*100)

	h.hub.Broadcast(state)
	h.peers.SyncToAllPeers(state)

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) HandlePeerSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var state SyncState
	if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := h.db.SaveState(state); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	log.Printf("[Peer Sync In] Received update from %s for %s: scroll=%.2f%%", r.RemoteAddr, state.URL, state.ScrollPercent*100)

	// Auto-register sender as a peer if not already present
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" && host != "127.0.0.1" && host != "::1" {
		ip := net.ParseIP(host)
		if ip != nil {
			if ip4 := ip.To4(); ip4 != nil {
				host = ip4.String()
			}
		}
		peerAddr := host + ":8788"
		_ = h.db.AddPeer(peerAddr, "Auto-discovered Peer")
	}

	h.hub.Broadcast(state)
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) HandlePeerState(w http.ResponseWriter, r *http.Request) {
	h.GetState(w, r)
}

func (h *Handlers) HandlePeerPing(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	lanIP := getLocalIP()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      true,
		"name":    hostname,
		"address": fmt.Sprintf("%s:8788", lanIP),
	})
}

func (h *Handlers) HandlePeers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		peers, err := h.db.ListPeers()
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		type PeerStatus struct {
			PeerInfo
			Online bool `json:"online"`
		}

		statuses := make([]PeerStatus, len(peers))
		var wg sync.WaitGroup
		client := &http.Client{Timeout: 2 * time.Second}

		for i, p := range peers {
			statuses[i].PeerInfo = p
			wg.Add(1)
			go func(idx int, addr string) {
				defer wg.Done()
				cleanAddr := cleanPeerAddress(addr)
				resp, err := client.Get("http://" + cleanAddr + "/peer/ping")
				if err == nil {
					resp.Body.Close()
					statuses[idx].Online = true
				}
			}(i, p.Address)
		}
		wg.Wait()

		if statuses == nil {
			statuses = []PeerStatus{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(statuses)

	case http.MethodPost:
		var req struct {
			Address string `json:"address"`
			Name    string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		req.Address = strings.TrimSpace(req.Address)
		if req.Address == "" {
			http.Error(w, "Address required", http.StatusBadRequest)
			return
		}
		cleanAddr := cleanPeerAddress(req.Address)
		if err := h.db.AddPeer(cleanAddr, req.Name); err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		var req struct {
			Address string `json:"address"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		cleanAddr := cleanPeerAddress(req.Address)
		if err := h.db.RemovePeer(cleanAddr); err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
