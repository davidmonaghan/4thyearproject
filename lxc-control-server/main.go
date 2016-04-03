package main

import (
	"fmt"
	"net"
	"encoding/json"
	"log"
)

type NewContainerJSON struct {
	Action string			`json:"Action"`
	ContainerName string	`json:"ContainerName"`
	BaseServer string		`json:"BaseServer"`
	CMS string				`json:"CMS"`
	WebsiteName string		`json:"WebsiteName"`
	DBrootPWD string		`json:"DBrootPWD"`
	DBadminUname string		`json:"DBadminUname"`
	DBadminPWD string		`json:"DBadminPWD"`
}

func main() {
	service := "127.0.0.1:8081"
	
	tcpAddr, err := net.ResolveTCPAddr("tcp", service)
	if err != nil {
		log.Fatalf("Failed to Resolve TCP Address: %s", err)
	}
	
	ln, err := net.ListenTCP("tcp", tcpAddr) 
	if err != nil {
		log.Fatalf("Failed to listen: %s", err)
	} 
	
	for{ 
		if conn, err := ln.Accept(); err == nil {
			go handleConnection(conn)
		}
	}
}

func handleConnection(conn net.Conn){
	defer conn.Close()
	
	decoder := json.NewDecoder(conn)
	
	var b NewContainerJSON
	if err := decoder.Decode(&b); err != nil {
		fmt.Println("encode.Encode error: ", err)
	}
	
    fmt.Println("Slave server recieved JSON from Master server: ",b)
    
    // Send JSON back to source
    encoder := json.NewEncoder(conn)
	if err := encoder.Encode(b); err != nil {
		fmt.Println("encode.Encode error: ", err)
	}
    
    
}
