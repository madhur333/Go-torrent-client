package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"mytorrent/peer"
	"mytorrent/torrentfile"
	"mytorrent/tracker"
)

type PieceResult struct {
	Index int64
	Data  []byte
	Err   error
}

func downloadPiece(p tracker.Peer, infoHash [20]byte, peerID string, tf *torrentfile.TorrentFile, pieceIndex int64, blockSize int) PieceResult {
	addr := fmt.Sprintf("%s:%d", p.IP, p.Port)

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return PieceResult{Index: pieceIndex, Err: fmt.Errorf("connection failed: %w", err)}
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(120 * time.Second))

	if err := peer.PerformHandshake(conn, infoHash, peerID); err != nil {
		return PieceResult{Index: pieceIndex, Err: fmt.Errorf("handshake failed: %w", err)}
	}

	if err := peer.SendMessage(conn, peer.MakeInterested()); err != nil {
		return PieceResult{Index: pieceIndex, Err: fmt.Errorf("failed to send interested: %w", err)}
	}

	for {
		msg, err := peer.ReadMessage(conn)
		if err != nil {
			return PieceResult{Index: pieceIndex, Err: fmt.Errorf("failed to read message: %w", err)}
		}
		if msg == nil {
			continue
		}
		if msg.ID == peer.MsgUnchoke {
			break
		}
	}

	pieceLength := tf.Info.PieceLength
	var currentPieceLength int
	numPieces := (int64(tf.Info.Length) + int64(pieceLength) - 1) / int64(pieceLength)
	if pieceIndex == numPieces-1 {
		currentPieceLength = int(int64(tf.Info.Length) - pieceIndex*int64(pieceLength))
	} else {
		currentPieceLength = pieceLength
	}

	numBlocks := (currentPieceLength + blockSize - 1) / blockSize
	fullPiece := make([]byte, currentPieceLength)

	for blk := 0; blk < numBlocks; blk++ {
		begin := blk * blockSize
		length := blockSize
		if begin+length > currentPieceLength {
			length = currentPieceLength - begin
		}

		req := peer.MakeRequest(int(pieceIndex), begin, length)
		if err := peer.SendMessage(conn, req); err != nil {
			return PieceResult{Index: pieceIndex, Err: fmt.Errorf("failed to send request: %w", err)}
		}

		for {
			msg, err := peer.ReadMessage(conn)
			if err != nil {
				return PieceResult{Index: pieceIndex, Err: fmt.Errorf("failed to read message: %w", err)}
			}
			if msg == nil {
				continue
			}
			if msg.ID == peer.MsgPiece {
				off := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
				data := msg.Payload[8:]
				copy(fullPiece[off:], data)
				break
			}
		}
	}

	hash := sha1.Sum(fullPiece)
	expectedStart := pieceIndex * 20
	expectedEnd := (pieceIndex + 1) * 20
	if expectedEnd > int64(len(tf.Info.Pieces)) {
		return PieceResult{Index: pieceIndex, Err: fmt.Errorf("torrent metadata 'pieces' field too short")}
	}
	expected := []byte(tf.Info.Pieces[expectedStart:expectedEnd])
	if !bytes.Equal(hash[:], expected) {
		return PieceResult{Index: pieceIndex, Err: fmt.Errorf("piece hash mismatch")}
	}

	return PieceResult{Index: pieceIndex, Data: fullPiece, Err: nil}
}

func main() {
	path := ""

	tf, err := torrentfile.Open(path)
	if err != nil {
		panic(err)
	}

	fmt.Println("Tracker URL:", tf.Announce)
	fmt.Println("File name:", tf.Info.Name)
	fmt.Println("File length:", tf.Info.Length)

	peers, err := tracker.GetPeers(tf)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Found %d peers\n", len(peers))

	peerID := "-GT0001-123456789012"
	infoHash := tf.HashInfo()

	pieceLength := tf.Info.PieceLength
	numPieces := (int64(tf.Info.Length) + int64(pieceLength) - 1) / int64(pieceLength)
	blockSize := 16 * 1024

	file, err := os.Create(tf.Info.Name)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	pieceJobs := make(chan int64, numPieces)
	results := make(chan PieceResult, numPieces)

	workerCount := len(peers)
	if workerCount > 10 {
		workerCount = 10 
	}

	var wg sync.WaitGroup

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func(peerIdx int) {
			defer wg.Done()
			for pieceIndex := range pieceJobs {
				res := downloadPiece(peers[peerIdx], infoHash, peerID, tf, pieceIndex, blockSize)
				if res.Err != nil {
					fmt.Printf("Peer %d failed to download piece %d: %v\n", peerIdx, pieceIndex, res.Err)
					
				} else {
					fmt.Printf("Peer %d downloaded and verified piece %d\n", peerIdx, pieceIndex)
				}
				results <- res
			}
		}(w)
	}

	startTime := time.Now()
	var downloadedPieces int64 = 0

	go func() {
		for i := int64(0); i < numPieces; i++ {
			pieceJobs <- i
		}
		close(pieceJobs)
	}()

	for i := int64(0); i < numPieces; i++ {
		res := <-results
		if res.Err != nil {
			fmt.Printf("Error downloading piece %d: %v\n", res.Index, res.Err)
			
			continue
		}

		_, err := file.WriteAt(res.Data, res.Index*int64(pieceLength))
		if err != nil {
			fmt.Printf("Failed to write piece %d: %v\n", res.Index, err)
			return
		}

		downloadedPieces++
		fmt.Printf("Downloaded piece %d/%d (%.2f%% complete)\n", downloadedPieces, numPieces, float64(downloadedPieces)*100/float64(numPieces))
	}

	wg.Wait()

	duration := time.Since(startTime)
	fmt.Printf("\nDownload completed successfully in %v\n", duration)
	fmt.Printf("Average download speed: %.2f KB/s\n", float64(tf.Info.Length)/1024/duration.Seconds())
}

