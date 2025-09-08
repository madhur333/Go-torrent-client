package torrentfile

import (
	"os"
	"bytes"
	"crypto/sha1"

	bencode "github.com/jackpal/bencode-go"
)

type TorrentFile struct {
	Announce string `bencode:"announce"`
	Info     struct {
		Name        string `bencode:"name"`
		Length      int    `bencode:"length"`
		PieceLength int    `bencode:"piece length"`
		Pieces      string `bencode:"pieces"`
	} `bencode:"info"`
}

func Open(path string) (*TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	tf := &TorrentFile{}
	err = bencode.Unmarshal(file, tf)
	if err != nil {
		return nil, err
	}

	return tf, nil
}

func (t *TorrentFile) HashInfo() [20]byte {
	var buf bytes.Buffer
	bencode.Marshal(&buf, t.Info)
	return sha1.Sum(buf.Bytes())
}
