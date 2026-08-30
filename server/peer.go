package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type PeerManager struct {
	db  *DB
	hub *WSHub
}

func NewPeerManager(db *DB, hub *WSHub) *PeerManager {
	return &PeerManager{db: db, hub: hub}
}

func (p *PeerManager) SyncToAllPeers(state SyncState) {
	go func() {
		peers, err := p.db.ListPeers()
		if err != nil {
			log.Printf("Failed to list peers for sync: %v", err)
			return
		}

		client := &http.Client{Timeout: 3 * time.Second}
		body, _ := json.Marshal(state)

		for _, peer := range peers {
			go func(addr string) {
				req, err := http.NewRequest(http.MethodPut, "http://"+addr+"/peer/sync", bytes.NewReader(body))
				if err != nil {
					return
				}
				req.Header.Set("Content-Type", "application/json")
				resp, err := client.Do(req)
				if err != nil {
					log.Printf("Failed to sync to peer %s: %v", addr, err)
					return
				}
				resp.Body.Close()
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

	client := &http.Client{Timeout: 3 * time.Second}

	for _, peer := range peers {
		go func(addr string) {
			resp, err := client.Get("http://" + addr + "/peer/state")
			if err != nil {
				log.Printf("Failed to pull from peer %s: %v", addr, err)
				return
			}
			defer resp.Body.Close()

			var states []SyncState
			if err := json.NewDecoder(resp.Body).Decode(&states); err != nil {
				return
			}

			for _, state := range states {
				_ = p.db.SaveState(state)
			}
		}(peer.Address)
	}
}
