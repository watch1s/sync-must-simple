package main

import (
	"bytes"
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

	h.hub.Broadcast(state)
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) HandlePeerState(w http.ResponseWriter, r *http.Request) {
	h.GetState(w, r)
}

func (h *Handlers) HandlePeerRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	cleanAddr := cleanPeerAddress(req.Address)
	
	// Check if already a peer
	peers, _ := h.db.ListPeers()
	exists := false
	for _, p := range peers {
		if cleanPeerAddress(p.Address) == cleanAddr {
			exists = true
			break
		}
	}
	
	if !exists {
		title := "New Peer Request"
		text := fmt.Sprintf("A new peer wants to connect:\n\nName: %s\nAddress: %s\n\nDo you want to accept this peer?", req.Name, cleanAddr)
		
		if PromptUser(title, text) {
			_ = h.db.AddPeer(cleanAddr, req.Name)
			w.WriteHeader(http.StatusOK)
		} else {
			http.Error(w, "Peer rejected by user", http.StatusForbidden)
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
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

		// Before adding to DB, send a peer request to the remote
		hostname, _ := os.Hostname()
		myAddr := fmt.Sprintf("%s:8788", getLocalIP())
		
		peerReqBody, _ := json.Marshal(map[string]string{
			"name":    hostname,
			"address": myAddr,
		})
		
		client := &http.Client{Timeout: 30 * time.Second} // Allow time for user to click
		resp, err := client.Post("http://"+cleanAddr+"/peer/request", "application/json", bytes.NewReader(peerReqBody))
		if err != nil || resp.StatusCode != http.StatusOK {
			http.Error(w, "Peer rejected the request or is unreachable", http.StatusForbidden)
			return
		}

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
