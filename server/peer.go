package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type PeerManager struct {
	db  *DB
	hub *WSHub
}

func NewPeerManager(db *DB, hub *WSHub) *PeerManager {
	return &PeerManager{db: db, hub: hub}
}

func cleanPeerAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	addr = strings.TrimPrefix(addr, "http://")
	addr = strings.TrimPrefix(addr, "https://")
	addr = strings.TrimRight(addr, "/")
	if !strings.Contains(addr, ":") {
		addr = addr + ":8788"
	}
	return addr
}

func (p *PeerManager) SyncToAllPeers(state SyncState) {
	go func() {
		peers, err := p.db.ListPeers()
		if err != nil {
			log.Printf("Failed to list peers for sync: %v", err)
			return
		}

		client := &http.Client{Timeout: 5 * time.Second}
		body, _ := json.Marshal(state)

		for _, peer := range peers {
			go func(addr string) {
				cleanAddr := cleanPeerAddress(addr)
				url := "http://" + cleanAddr + "/peer/sync"
				req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
				if err != nil {
					log.Printf("Failed to create request for peer %s: %v", cleanAddr, err)
					return
				}
				req.Header.Set("Content-Type", "application/json")
				resp, err := client.Do(req)
				if err != nil {
					log.Printf("Failed to sync to peer %s: %v", cleanAddr, err)
					return
				}
				resp.Body.Close()
				log.Printf("Successfully synced state to peer %s", cleanAddr)
			}(peer.Address)
		}
	}()
}

func (p *PeerManager) PullFromAllPeers() {
	peers, err := p.db.ListPeers()
	if err != nil {
		log.Printf("Failed to list peers for pull: %v", err)
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, peer := range peers {
		go func(addr string) {
			cleanAddr := cleanPeerAddress(addr)
			url := "http://" + cleanAddr + "/peer/state"
			resp, err := client.Get(url)
			if err != nil {
				log.Printf("Failed to pull from peer %s: %v", cleanAddr, err)
				return
			}
			defer resp.Body.Close()

			var states []SyncState
			if err := json.NewDecoder(resp.Body).Decode(&states); err != nil {
				log.Printf("Failed to decode states from peer %s: %v", cleanAddr, err)
				return
			}

			for _, state := range states {
				_ = p.db.SaveState(state)
			}
			log.Printf("Successfully pulled %d states from peer %s", len(states), cleanAddr)
		}(peer.Address)
	}
}
