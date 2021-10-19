package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

type server struct {
	host string
	port string
	pass string
}

func main() {
	var s server

	flag.StringVar(&s.host, "host", "127.0.0.1", "minecraft host ip")
	flag.StringVar(&s.port, "port", "25580", "minecraft RCON port")
	flag.StringVar(&s.pass, "pass", "", "RCON password")
	flag.Parse()

	var buf bytes.Buffer

	if err := s.send(strings.Join(flag.Args(), " "), &buf); err != nil {
		log.Fatal(err)
	}

	fmt.Println(buf.String())
}

func (s server) send(command string, out io.Writer) error {
	conn, err := net.Dial(`tcp`, net.JoinHostPort(s.host, s.port))
	if err != nil {
		return fmt.Errorf("failed to connect with error %v", err)
	}
	defer conn.Close()

	// Login
	loginPacket := createRconConnectionPacket(1, packetTypeLogin, []byte(s.pass))

	if _, err := conn.Write(loginPacket); err != nil {
		return err
	}
	if err := readResponseFromConn(conn, 1, nil); err != nil {
		return fmt.Errorf("failed to login: %v", err)
	}

	commandPacket := createRconConnectionPacket(2, packetTypeCommand, []byte(command))
	if _, err := conn.Write(commandPacket); err != nil {
		return err
	}

	if err := readResponseFromConn(conn, 2, out); err != nil {
		return fmt.Errorf("command failed: %v", err)
	}

	return nil
}

const (
	packetTypeResponse uint32 = 0
	packetTypeCommand  uint32 = 2
	packetTypeLogin    uint32 = 3
)

func createRconConnectionPacket(requestID, packetType uint32, data []byte) []byte {
	for _, b := range data {
		if b > 127 {
			panic("supported - non-ASCII byte in packet")
		}
	}
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(len(data)+10))
	binary.Write(&buf, binary.LittleEndian, requestID)
	binary.Write(&buf, binary.LittleEndian, packetType)
	buf.Write(data)
	buf.Write([]byte{0, 0})
	return buf.Bytes()
}

func readResponseFromConn(conn net.Conn, requestID uint32, out io.Writer) error {
	var respBuf bytes.Buffer

	for {
		readBuf := make([]byte, 4096)
		readCount, err := conn.Read(readBuf)
		if err != nil {
			return err
		}
		respBuf.Write(readBuf[:readCount])
		if readCount < 4096 {
			break
		}
	}

	var rLen, rID, pType uint32

	binary.Read(&respBuf, binary.LittleEndian, &rLen)
	binary.Read(&respBuf, binary.LittleEndian, &rID)
	binary.Read(&respBuf, binary.LittleEndian, &pType)
	if rID != requestID {
		return fmt.Errorf("unexpected request id (%d) from response", rID)
	}

	if out != nil {
		body := respBuf.Bytes()
		out.Write(body[:len(body)-2])
	}

	return nil
}
