console.log("sync-must-simple background script loaded");

const browserAPI = globalThis.browser || globalThis.chrome;
const SERVER_ADDRESS = "127.0.0.1:8787";

let ws = null;
let reconnectTimer = null;
let localStateCache = {}; // URL -> state
const DEVICE_ID = "device-" + Math.random().toString(36).substr(2, 9);

function connectWS() {
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return;
    
    const wsUrl = `ws://${SERVER_ADDRESS}/ws`;
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
            console.log("WS received state update:", state);
            if (state && state.url) {
                localStateCache[state.url] = state;
                if (state.deviceId !== DEVICE_ID) {
                    browserAPI.tabs.query({}).then(tabs => {
                        const targetBaseUrl = state.url.split('#')[0];
                        for (const tab of tabs) {
                            if (!tab.url) continue;
                            const tabBaseUrl = tab.url.split('#')[0];
                            if (tabBaseUrl === targetBaseUrl || tab.url === state.url) {
                                browserAPI.tabs.sendMessage(tab.id, {
                                    type: "SYNC_BROADCAST",
                                    payload: state
                                }).catch(() => {
                                    // Ignored: tab might not have content script injected
                                });
                            }
                        }
                    }).catch(err => {
                        console.error("Failed to query tabs for sync broadcast:", err);
                    });
                }
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
    reconnectTimer = setTimeout(connectWS, 5000);
}

async function fetchState() {
    try {
        const res = await fetch(`http://${SERVER_ADDRESS}/state`);
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
        await fetch(`http://${SERVER_ADDRESS}/state`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(state)
        });
    } catch (e) {
        console.error("Failed to update state on server", e);
    }
}

browserAPI.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.type === "PING_SERVER") {
        (async () => {
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 5000);
            try {
                const res = await fetch(`http://${SERVER_ADDRESS}/state`, {
                    cache: "no-store",
                    signal: controller.signal
                });
                clearTimeout(timeoutId);
                if (res.ok) {
                    sendResponse({ ok: true });
                } else {
                    sendResponse({ ok: false });
                }
            } catch (err) {
                clearTimeout(timeoutId);
                sendResponse({ ok: false });
            }
        })();
        return true;
    }
    
    if (message.type === "GET_STATE") {
        sendResponse({ state: localStateCache[message.url] });
        return true; 
    }
    
    if (message.type === "UPDATE_STATE") {
        if (!ws || ws.readyState !== WebSocket.OPEN) {
            connectWS();
        }
        updateState(message.payload.url, message.payload.scrollPercent);
        sendResponse({ success: true });
        return true;
    }
});

connectWS();
