package peer

import (
	"bytes"
	"fmt"
	"io"
	"net"
)

const protocol = "BitTorrent protocol"
const handshakeLen = 49 + len(protocol)

type Handshake struct {
	InfoHash [20]byte
	PeerID   [20]byte
}

func NewHandshake(infoHash [20]byte, peerID string) []byte {
	buf := make([]byte, handshakeLen)
	buf[0] = byte(len(protocol))
	copy(buf[1:], []byte(protocol))
	// reserved bytes (8 bytes of 0)
	copy(buf[20:], make([]byte, 8))
	copy(buf[28:], infoHash[:])
	copy(buf[48:], []byte(peerID))
	return buf
}

func PerformHandshake(conn net.Conn, infoHash [20]byte, peerID string) error {
	
	handshake := NewHandshake(infoHash, peerID)
	_, err := conn.Write(handshake)
	if err != nil {
		return fmt.Errorf("failed to send handshake: %w", err)
	}

	resp := make([]byte, handshakeLen)
	_, err = io.ReadFull(conn, resp)
	if err != nil {
		return fmt.Errorf("failed to read handshake: %w", err)
	}

	if int(resp[0]) != len(protocol) || string(resp[1:20]) != protocol {
		return fmt.Errorf("invalid protocol string in handshake")
	}

	if !bytes.Equal(resp[28:48], infoHash[:]) {
		return fmt.Errorf("info_hash mismatch in handshake")
	}

	fmt.Println("✅ Handshake successful with", conn.RemoteAddr())
	return nil

}
