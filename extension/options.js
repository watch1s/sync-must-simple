const browserAPI = globalThis.browser || globalThis.chrome;
const SERVER = 'localhost:8787';
const clientStatus = document.getElementById('client-status');
const newDomainInput = document.getElementById('new-domain');
const domainMsg = document.getElementById('domain-msg');
const domainList = document.getElementById('domain-list');

function showMessage(text, isError = false) {
    if (!domainMsg) return;
    domainMsg.textContent = text;
    domainMsg.style.display = 'block';
    domainMsg.style.color = isError ? '#dc3545' : '#198754';
    setTimeout(() => {
        domainMsg.style.display = 'none';
    }, 5000);
}

async function checkClient() {
    try {
        const res = await browserAPI.runtime.sendMessage({ type: 'PING_SERVER' });
        if (res && res.ok) {
            clientStatus.textContent = '\u2705 Connected to local client';
            clientStatus.className = 'status-bar success';
        } else {
            throw new Error('not ok');
        }
    } catch (e) {
        clientStatus.textContent = '\u274c Local client not running. Please start the sync-must-simple app.';
        clientStatus.className = 'status-bar error';
    }
}

async function loadSettings() {
    try {
        const data = await browserAPI.storage.local.get(['domains']);
        renderDomains(data.domains || []);
    } catch (e) {
        console.error("Failed to load settings:", e);
    }
}

function cleanDomain(input) {
    if (!input) return '';
    let domain = input.trim().toLowerCase();
    // Remove protocol
    domain = domain.replace(/^https?:\/\//, '');
    // Remove path and query
    domain = domain.split('/')[0].split('?')[0];
    // Remove port if present
    domain = domain.split(':')[0];
    return domain;
}

function getSafeScriptId(domain) {
    return 'sync_script_' + domain.replace(/[^a-zA-Z0-9_-]/g, '_');
}

function getOriginsForDomain(domain) {
    if (domain === 'localhost' || /^(\d{1,3}\.){3}\d{1,3}$/.test(domain) || domain.startsWith('*.')) {
        return [`*://${domain}/*`];
    }
    return [`*://${domain}/*`, `*://*.${domain}/*`];
}

async function registerContentScript(domain) {
    if (!browserAPI.scripting || !browserAPI.scripting.registerContentScripts) {
        console.log("scripting API not available, skipping dynamic content script registration");
        return;
    }
    const scriptId = getSafeScriptId(domain);
    try {
        await browserAPI.scripting.unregisterContentScripts({ ids: [scriptId] });
    } catch (e) {
        // Ignore if wasn't registered
    }

    try {
        await browserAPI.scripting.registerContentScripts([{
            id: scriptId,
            matches: [`*://${domain}/*`, `*://*.${domain}/*`],
            js: ["content.js"],
            runAt: "document_idle"
        }]);
    } catch (e) {
        console.warn("Failed to register with wildcard subdomain, falling back to exact:", e);
        await browserAPI.scripting.registerContentScripts([{
            id: scriptId,
            matches: [`*://${domain}/*`],
            js: ["content.js"],
            runAt: "document_idle"
        }]);
    }
}

async function unregisterContentScript(domain) {
    if (!browserAPI.scripting || !browserAPI.scripting.unregisterContentScripts) return;
    const scriptId = getSafeScriptId(domain);
    try {
        await browserAPI.scripting.unregisterContentScripts({ ids: [scriptId] });
    } catch (e) {
        console.error("Failed to unregister content script:", e);
    }
}

async function handleAddDomain() {
    const rawInput = newDomainInput.value;
    const domain = cleanDomain(rawInput);
    if (!domain) {
        showMessage("Please enter a valid domain name (e.g. example.com).", true);
        return;
    }

    try {
        // IMPORTANT: browserAPI.permissions.request MUST be called synchronously
        // inside the click handler turn before any other `await` (storage, network, etc.)
        // to preserve the transient user activation / gesture token.
        const origins = getOriginsForDomain(domain);
        let granted = false;
        try {
            granted = await browserAPI.permissions.request({ origins });
        } catch (originErr) {
            console.warn("Full wildcard origin request failed, trying single origin:", originErr);
            granted = await browserAPI.permissions.request({ origins: [`*://${domain}/*`] });
        }

        if (!granted) {
            showMessage("Permission was not granted for this domain.", true);
            return;
        }

        // Now that permission is granted, read storage and persist
        const data = await browserAPI.storage.local.get(['domains']);
        const domains = data.domains || [];
        if (!domains.includes(domain)) {
            domains.push(domain);
            await browserAPI.storage.local.set({ domains });
        }

        try {
            await registerContentScript(domain);
        } catch (regErr) {
            console.error("Register content script warning:", regErr);
        }

        renderDomains(domains);
        newDomainInput.value = '';
        showMessage(`Successfully added '${domain}' to synced sites!`, false);
    } catch (err) {
        console.error("Error adding domain:", err);
        showMessage("Error: " + (err.message || err), true);
    }
}

document.getElementById('add-domain').addEventListener('click', handleAddDomain);

newDomainInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
        e.preventDefault();
        handleAddDomain();
    }
});

function renderDomains(domains) {
    domainList.innerHTML = '';
    domains.forEach(domain => {
        const li = document.createElement('li');
        li.textContent = domain;

        const btn = document.createElement('button');
        btn.textContent = "Remove";
        btn.className = "btn-danger";
        btn.onclick = async () => {
            try {
                const data = await browserAPI.storage.local.get(['domains']);
                const newDomains = (data.domains || []).filter(d => d !== domain);
                await browserAPI.storage.local.set({ domains: newDomains });
                await unregisterContentScript(domain);
                try {
                    if (browserAPI.permissions && browserAPI.permissions.remove) {
                        await browserAPI.permissions.remove({ origins: [`*://${domain}/*`, `*://*.${domain}/*`] });
                    }
                } catch (e) {}
                renderDomains(newDomains);
                showMessage(`Removed '${domain}'.`, false);
            } catch (err) {
                console.error("Failed to remove domain:", err);
                showMessage("Failed to remove: " + err.message, true);
            }
        };

        li.appendChild(btn);
        domainList.appendChild(li);
    });
}

checkClient();
loadSettings();
