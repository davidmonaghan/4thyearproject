package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"encoding/json"
	"github.com/gorilla/websocket"
)

type NewContainerJSON struct {
    CustomerType string				`json:"CustomerType,omitempty"`
    CustomerUname string			`json:"CustomerUname,omitempty"`
    Action string                   `json:"Action,omitempty"`
    ContainerName string            `json:"ContainerName,omitempty"`
    BaseServer string               `json:"BaseServer,omitempty"`
    CMS string                      `json:"CMS,omitempty"`
    WebsiteName string              `json:"WebsiteName,omitempty"`
    DBrootPWD string                `json:"DBrootPWD,omitempty"`
    DBadminUname string             `json:"DBadminUname,omitempty"`
    DBadminPWD string               `json:"DBadminPWD,omitempty"`
    ContainerStatus string          `json:"ContainerStatus,omitempty"`
    ContainerIPaddress string       `json:"ContainerIPaddress,omitempty"`
    WordpressStatus string			`json:"WordpressStatus,omitempty"`
}

var addr = flag.String("addr", "192.168.1.101:8080" , "http service address") // "127.0.0.1:8080"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

func main() {
    // http.HandleFunc("/", sayhelloName) // setting router rule
	log.Print("Proxy server is up")
    flag.Parse()
	log.Println("Listening for data from Web-Server 1")
	http.HandleFunc("/user", user)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func user(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
		
	if err != nil {
		log.Printf("Websocket upgrade error:", err)
		return
	}
	if err == nil {
		log.Printf("Websocket upgrade completed")
	}
	
	go handleWebServerConnection(c)
}

func handleWebServerConnection(ws *websocket.Conn) {
	defer ws.Close()
	
	for {
		log.Println("Listening on WebSocket")
		
		var b NewContainerJSON
		
		err := ws.ReadJSON(&b)
		if err != nil {
			log.Printf("Error reading json:", err)
			break
		}
		
		log.Println("Recieved JSON from web server: ", b)
		
/////------------- Get List of Containers -------------\\\\\ 		
		if b.Action == "getListOfContainers" {
			log.Println("Getting list of containers")
			if err := getListOfContainers(ws, b.Action, b.CustomerType, b.CustomerUname); err == nil {
				log.Println("Finished getting list of Container")
			}
			if err != nil {
				log.Println("Error getting list of Container: ", err)
			}
		}
				
/////------------- Create New Container or Backup Container -------------\\\\\ 		
		if b.Action == "createNew" || b.Action == "containerStart" || b.Action == "containerStop" || b.Action == "containerDelete" || b.Action == "containerRestart" || b.Action == "containerBackUp" || b.Action == "containerSnapShot" {
			
			dialService := "192.168.1.100:8081" //"127.0.0.1:8081"

			conn, err := net.Dial("tcp", dialService)
			if err != nil {
				log.Println("LXC Server Dial error: ", err)
				log.Println("Dialling WEB server as ", err)
				// dialWebServerContainerAlreadyExists(ws, dbConStatus)
				 c := NewContainerJSON {
				 	Action: "LXC Server Dial Error",
					ContainerStatus: "Error Dialing LXC-Control-Server Server",
				}
		
				err = ws.WriteJSON(c)
				if err != nil {
					log.Printf("Error sending JSON to Webserver:", err)
					break
				} else {
					log.Println("JSON sent to Webserver: ", c)			
				}
				
			} else {
				log.Println("Dialing LXC-Control-Server")
			
				dbConStatus, err := updateDatabaseContainerStatus(b.Action, b.CustomerUname, b.ContainerName, b.ContainerStatus)
				if err != nil {
					log.Printf("Error Dialing Database Server: ", err)
					b.ContainerStatus = "Error Dialing Database Server"
				} else {
					log.Println("Dialing Database-Control-Server")
				}
			
				// Database returns an error, break loop
				if dbConStatus != "GoodToGo" {
					log.Println("Dialling WEB server as ", dbConStatus)
					// dialWebServerContainerAlreadyExists(ws, dbConStatus)
					 c := NewContainerJSON {
					 	Action: "Database Error",
						ContainerStatus: dbConStatus,
					}
			
					err = ws.WriteJSON(c)
					if err != nil {
						log.Printf("Error sending JSON to Webserver:", err)
						break
					} else {
						log.Println("JSON sent to Webserver: ", c)			
					}
				} 
				// Database returns NO error, send signal to LXC server to perform action
				if dbConStatus == "GoodToGo" {
					encoder := json.NewEncoder(conn)
					if err := encoder.Encode(b); err != nil {
						log.Printf("JSON Encode error: ", err)
					}

					decoder := json.NewDecoder(conn)

					var d NewContainerJSON
			
					if err := decoder.Decode(&d); err != nil {
						log.Printf("JSON Decode error: ", err)
					}

					log.Println("Proxy Control Server recieved JSON from LXC Control server: ",d)
				
					var dbStatus string
			
					// Update database with status of customers request
					dbStatus, err = updateDatabaseContainerStatus(d.Action, d.CustomerUname, d.ContainerName, d.ContainerStatus)
					if err != nil {
						log.Printf("Error Dialing Database Server: ", err)
						break
					}
				
					// Send web-server status of customer request
					d.ContainerStatus = dbStatus
			
					err = ws.WriteJSON(d)
					if err != nil {
						log.Printf("Error JSON to Webserver:", err)
						break
					} else {
						log.Printf("JSON sent to Webserver")			
					}
				}
			
			}
			
		}
	}
}

func getListOfContainers(ws *websocket.Conn, action string, cusType string, cusUname string) (error) {
	var b NewContainerJSON
	b.Action = action
	b.CustomerUname = cusUname
	b.CustomerType = cusType

	dialService := "192.168.1.103:8082" //"127.0.0.1:8081"

	conn, err := net.Dial("tcp", dialService)
	if err != nil {
		log.Println("Database Dial error: ", err)
		
		b.Action = "Database Error"
		b.ContainerStatus = "Error Dialing Database Server"
		
		err = ws.WriteJSON(b)
		if err != nil {
			log.Println("Error JSON to Webserver:", err)
			return err
		} else {
			log.Println("JSON sent to Webserver")			
		}
		
		return err
	}
	
	log.Println("Dialing Database-Control-Server")
	
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(b); err != nil {
		log.Printf("JSON Encode error: ", err)
	}
	
		
	killSig := false

	for killSig == false {
		decoder := json.NewDecoder(conn)

		var d NewContainerJSON

		if err := decoder.Decode(&d); err != nil {
			log.Println("JSON Decode error: ", err)
		}

		log.Println("Proxy Control Server recieved JSON from LXC Control server: ",d)
	
		if d.Action == "kill" {
			killSig = true
		}		
		err = ws.WriteJSON(d)
		if err != nil {
			log.Println("Error JSON to Webserver:", err)
			return err
		} else {
			log.Println("JSON sent to Webserver")			
		}

	}
	
	return nil
}

func updateDatabaseContainerStatus(action string, custUname string, cName string, conStatus string) (string, error) {
	if action == "CloneCreationFailed" || action == "snapshotCreationFailed" {
		return conStatus, nil
	}
	
	dialService := "192.168.1.103:8082" //"127.0.0.1:8081"

	conn, err := net.Dial("tcp", dialService)
	if err != nil {
		log.Printf("Database Server Dial error: ", err)
		return "Database Dial Error", err
	}
	defer conn.Close()
	
	log.Printf("Dialing Database Server")
	
	var b NewContainerJSON
	b.Action = action
	b.CustomerUname = custUname
	b.ContainerName = cName
	b.ContainerStatus = conStatus

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(b); err != nil {
		log.Printf("JSON Encode error: ", err)
	}

	decoder := json.NewDecoder(conn)

	var c NewContainerJSON
	if err := decoder.Decode(&c); err != nil {
		log.Printf("JSON Decode error: ", err)
		return "Database Dial Error", err
	}

	log.Println("Master server recieved JSON from Database server: ",c)
		
	return c.ContainerStatus, nil
}

/*func dialWebServerContainerAlreadyExists(ws *websocket.Conn, status string) {
	defer ws.Close()
	var c NewContainerJSON
	
	c.ContainerStatus = status
	
	err := ws.WriteJSON(c)
	if err != nil {
		log.Printf("Error sending JSON to Webserver:", err)
		
	} else {
		log.Printf("JSON sent to Webserver")			
	}
}*/

