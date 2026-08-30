const SERVER = 'localhost:8787';
const clientStatus = document.getElementById('client-status');
const newDomainInput = document.getElementById('new-domain');
const domainList = document.getElementById('domain-list');

async function checkClient() {
    try {
        const res = await browser.runtime.sendMessage({ type: 'PING_SERVER' });
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
    const data = await browser.storage.local.get(['domains']);
    renderDomains(data.domains || []);
}

checkClient();
loadSettings();

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
