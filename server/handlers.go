package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
				resp, err := client.Get("http://" + addr + "/peer/ping")
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
		if req.Address == "" {
			http.Error(w, "Address required", http.StatusBadRequest)
			return
		}
		if err := h.db.AddPeer(req.Address, req.Name); err != nil {
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
		if err := h.db.RemovePeer(req.Address); err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
