package main

import (
	"database/sql"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	mu  sync.Mutex
	sql *sql.DB
}

func InitDB(filepath string) (*DB, error) {
	db, err := sql.Open("sqlite", filepath)
	if err != nil {
		return nil, err
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS sync_state (
		site TEXT PRIMARY KEY,
		url TEXT,
		scroll_percent REAL,
		updated_at INTEGER,
		device_id TEXT
	);
	CREATE TABLE IF NOT EXISTS peers (
		address TEXT PRIMARY KEY,
		name TEXT,
		added_at INTEGER
	);
	`
	if _, err := db.Exec(createTableQuery); err != nil {
		return nil, err
	}

	return &DB{sql: db}, nil
}

func (db *DB) SaveState(state SyncState) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	query := `
	INSERT INTO sync_state (site, url, scroll_percent, updated_at, device_id)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(site) DO UPDATE SET
		url = excluded.url,
		scroll_percent = excluded.scroll_percent,
		updated_at = excluded.updated_at,
		device_id = excluded.device_id
	WHERE excluded.updated_at > sync_state.updated_at;
	`
	_, err := db.sql.Exec(query, state.Site, state.URL, state.ScrollPercent, state.UpdatedAt, state.DeviceID)
	return err
}

func (db *DB) GetAllStates() ([]SyncState, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	rows, err := db.sql.Query("SELECT site, url, scroll_percent, updated_at, device_id FROM sync_state")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []SyncState
	for rows.Next() {
		var s SyncState
		if err := rows.Scan(&s.Site, &s.URL, &s.ScrollPercent, &s.UpdatedAt, &s.DeviceID); err != nil {
			return nil, err
		}
		states = append(states, s)
	}
	return states, nil
}

func (db *DB) AddPeer(address, name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	query := `
	INSERT INTO peers (address, name, added_at)
	VALUES (?, ?, ?)
	ON CONFLICT(address) DO UPDATE SET name = excluded.name;
	`
	_, err := db.sql.Exec(query, address, name, time.Now().UnixMilli())
	return err
}

func (db *DB) RemovePeer(address string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.sql.Exec("DELETE FROM peers WHERE address = ?", address)
	return err
}

func (db *DB) ListPeers() ([]PeerInfo, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	rows, err := db.sql.Query("SELECT address, name, added_at FROM peers")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []PeerInfo
	for rows.Next() {
		var p PeerInfo
		if err := rows.Scan(&p.Address, &p.Name, &p.AddedAt); err != nil {
			return nil, err
		}
		peers = append(peers, p)
	}
	return peers, nil
}
