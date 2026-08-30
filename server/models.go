package main

type SyncState struct {
	Site          string  `json:"site"`
	URL           string  `json:"url"`
	ScrollPercent float64 `json:"scrollPercent"`
	UpdatedAt     int64   `json:"updatedAt"`
	DeviceID      string  `json:"deviceId"`
}

type PeerInfo struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	AddedAt int64  `json:"addedAt"`
}
