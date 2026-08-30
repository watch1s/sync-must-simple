const serverInput = document.getElementById('server-address');
const serverStatus = document.getElementById('server-status');
const newDomainInput = document.getElementById('new-domain');
const domainList = document.getElementById('domain-list');

async function loadSettings() {
    const data = await browser.storage.local.get(['serverAddress', 'domains']);
    if (data.serverAddress) {
        serverInput.value = data.serverAddress;
    }
    renderDomains(data.domains || []);
}

loadSettings();

document.getElementById('save-server').addEventListener('click', async () => {
    let addr = serverInput.value.trim();
    if (!addr) return;
    
    if (addr.startsWith('http://')) addr = addr.substring(7);
    if (addr.startsWith('https://')) addr = addr.substring(8);
    addr = addr.split('/')[0];
    
    // Request permission to communicate with the server
    try {
        await browser.permissions.request({
            origins: [`http://${addr}/*`]
        });
    } catch (e) {
        console.log("Permission request ignored or failed", e);
    }
    
    await browser.storage.local.set({ serverAddress: addr });
    
    serverStatus.textContent = "Testing...";
    serverStatus.className = "status";
    
    try {
        const res = await fetch(`http://${addr}/state`);
        if (res.ok) {
            serverStatus.textContent = "Connected!";
            serverStatus.className = "status success";
        } else {
            throw new Error("Bad response");
        }
    } catch (e) {
        serverStatus.textContent = "Failed to connect. Is the server running?";
        serverStatus.className = "status error";
    }
});

document.getElementById('add-domain').addEventListener('click', async () => {
    let domain = newDomainInput.value.trim();
    if (!domain) return;
    
    domain = domain.replace(/^https?:\/\//, '').split('/')[0];
    
    const permission = {
        origins: [`*://${domain}/*`]
    };
    
    const granted = await browser.permissions.request(permission);
    if (granted) {
        const data = await browser.storage.local.get(['domains']);
        const domains = data.domains || [];
        if (!domains.includes(domain)) {
            domains.push(domain);
            await browser.storage.local.set({ domains });
            await registerContentScript(domain);
            renderDomains(domains);
            newDomainInput.value = '';
        } else {
            alert("Domain already added.");
        }
    } else {
        alert("Permission denied for this domain.");
    }
});

async function registerContentScript(domain) {
    const scriptId = `sync-script-${domain}`;
    try {
        await browser.scripting.unregisterContentScripts({ ids: [scriptId] });
    } catch (e) {}
    
    await browser.scripting.registerContentScripts([{
        id: scriptId,
        matches: [`*://${domain}/*`],
        js: ["content.js"],
        runAt: "document_idle"
    }]);
}

async function unregisterContentScript(domain) {
    const scriptId = `sync-script-${domain}`;
    try {
        await browser.scripting.unregisterContentScripts({ ids: [scriptId] });
    } catch (e) {
        console.error("Failed to unregister", e);
    }
}

function renderDomains(domains) {
    domainList.innerHTML = '';
    domains.forEach(domain => {
        const li = document.createElement('li');
        li.textContent = domain;
        
        const btn = document.createElement('button');
        btn.textContent = "Remove";
        btn.className = "btn-danger";
        btn.onclick = async () => {
            const data = await browser.storage.local.get(['domains']);
            const newDomains = (data.domains || []).filter(d => d !== domain);
            await browser.storage.local.set({ domains: newDomains });
            await unregisterContentScript(domain);
            try {
                await browser.permissions.remove({ origins: [`*://${domain}/*`] });
            } catch (e) {}
            renderDomains(newDomains);
        };
        
        li.appendChild(btn);
        domainList.appendChild(li);
    });
}
