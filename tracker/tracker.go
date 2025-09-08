package tracker

import (
	"bytes"
	crand "crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/url"

	bencode "github.com/jackpal/bencode-go"
	"mytorrent/torrentfile"
)

type Peer struct {
	IP   string
	Port uint16
}

// Generates a 20-byte peer ID like: -GT0001-<random_bytes>
func generatePeerID() string {
	prefix := "-GT0001-"
	suffix := make([]byte, 12)
	_, err := crand.Read(suffix)
	if err != nil {
		panic(err)
	}
	return prefix + string(suffix)
}

// Computes SHA-1 hash of the bencoded "info" dictionary
func computeInfoHash(info interface{}) [20]byte {
	var buf bytes.Buffer
	if err := bencode.Marshal(&buf, info); err != nil {
		panic(err)
	}
	return sha1.Sum(buf.Bytes())
}

type trackerResponse struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"` // compact peer list format
}

// Sends request to tracker and parses peer list
func GetPeers(tf *torrentfile.TorrentFile) ([]Peer, error) {
	peerID := generatePeerID()
	infoHash := computeInfoHash(tf.Info)

	// Properly use url.Values without double-encoding
	params := url.Values{}
	params.Add("info_hash", string(infoHash[:]))
	params.Add("peer_id", peerID)
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", fmt.Sprintf("%d", tf.Info.Length))
	params.Add("compact", "1")

	fullURL := fmt.Sprintf("%s?%s", tf.Announce, params.Encode())

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tr trackerResponse
	err = bencode.Unmarshal(resp.Body, &tr)
	if err != nil {
		return nil, err
	}

	peers := parsePeers([]byte(tr.Peers))
	return peers, nil
}

// Parses peers from compact peer format (6 bytes per peer: 4 for IP, 2 for port)
func parsePeers(peersBin []byte) []Peer {
	const peerSize = 6
	numPeers := len(peersBin) / peerSize
	peers := make([]Peer, numPeers)

	for i := 0; i < numPeers; i++ {
		offset := i * peerSize
		ip := net.IP(peersBin[offset : offset+4]).String()
		port := binary.BigEndian.Uint16(peersBin[offset+4 : offset+6])
		peers[i] = Peer{IP: ip, Port: port}
	}

	return peers
}
