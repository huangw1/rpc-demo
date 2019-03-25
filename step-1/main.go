package main

import (
	"net"
	"log"
	"encoding/binary"
	"time"
	"bytes"
)

const HeadSize = 4

func doConn(conn net.Conn) {
	headBuf := bytes.NewBuffer(make([]byte, 0, HeadSize))
	headData := make([]byte, HeadSize)
	headSize := HeadSize
	for {
		num, err := conn.Read(headData)
		if err != nil {
			log.Fatal(err)
			return
		}
		headBuf.Write(headData[:num])
		if headBuf.Len() == HeadSize {
			break
		} else {
			headData = make([]byte, headSize-num)
			headSize -= num
		}
	}

	bodyLen := int(binary.BigEndian.Uint32(headBuf.Bytes()))
	bodySize := bodyLen
	bodyBuf := bytes.NewBuffer(make([]byte, 0, bodyLen))
	bodyData := make([]byte, bodyLen)
	for {
		num, err := conn.Read(bodyData)
		if err != nil {
			log.Fatal(err)
			return
		}
		bodyBuf.Write(bodyData[:num])
		if bodyBuf.Len() == bodyLen {
			log.Printf("receive body: %v", bodyBuf.Bytes())
			break
		} else {
			bodyData = make([]byte, bodySize-num)
			bodySize -= num
		}
	}
}

func handleHTTP() {
	listener, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Println("start listening at 1234")
	for {
		conn, err := listener.Accept()
		log.Println("receive conn")
		if err != nil {
			log.Fatal(err)
			return
		}
		go doConn(conn)
	}
}

func sendStringWithTcp(str string) error {
	conn, err := net.Dial("tcp", ":1234")
	if err != nil {
		log.Fatal(err)
		return err
	}
	head := make([]byte, HeadSize)
	content := []byte(str)
	binary.BigEndian.PutUint32(head, uint32(len(content)))
	_, err = conn.Write(head)
	if err != nil {
		log.Fatal(err)
		return err
	}
	_, err = conn.Write(content)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func main() {
	go func() {
		time.Sleep(time.Second)
		log.Println("start send tcp")
		sendStringWithTcp("test")
	}()
	handleHTTP()
}
