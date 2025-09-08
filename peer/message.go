package peer

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
)

const (
	MsgChoke         = 0
	MsgUnchoke       = 1
	MsgInterested    = 2
	MsgNotInterested = 3
	MsgHave          = 4
	MsgBitfield      = 5
	MsgRequest       = 6
	MsgPiece         = 7
)

type Message struct {
	ID      uint8
	Payload []byte
}

func ReadMessage(conn net.Conn) (*Message, error) {
	var lengthBuf [4]byte
	_, err := io.ReadFull(conn, lengthBuf[:])
	if err != nil {
		return nil , err
	}
	length := binary.BigEndian.Uint32(lengthBuf[:])

	if length == 0 {
		return nil, nil // keep-alive message
	}

	buf := make([]byte, length)
	_, err = io.ReadFull(conn , buf) 
	if err != nil {
	return nil , err
	}

	msg := &Message{
		ID:      buf[0],
		Payload: buf[1:],
	}
	return msg, nil
}

func SendMessage(conn net.Conn , msg *Message) error{
	var buf bytes.Buffer
		length := uint32(len(msg.Payload) + 1)
		binary.Write(&buf, binary.BigEndian, length)
		buf.WriteByte(msg.ID)
		buf.Write(msg.Payload)
		_, err := conn.Write(buf.Bytes())
		return err

}


func MakeInterested() *Message {
	return &Message{ID: MsgInterested}
}


func MakeRequest(index, begin, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))
	return &Message{ID: MsgRequest, Payload: payload}
}