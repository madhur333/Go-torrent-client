# Torrent-client

A simple  BitTorrent client written in Go. The project demonstrates parsing .torrent files, contacting a tracker, connecting to peers, performing the BitTorrent handshake, and downloading pieces concurrently.

Requirements
- Go 1.18+ (with modules enabled)

Quick start
1. Install dependencies:
   go mod download

2. Run (no build required):
   go run main.go

     To download a torrent file, edit the `path` variable near the top of `main.go` and run again.

Project layout
- `main.go` — CLI entry point and download orchestrator
- `torrentfile/` — .torrent parsing and helpers
- `tracker/` — tracker announce and peer parsing
- `peer/` — peer handshake and message handling

Test results / benchmark
- Test file: Debian ISO (~800 MB)
- Peers: 10 concurrent peers
- Result: Download completed successfully in 17m45.1185913s
- Average download speed: 752.77 KB/s

Notes & limitations
- Single-file torrents only (uses `info.length`).
- Minimal BitTorrent protocol support: no DHT, limited choking logic, limited error recovery.


