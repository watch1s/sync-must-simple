# sync-must-simple

An open-source, LAN-only tool to synchronize reading/scroll positions across multiple devices. 
Perfect for keeping track of your progress on long articles, webtoons, or documentation without relying on cloud services.

## Why LAN-only?
Most sync tools require you to create an account and send your reading habits to a cloud server. **sync-must-simple** keeps all your data entirely on your local network. Your reading positions sync directly between your PC, laptop, and phone using a lightweight local server you control.

---

## 1. The Server (Go)

The server is a single, lightweight binary that runs on your computer (typically your desktop PC or a Raspberry Pi). It listens on port `8787` by default.

### Running the Pre-built Binary
*(Note: When you download the release from GitHub, you can just double-click the `.exe` on Windows or run the binary on Linux. A system tray icon will appear indicating the server is running.)*

### Compiling from Source
If you want to build it yourself, you must compile it natively on your target OS (CGO is required for the system tray).

**Windows:**
1. Ensure you have Go and a C compiler (like GCC via MinGW) installed.
2. Run `.\build.ps1` in PowerShell.
3. The binary will be output to `dist\sync-must-simple.exe`.

**Linux:**
1. Run `make deps` to install `libgtk-3-dev` and `libappindicator3-dev`.
2. Run `make build`.
3. The binary will be output to `dist/sync-must-simple-linux`.

*(For CGO-less pure Go testing, just run `go run ./server` from the root).*

---

## 2. The Extension

The browser extension (available for Firefox, Zen Browser, and Firefox for Android) tracks where you are on a page and talks to your local server.

### Setup Instructions
1. Install the extension from the Firefox Add-ons store (or load manually for development).
2. The **Options** page will open.
3. Enter your local server's IP address (e.g., `192.168.1.100:8787`) and click **Save & Test Connection**.
4. Under **Synced Sites**, type the domain of the site you want to track (e.g., `example.com`) and click **Add Domain**.
5. Firefox will ask you for permission to access that specific site. Click Allow.

That's it! Whenever you read on that site, your scroll position will be saved to your local server and synced instantly to your other devices.

## Privacy Policy
Please see [docs/PRIVACY.md](docs/PRIVACY.md) for full details. In short: no cloud, no tracking, your data stays strictly on your network.

## License
MIT License. See [LICENSE](LICENSE) for details.