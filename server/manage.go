package main

import (
	"fmt"
	"net/http"
)

const manageHTML = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Peer Management</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; padding: 20px; max-width: 600px; margin: 0 auto; color: #333; }
        .card { border: 1px solid #ddd; border-radius: 8px; padding: 15px; margin-bottom: 20px; background: #fff; }
        h1, h2 { margin-top: 0; }
        .peer-item { display: flex; justify-content: space-between; align-items: center; padding: 10px 0; border-bottom: 1px solid #eee; }
        .peer-item:last-child { border-bottom: none; }
        .status { display: inline-block; width: 10px; height: 10px; border-radius: 50%; margin-right: 5px; }
        .online { background: #4caf50; }
        .offline { background: #f44336; }
        .btn { background: #007bff; color: white; border: none; padding: 6px 12px; border-radius: 4px; cursor: pointer; }
        .btn:hover { background: #0056b3; }
        .btn-danger { background: #dc3545; }
        .btn-danger:hover { background: #c82333; }
        input { padding: 6px; margin-right: 10px; border: 1px solid #ccc; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="card">
        <h1>Peer Management</h1>
        <p>Share this address with other devices: <strong id="myAddress">Loading...</strong></p>
    </div>

    <div class="card">
        <h2>Add Peer</h2>
        <form id="addForm" onsubmit="addPeer(event)">
            <input type="text" id="peerAddress" placeholder="IP:8788" required>
            <input type="text" id="peerName" placeholder="Name (optional)">
            <button type="submit" class="btn">Add</button>
        </form>
    </div>

    <div class="card">
        <h2>Peers</h2>
        <div id="peerList">Loading...</div>
    </div>

    <script>
        var host = window.location.host;

        (function() {
            fetch('/peer/ping').then(function(res) { return res.json(); }).then(function(data) {
                document.getElementById('myAddress').textContent = data.address || host;
            }).catch(function() {
                document.getElementById('myAddress').textContent = host;
            });
        })();

        function renderPeer(p) {
            var statusClass = p.online ? 'online' : 'offline';
            var statusTitle = p.online ? 'Online' : 'Offline';
            var nameStr = p.name ? ' (' + p.name + ')' : '';
            var html = '<div class="peer-item">';
            html += '<div>';
            html += '<span class="status ' + statusClass + '" title="' + statusTitle + '"></span>';
            html += '<strong>' + p.address + '</strong>' + nameStr;
            html += '</div>';
            html += '<button class="btn btn-danger" onclick="removePeer(\'' + p.address + '\')">Remove</button>';
            html += '</div>';
            return html;
        }

        function loadPeers() {
            fetch('/peers').then(function(res) { return res.json(); }).then(function(peers) {
                var list = document.getElementById('peerList');
                if (peers.length === 0) {
                    list.innerHTML = '<p>No peers added yet.</p>';
                    return;
                }
                var html = '';
                for (var i = 0; i < peers.length; i++) {
                    html += renderPeer(peers[i]);
                }
                list.innerHTML = html;
            }).catch(function(err) {
                console.error(err);
            });
        }

        function addPeer(e) {
            e.preventDefault();
            var address = document.getElementById('peerAddress').value;
            var name = document.getElementById('peerName').value;
            fetch('/peers', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ address: address, name: name })
            }).then(function(res) {
                if (!res.ok) {
                    return res.text().then(function(text) { throw new Error(text); });
                }
                document.getElementById('peerAddress').value = '';
                document.getElementById('peerName').value = '';
                loadPeers();
                alert('Peer added successfully!');
            }).catch(function(err) {
                alert('Failed to add peer: ' + err.message);
            });
        }

        function removePeer(address) {
            if (!confirm('Remove peer ' + address + '?')) return;
            fetch('/peers', {
                method: 'DELETE',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ address: address })
            }).then(function() {
                loadPeers();
            });
        }

        loadPeers();
        setInterval(loadPeers, 5000);
    </script>
</body>
</html>`

func HandleManage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, manageHTML)
}
