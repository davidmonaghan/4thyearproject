// +build linux,cgo

package main

import (
	"fmt"
	"net"
	"encoding/json"
	"log"
	"time"
	"flag"
	"regexp"
	"io"
    "os"
    "os/exec"
    "bufio"
	"gopkg.in/lxc/go-lxc.v2"
	"github.com/codeskyblue/go-sh"
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

var lxcpath string

func init() {	
	flag.StringVar(&lxcpath, "lxcpath", lxc.DefaultConfigPath(), "Use specified container path")
	flag.Parse()
}

func main() {
	log.Println("LXC-Control-Sever is up")
	service := "192.168.1.100:8081"
	
	tcpAddr, err := net.ResolveTCPAddr("tcp", service)
	if err != nil {
		log.Println("Failed to Resolve TCP Address: %s", err)
	} else {
		log.Println("Resolved TCP Address: ", service)
		
		ln, err := net.ListenTCP("tcp", tcpAddr) 
		if err != nil {
			log.Println("Failed to listen: %s", err)
		} else { 
			log.Println("Listening on: ", tcpAddr)
			
			for{ 
				if conn, err := ln.Accept(); err == nil {
					go handleConnection(conn)
				}
			}
		}
	}
}

func handleConnection(conn net.Conn){
	defer conn.Close()
	
	log.Println("Connection Open")
	log.Println("Waiting for Proxy-Control-Server...")
	
	decoder := json.NewDecoder(conn)
	
	var b NewContainerJSON
	if err := decoder.Decode(&b); err != nil {
		log.Println("JSON Decode error: ", err)
	}
	    
    var (
   		newContainerName, containerAction, cStatus, cIPaddress, wordpressStatus string
   	)
   	
    log.Println("LXC-Control-Server recieved JSON from Proxy-Control-Server: ", b)
	
   	if b.Action == "createNew" {
   		newContainerName = fmt.Sprint(b.CustomerUname + "-" + b.ContainerName)
   		originalContainer := "standardClone"  // Temporary measure until logic develpoed to determine which original Template Clone is to be used
   		cStatus, cIPaddress, wordpressStatus = createNewContainer(originalContainer, newContainerName, b.DBrootPWD, b.WebsiteName, b.DBadminUname, b.DBadminPWD)
		containerAction = "UpdateContainerStatus"
	}
	
	if b.Action == "updateDBcontainerStatus" {
		newContainerName = b.ContainerName
		cStatus = updateDBcontainerStatus(b.ContainerName)
		containerAction = "UpdateContainerStatus"
	}
	
	if b.Action == "containerBackUp" {
		t := time.Now()
		var err error
     	newContainerName = fmt.Sprint(b.ContainerName + "-Clone-" + t.Format("2006-01-02-15:04:05"))
		cStatus, err = cloneNewContainer(b.ContainerName, newContainerName)
		if err != nil {
			containerAction = "CloneCreationFailed"
		} else {
			containerAction = "createDBentryForClone"
		}
	}
	
	if b.Action == "containerSnapShot" {
		t := time.Now()
		var err error
     	newContainerName = fmt.Sprint(b.ContainerName + "-Snapshot-" + t.Format("2006-01-02-15:04:05"))
		cStatus, err = snapshotNewContainer(b.ContainerName, newContainerName)
		if err != nil {
			containerAction = "snapshotCreationFailed"
		} else {
			containerAction = "createDBentryForSnaphot"
		}
	}
	
	if b.Action == "containerStart" {
		cStatus, cIPaddress = startContainer(b.ContainerName)
		containerAction = "UpdateContainerStatus"
		newContainerName = b.ContainerName
	}
	
	if b.Action == "containerStop" {
		cStatus = stopContainer(b.ContainerName)
		containerAction = "UpdateContainerStatus"
		newContainerName = b.ContainerName
	}
	
	if b.Action == "containerDelete" {
		cStatus = deleteContainer(b.ContainerName)
		containerAction = "deleteContainerFromDatabase"
		newContainerName = b.ContainerName
	}
	
	if b.Action == "containerRestart" {
		cStatus = stopContainer(b.ContainerName)
		
		if cStatus == "Stopped" {
			cStatus, cIPaddress = startContainer(b.ContainerName)
			containerAction = "UpdateContainerStatus"
			newContainerName = b.ContainerName
		} else {
			containerAction = "UpdateContainerStatus"
			newContainerName = b.ContainerName
		}
		
	}
	
   d := NewContainerJSON {
   		CustomerUname: b.CustomerUname,
		Action: containerAction,
    	ContainerName: newContainerName,
    	BaseServer: b.BaseServer,
    	CMS: b.CMS,
    	WebsiteName: b.WebsiteName,
    	ContainerStatus: cStatus,
    	ContainerIPaddress: cIPaddress,
    	WordpressStatus: wordpressStatus,
    }
    
    log.Println("Sending Proxy-Control-Server the following JSON: ", d)
    
    // Send JSON back to source
    encoder := json.NewEncoder(conn)
	if err := encoder.Encode(d); err != nil {
		fmt.Println("JSON Encode error:: ", err)
	}
}

func updateDBcontainerStatus(containerName string) (string) {
	x, err := regexp.Compile(containerName)
	if err != nil {
		log.Println("Error compiling '", containerName, "' regex: ", err)
		cStatus := "Error"
		return cStatus
	}
	
	y, err := regexp.Compile("RUNNING")
	if err != nil {
		log.Println("Error compiling 'RUNNING' regex: ", err)
		cStatus := "Error"
		return cStatus
	}
	
	z, err := regexp.Compile("STOPPED")
	if err != nil {
		log.Println("Error compiling 'STOPPED' regex: ", err)
		cStatus := "Error"
		return cStatus
	}
	
	cmd := exec.Command("lxc-ls", "-f")

	// capture the output and error pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Error opening stdout pipe: ", err)
		cStatus := "Error"
		return cStatus
		os.Exit(1)
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Println("Error opening stderr pipe: ", err)
		cStatus := "Error"
		return cStatus
		os.Exit(1)
	}
	
	err = cmd.Start()
	if err != nil {
		log.Println("Error running shell commands: ", err)
		cStatus := "Error"
		return cStatus
		os.Exit(1)
	}
	
	defer cmd.Wait()
	
	go io.Copy(os.Stderr, stderr)
	
	buff := bufio.NewScanner(stdout)
	var allText []string

	for buff.Scan() {
		 allText = append(allText, buff.Text()+"\n")
	}
	
	for i := range allText {
		log.Println("Doing a Regex Match on: ", containerName)
		log.Println("Matching Regex against string: ", allText[i])
		
		if x.MatchString(allText[i]) == true {
			log.Println("Regex Match On: ", containerName)
			
			if y.MatchString(allText[i]) == true {
				cStatus := "Running"
				log.Println("Sending database ", cStatus, " for container", containerName)
				return cStatus
			}
			
			if z.MatchString(allText[i]) == true {
				cStatus := "Stopped"
				log.Println("Sending database ", cStatus, " for container: ", containerName)
				return cStatus
			}
		}  
	}
		
	log.Println("No Regex Match On: ", containerName)
	cStatus := "Suspended"
	log.Println("Sending database ", cStatus, " for container: ", containerName)
	return cStatus
}

func createNewContainer(oldContainerName string, newContainerName string, cDBrootPWD string, cWebsiteName string, cDBadminUname string, cDBadminPWD string) (string, string, string) {
	
   	log.Println(newContainerName)
   	
   	var (
   		cStatus, cIPaddress, wordpressStatus string
   		err error
   	) 
   	
	if cStatus, err = cloneNewContainer(oldContainerName, newContainerName); err == nil {
		cStatus, cIPaddress = startContainer(newContainerName)
		wordpressStatus = configureWordpressContainer(newContainerName, cDBrootPWD, cWebsiteName, cDBadminUname, cDBadminPWD)
	} else {
		log.Printf("Not getting IP address or configuring container due to container creation failure")
		cIPaddress = ""
		wordpressStatus = "Wordpress not started"
		cStatus = "Error Creating Container"
	}
		
	log.Println(newContainerName, " Status: ", cStatus)
	log.Println(newContainerName, " IP Address: ", cIPaddress)
	log.Println(newContainerName, " Wordpress Status: ", cStatus)
	
	return cStatus, cIPaddress, wordpressStatus
	
}

func snapshotNewContainer(originalContainerName string, newContainerName string) (string, error) {
	var cStatus string
	
	c, err := lxc.NewContainer(originalContainerName, lxcpath)
	if err != nil {
		cStatus = "Error Creating Snapshot"
		return cStatus, err
	}

	log.Printf("Snapshoting the container...\n")
	if _, err := c.CreateSnapshot(); err != nil {
		cStatus = "Error Creating Snapshot"
		return cStatus, err
	}
	
	cStatus = "Snapshot Created Successfully"
	log.Println(newContainerName, " created successfully")
	return cStatus, nil
	
}

func cloneNewContainer(originalContainer string, containerName string) (string, error) {
	var (
		backend lxc.BackendStore
		cStatus string
	)
	
	if originalContainer == "standardClone" {
		originalContainer = "clone-u1404-A2-MySQL-WP"
	}
	
	lxcpath = lxc.DefaultConfigPath()
	backend = 2
	
	log.Println("Creating new container: ", containerName)
	
	c, err := lxc.NewContainer(originalContainer, lxcpath)
	if err != nil {
		log.Println("Error setting up details for ", containerName, ": ", err)
		cStatus = "Error Creating Container"
		return cStatus, err
	}

	if backend == 0 {
		log.Println("Error setting up Backend Store for ", containerName, ": ", lxc.ErrUnknownBackendStore)
		cStatus = "Error Creating Container"
		return cStatus, err
	}

	log.Println("Creating ", containerName, " using ", backend, " backend...")
	err = c.Clone(containerName, lxc.CloneOptions{
		Backend: backend,
	})
	if err != nil {
		log.Println("Error creating ", containerName, ": ", err)
		cStatus = "Error Creating Container"
		return cStatus, err
	}
	
	cStatus = "Clone Created Successfully"
	log.Println(containerName, " created successfully")
	return cStatus, nil
}

func startContainer(containerName string) (string, string) {
	var (
		cStatus, cIPaddress string
	)
	
	lxcpath = lxc.DefaultConfigPath()
		
	c, err := lxc.NewContainer(containerName, lxcpath)
	if err != nil {
		log.Println("Error Starting Container: ", err)
		cStatus = "Error Starting Container"
		cIPaddress = "Error Starting Container"
		return cStatus, cIPaddress
	}

	c.SetLogFile("/tmp/" + containerName + ".log")
	c.SetLogLevel(lxc.TRACE)

	log.Println("Starting: ", containerName)
	if err := c.Start(); err != nil {
		log.Println("Error Starting", containerName, " : ", err)
		cStatus = "Error Starting Container"
		cIPaddress = "Error Starting Container"
		return cStatus, cIPaddress
	}

	log.Println("Waiting for ", containerName, " to startup networking...")
	if _, err := c.WaitIPAddresses(5 * time.Second); err == nil {
		cStatus = "Running"
	}
	if err != nil {
		log.Println("Error starting ", containerName, ": ", err)
		cStatus = "Error Starting Container"
		cIPaddress = "Error Starting Container"
		return cStatus, cIPaddress
	}
	
	log.Println(containerName, " started successfully")
	cStatus, cIPaddress = getContainerIPaddress(containerName)
	
	return cStatus, cIPaddress
}

func getContainerIPaddress(containerName string) (string, string){
	var (
		lxcpath string
		cIPaddress, cStatus string
	)
	
	log.Println("Getting IP address for container: ", containerName)
	lxcpath = lxc.DefaultConfigPath()
	
	c, err := lxc.NewContainer(containerName, lxcpath)
	if err != nil {
		log.Println("Error Getting IP address: ", err)
		cStatus = "Error Starting Container"
		cIPaddress = "Error Getting IP address"
		return cStatus, cIPaddress
	}
	
	loopControl := 0
	for loopControl != -1 && loopControl <= 5 {
		if addresses, err := c.IPv4Addresses(); err != nil {
			if loopControl == 4 {
				log.Println("Error Getting IP address for ", containerName, ": ", err)
				cStatus = "Error Starting Container"
				cIPaddress = "Error Getting IP address"
				return cStatus, cIPaddress
			}
			log.Println("Error Getting IP address for ", containerName, ": ", err)
			log.Println("Waiting 5 seconds before trying again.")
			loopControl += 1
			time.Sleep(5 * time.Second)
		} else {
			for i := range addresses {
				cIPaddress = addresses[i]
				cStatus = "Running"
				log.Println("Obtained IP address for ", containerName, ": ", cIPaddress)
				loopControl = -1
			}
		}
	}
	
	return cStatus, cIPaddress
}

func stopContainer(containerName string) (string) {
	var (
		cStatus string
	)
	
	c, err := lxc.NewContainer(containerName, lxcpath)
	if err != nil {
		log.Println("Error stopping ", containerName,": ", err)
		cStatus = "Error stopping container"
		return cStatus
	}

	c.SetLogFile("/tmp/" + containerName + ".log")
	c.SetLogLevel(lxc.TRACE)

	log.Println("Stopping: ", containerName)
	if err := c.Stop(); err != nil {
		log.Println("Error stopping ", containerName,": ", err)
		cStatus = "Error stopping container"
		return cStatus
	} else {
		cStatus = "Stopped"
		return cStatus
	}
}

func deleteContainer(containerName string) (string) {
	var (
		cStatus string
	)
	
	c, err := lxc.NewContainer(containerName, lxcpath)
	if err != nil {
		log.Println("Error deleting ", containerName, ": ", err)
		cStatus = "Error Deleting container"
		return cStatus
	}

	c.SetLogFile("/tmp/" + containerName + ".log")
	c.SetLogLevel(lxc.TRACE)

	log.Println("Deleting: ", containerName)
	if err := c.Destroy(); err != nil {
		log.Println("Error deleting ", containerName, ": ", err)
		cStatus = "Error Deleting containe"
		return cStatus
	} else {
		log.Println("Successfully deleted: ", containerName)
		cStatus = "Deleted"
		return cStatus
	}
}

func configureWordpressContainer(containerName string, cDBrootPWD string, cWebsiteName string, cDBadminUname string, cDBadminPWD string) (string) {
	time.Sleep(5 * time.Second) // Sleeping to give time for container to fully boot
	var wpStatus string
	
	log.Println("Waiting 5 seconds for ", containerName, " to fully boot up.")
	log.Println("Configuring Wordpress MySQL database for container: ", containerName)
	
	loopControl := 0
	for loopControl != -1 && loopControl <= 5 {
		if err := sh.Command("lxc-attach", "-n", containerName, "--", "/home/dave/lxc-config-1.sh", cWebsiteName, cDBadminUname, cDBadminPWD).Run(); err != nil {
			if loopControl == 4 {
				log.Println("Wordpress MySQL Database configuration failed: ", err)
				wpStatus = "Error Configuring Worpress MySQL Database.\n"
			}
			log.Println("Waiting 5 more seconds for ", containerName, " to fully boot up.")
			loopControl += 1
			time.Sleep(5 * time.Second)
		} else {
			log.Println("Wordpress MySQL Database successfully configured for container: " + containerName)
			loopControl = -1
		}
	}
		
	log.Println("Configuring Wordpress wp-config.php file for container: ", containerName)
	
	if err := sh.Command("lxc-attach", "-n", containerName, "--", "/home/dave/lxc-config-2.sh", cDBrootPWD, cWebsiteName, cDBadminUname, cDBadminPWD).Run(); err != nil {
		log.Println("Error Configuring Worpress wp-config.php file: ", err)
		wpStatus2 := "Error Configuring Worpress wp-config.php file."
		wpStatus = fmt.Sprint(wpStatus + " " + wpStatus2)
		
	}
	
	log.Println("Wordpress wp-config.php file successfully configured for container: ", containerName)
	wpStatus = "Wordpress Configured"
	return wpStatus	
}
