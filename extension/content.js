console.log("sync-must-simple content script loaded on", window.location.href);

const browserAPI = globalThis.browser || globalThis.chrome;

let debounceTimer = null;
let isRestoring = false;

function getScrollPercent() {
    const h = document.documentElement;
    const b = document.body;
    const st = 'scrollTop';
    const sh = 'scrollHeight';
    
    const scrollHeight = (h[sh] || b[sh]) - h.clientHeight;
    if (scrollHeight <= 0) return 0;
    
    return (h[st] || b[st]) / scrollHeight;
}

function setScrollPercent(percent) {
    const h = document.documentElement;
    const b = document.body;
    const sh = 'scrollHeight';
    
    const scrollHeight = (h[sh] || b[sh]) - h.clientHeight;
    if (scrollHeight <= 0) return;
    
    isRestoring = true;
    window.scrollTo({
        top: scrollHeight * percent,
        behavior: 'smooth'
    });
    
    // Release restore lock after smooth scroll is likely done
    setTimeout(() => {
        isRestoring = false;
    }, 1500);
}

function isSameUrl(url1, url2) {
    if (!url1 || !url2) return false;
    if (url1 === url2) return true;
    try {
        const u1 = new URL(url1);
        const u2 = new URL(url2);
        return u1.origin === u2.origin && 
               u1.pathname.replace(/\/$/, '') === u2.pathname.replace(/\/$/, '') && 
               u1.search === u2.search;
    } catch (e) {
        return url1.split('#')[0].replace(/\/$/, '') === url2.split('#')[0].replace(/\/$/, '');
    }
}

// 1. Ask background for current position when loaded
setTimeout(() => {
    browserAPI.runtime.sendMessage({ type: "GET_STATE", url: window.location.href }).then(response => {
        if (response && response.state && response.state.scrollPercent !== undefined) {
            console.log("Restoring scroll position:", response.state.scrollPercent);
            setScrollPercent(response.state.scrollPercent);
        }
    }).catch(err => console.log("Failed to get state:", err));
}, 500); // Slight delay to ensure page is somewhat rendered

// 2. Listen to scroll events and send to background
window.addEventListener('scroll', () => {
    if (isRestoring) return; // Don't sync while we are auto-scrolling
    
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => {
        const percent = getScrollPercent();
        console.log("Syncing scroll position:", percent);
        browserAPI.runtime.sendMessage({
            type: "UPDATE_STATE",
            payload: {
                url: window.location.href,
                scrollPercent: percent
            }
        }).catch(err => console.log("Failed to sync state:", err));
    }, 1500);
});

// 3. Listen for broadcast updates from background
browserAPI.runtime.onMessage.addListener((message) => {
    if (message.type === "SYNC_BROADCAST" && message.payload) {
        if (isSameUrl(message.payload.url, window.location.href)) {
            console.log("Received remote sync broadcast:", message.payload.scrollPercent);
            setScrollPercent(message.payload.scrollPercent);
        }
    }
});
