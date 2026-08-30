console.log("sync-must-simple content script loaded on", window.location.href);

const browserAPI = globalThis.browser || globalThis.chrome;

let debounceTimer = null;
let isRestoring = false;

function getScrollHeight() {
    return Math.max(
        document.body.scrollHeight, document.documentElement.scrollHeight,
        document.body.offsetHeight, document.documentElement.offsetHeight,
        document.body.clientHeight, document.documentElement.clientHeight
    );
}

function getClientHeight() {
    return window.innerHeight || document.documentElement.clientHeight || document.body.clientHeight;
}

function getScrollTop() {
    return window.scrollY || window.pageYOffset || document.documentElement.scrollTop || document.body.scrollTop || 0;
}

function getScrollPercent() {
    const scrollHeight = getScrollHeight();
    const clientHeight = getClientHeight();
    const maxScroll = scrollHeight - clientHeight;
    
    if (maxScroll <= 0) return 0;
    
    return getScrollTop() / maxScroll;
}

function setScrollPercent(percent) {
    const scrollHeight = getScrollHeight();
    const clientHeight = getClientHeight();
    const maxScroll = scrollHeight - clientHeight;
    
    if (maxScroll <= 0) return;
    
    isRestoring = true;
    window.scrollTo({
        top: maxScroll * percent,
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
