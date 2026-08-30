console.log("sync-must-simple background script loaded");

let ws = null;
let reconnectTimer = null;
let localStateCache = {}; // URL -> state
const DEVICE_ID = "device-" + Math.random().toString(36).substr(2, 9);

let serverAddress = ""; // Loaded from storage

async function init() {
    const data = await browser.storage.local.get(['serverAddress']);
    if (data.serverAddress) {
        serverAddress = data.serverAddress;
        connectWS();
    }
}

// Listen for settings changes
browser.storage.onChanged.addListener((changes, area) => {
    if (area === 'local' && changes.serverAddress) {
        serverAddress = changes.serverAddress.newValue;
        if (ws) ws.close(); // will trigger reconnect with new address
        if (!ws && serverAddress) connectWS();
    }
});

function connectWS() {
    if (!serverAddress) return;
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return;
    
    const wsUrl = `ws://${serverAddress}/ws`;
    console.log("Connecting to WebSocket:", wsUrl);
    
    try {
        ws = new WebSocket(wsUrl);
    } catch (e) {
        console.error("WebSocket creation failed", e);
        scheduleReconnect();
        return;
    }
    
    ws.onopen = () => {
        console.log("WebSocket connected");
        clearTimeout(reconnectTimer);
        fetchState();
    };
    
    ws.onmessage = (event) => {
        try {
            const state = JSON.parse(event.data);
            if (state.deviceId !== DEVICE_ID) {
                localStateCache[state.url] = state;
                browser.tabs.query({ url: state.url }).then(tabs => {
                    for (const tab of tabs) {
                        browser.tabs.sendMessage(tab.id, {
                            type: "SYNC_BROADCAST",
                            payload: state
                        }).catch(err => {
                            // Ignored: tab might not have content script injected
                        });
                    }
                });
            }
        } catch (e) {
            console.error("Failed to parse WS message", e);
        }
    };
    
    ws.onclose = () => {
        console.log("WebSocket closed.");
        scheduleReconnect();
    };
    
    ws.onerror = (err) => {
        console.error("WebSocket error", err);
    };
}

function scheduleReconnect() {
    clearTimeout(reconnectTimer);
    if (serverAddress) {
        reconnectTimer = setTimeout(connectWS, 5000);
    }
}

async function fetchState() {
    if (!serverAddress) return;
    try {
        const res = await fetch(`http://${serverAddress}/state`);
        if (res.ok) {
            const states = await res.json();
            for (const s of states) {
                localStateCache[s.url] = s;
            }
        }
    } catch (e) {
        console.error("Failed to fetch state", e);
    }
}

async function updateState(url, scrollPercent) {
    if (!serverAddress) return;
    let site = "unknown";
    try {
        site = new URL(url).hostname || "unknown";
    } catch(e) {
        site = "local-file";
    }

    const state = {
        site: site,
        url: url,
        scrollPercent: scrollPercent,
        updatedAt: Date.now(),
        deviceId: DEVICE_ID
    };
    
    localStateCache[url] = state;
    
    try {
        await fetch(`http://${serverAddress}/state`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(state)
        });
    } catch (e) {
        console.error("Failed to update state on server", e);
    }
}

browser.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.type === "GET_STATE") {
        sendResponse({ state: localStateCache[message.url] });
        return true; 
    }
    
    if (message.type === "UPDATE_STATE") {
        if (serverAddress && (!ws || ws.readyState !== WebSocket.OPEN)) {
            connectWS();
        }
        updateState(message.payload.url, message.payload.scrollPercent);
        sendResponse({ success: true });
        return true;
    }
});

// Run init
init();
