package main

import (
	"encoding/json"
	"net/http"
)

type Handlers struct {
	db  *DB
	hub *WSHub
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

	// Broadcast the update to all connected WS clients
	h.hub.Broadcast(state)

	w.WriteHeader(http.StatusOK)
}
