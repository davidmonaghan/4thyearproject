package main

import(
	"net"
	"encoding/json"
	"log"
	"fmt"
	"strings"
	"time"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
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

func main() {
	func main() {
	log.Println("Database-Control-Server is up")

	service := "192.168.1.103:8082" //"127.0.0.1:8081"
	
	db, err := sql.Open("mysql","dave:admin@tcp(127.0.0.1:3306)/praxis")
	if err != nil {
		log.Println("Database Connection failed: ", err)
	}
	
	defer db.Close()	
	
	err = db.Ping()
	if err != nil {
		log.Println("MySQL Database Ping failed: ", err)
		log.Println("Database Server is down")
	} else {
		log.Println("DB Ping successful")
		
		err = updateDBcontainerStatusOnStartUp(db)
		if err != nil {
			log.Println("Error getting update from LXC-Control-Server: ", err)
			log.Println("Database Server is down")
		} else {
			log.Println("Finished getting update of current registered containers status from LXC-Control Server")
			
			tcpAddr, err := net.ResolveTCPAddr("tcp", service)
			if err != nil {
				log.Println("Failed to Resolve TCP Address: %s", err)
				log.Println("Database Server is down")
			} else {
				log.Println("Resolved TCP Address: ", service)
				ln, err := net.ListenTCP("tcp", tcpAddr) 
								
				if err != nil {
					log.Println("Failed to listen: %s", err)
					log.Println("Database Server is down")
				} else { 
					log.Println("Listening on: ", tcpAddr)
					
					for{ 
						if conn, err := ln.Accept(); err == nil {
							go handleConnection(conn, db)
						}
					}
				}
			}
		}
	}
}

func handleConnection(conn net.Conn, db *sql.DB){
	
	defer conn.Close()

	decoder := json.NewDecoder(conn)

	var b NewContainerJSON
	if err := decoder.Decode(&b); err != nil {
		log.Println("JSON Decode error: ", err)
	}

	log.Println("Database Control server recieved JSON from Master Control server: ",b)
	
	var cStatus, cName string
	
	if b.Action == "createNew" {
		cName = fmt.Sprint(b.CustomerUname + "-" + b.ContainerName)
	} else {
		cName = b.ContainerName
	}

	if b.Action == "getListOfContainers" {
		
		if err := getListOfContainers(conn, db, b.Action, b.CustomerUname, b.CustomerType); err==nil {
			log.Println ("All containers sent to Praxis-Proxy-Server")
		} else {
			log.Println ("Error sending containers to Praxis-Proxy-Server: ", err)
		}
// Action "createNew" "UpdateContainerStatus" "createDBentryForClone" "deleteContainerFromDatabase" "getListOfContainers"
// Action "containerStart" "containerStop" "containerDelete" "containerRestart" "containerBackUp" "containerSnapShot" createDBentryForSnaphot		
	} else {
		cStatus = handleDatabaseQuery(db, b.Action, b.CustomerUname, cName, b.ContainerStatus)
		
		c := NewContainerJSON {
			Action: b.Action,
			CustomerUname: b.CustomerUname,
			ContainerName: cName,
			ContainerStatus: cStatus,
		}

		log.Println("This is the new JSON we will send back: ", c)

		// Send JSON back to source
		encoder := json.NewEncoder(conn)
		if err := encoder.Encode(c); err != nil {
			log.Println("JSON Encode error:: ", err)
		}
	}
	
}

func handleDatabaseQuery(db *sql.DB, action string, cusUname string, cName string, conStatus string) (string) {
	var cStatus string
	    
	if action == "createNew" {
		
		cStatus = runCheckIfContainerExist(db, cusUname, cName, action)
		if cStatus == "GoodToGo" {
			cStatus = addDataBaseEntryForNewContainer(db, action, cusUname, cName)
		}
		log.Println("Here 3", cStatus)
	}
	
	if action == "containerStart" || action == "containerStop" || action == "containerRestart" || action == "containerBackUp" || action == "containerSnapShot" {
		cStatus = runCheckIfContainerExist(db, cusUname, cName, action)
	}
	
	if action == "containerDelete" {
		cStatus = runCheckIfContainerExist(db, cusUname, cName, action)
		
		if cStatus == "GoodToGo" {
			cStatus = checkIfExistingContainerHasAClone(db, cusUname, cName, action)
			
			if cStatus == "GoodToGo" {
				cStatus = checkIfExistingContainerHasASnaphot(db, cusUname, cName, action)
			}
		}
		
		return cStatus
	}
	
    if action == "createDBentryForClone" || action == "createDBentryForSnaphot" {
    	cStatus = addDataBaseEntryForNewContainer(db, action, cusUname, cName)
    	log.Println("Here 5")
    } 
    
   
	
	if action == "UpdateContainerStatus" {
		cStatus = updateContainerStatus(db, cName, conStatus)
	}
	
	if action == "deleteContainerFromDatabase" {
		cStatus = deleteContainerFromContainer_Table(db, cusUname, cName)
	}
	
	log.Println("Here 4", cStatus)
		
	return cStatus
}


func checkIfExistingContainerHasASnaphot(db *sql.DB, cusUname string, cName string, action string) (string) {
	var qryResult string
	
	rows, err := db.Query("SELECT Container_Snapshot_Table.Container_Snapshot_Name FROM Container_Snapshot_Table INNER JOIN Container_Table ON Container_Snapshot_Table.Container_ID=Container_Table.Container_ID WHERE Container_Table.Container_Name=(?);", cName)
	
	if err != nil {
		log.Println("Error creating MySQL Query: ", err)
		cStatus := "Error Reading Database"
			return cStatus
	}
	
	defer rows.Close()
	
	for rows.Next() {
		err := rows.Scan(&qryResult)
		if err != nil {
			log.Println("Error 1 reading MySQL output: ", err)
			cStatus := "Error Reading Database"
			return cStatus
		}
		log.Println("This is the query result", qryResult)
	}
	
	err = rows.Err()
	if err != nil {
		log.Println("Error 2 reading MySQL output: ", err)
		cStatus := "Error Reading Database"
		return cStatus
	}
	
	// Container Clone and Snapshot will be evaulated here
	if qryResult == "" {
		log.Println("Here 1100: Exit GoodToGo")
		cStatus := "GoodToGo"
		return cStatus
	} else {
		log.Println("Here 1200: Backup of original container already exists")
		cStatus := "Cannot delete container as a backup already exists"
		return cStatus
	}
}

func checkIfExistingContainerHasAClone(db *sql.DB, cusUname string, cName string, action string) (string) {
	var qryResult string
	
	rows, err := db.Query("SELECT Container_Backup_Table.Container_Backup_Name FROM Container_Backup_Table INNER JOIN Container_Table ON Container_Backup_Table.Container_ID=Container_Table.Container_ID WHERE Container_Table.Container_Name=(?);", cName)
	
	if err != nil {
		log.Println("Error creating MySQL Query: ", err)
		cStatus := "Error Reading Database"
			return cStatus
	}
	
	defer rows.Close()
	
	for rows.Next() {
		err := rows.Scan(&qryResult)
		if err != nil {
			log.Println("Error 1 reading MySQL output: ", err)
			cStatus := "Error Reading Database"
			return cStatus
		}
		log.Println("This is the query result", qryResult)
	}
	
	err = rows.Err()
	if err != nil {
		log.Println("Error 2 reading MySQL output: ", err)
		cStatus := "Error Reading Database"
		return cStatus
	}
	
	// Container Clone and Snapshot will be evaulated here
	if qryResult == "" {
		log.Println("Here 900: Exit GoodToGo")
		cStatus := "GoodToGo"
		return cStatus
	} else {
		log.Println("Here 1000: Backup of original container already exists")
		cStatus := "Cannot delete container as a backup already exists"
		return cStatus
	}
}

func getListOfContainers(conn net.Conn, db *sql.DB, cAction string, cusUname string, cusType string) (error){
	defer conn.Close()
	
	var (
		containerStatus string
		containerName string
	)
		
	rows, err := db.Query("SELECT Container_Table.Container_Name, Container_Table.Container_Status FROM Container_Table  INNER JOIN Customer_Table ON Container_Table.Customer_ID=Customer_Table.Customer_ID WHERE Customer_Table.Customer_UserName=(?)",cusUname)
	
	c := NewContainerJSON {
		Action: cAction,
		CustomerUname: cusUname,
	}
	
	if cusType == "admin" {
		rows, err = db.Query("select Container_Table.Container_Name, Container_Table.Container_Status from Container_Table;")
	}
	
	if err != nil {
		log.Println("Error creating MySQL Query: ", err)
		return err
	}
	
	defer rows.Close()
		
	for rows.Next() {
		err := rows.Scan(&containerName, &containerStatus)
		if err != nil {
			log.Println("Error 1 reading MySQL output: ", err)
			return err
		}
		log.Println("This is the query result: ", containerName, "-",containerStatus)	
		
		c.ContainerName = containerName
		c.ContainerStatus = containerStatus
		log.Println("This is the new JSON we will send back: ", c)

		// Send JSON back to source
		encoder := json.NewEncoder(conn)
		if err := encoder.Encode(c); err != nil {
			log.Println("JSON Encode error: ", err)
		}
		
		log.Println("Waiting 1 second")
		time.Sleep(1 * time.Second)
	}
	
	err = rows.Err()
	if err != nil {
		log.Println("Error 2 reading MySQL output: ", err)
		return err
	}
	
	c.Action = "kill"
	log.Println("This is the new JSON we will send back: ", c)
	
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(c); err != nil {
		log.Println("JSON Encode error:: ", err)
	}
	
	return nil
}

func runCheckIfContainerExist(db *sql.DB, cusUname string, cName string, action string) (string) {
	var qryResult string
		
	rows, err := db.Query("SELECT Container_Table.Container_Name FROM Container_Table INNER JOIN Customer_Table ON Container_Table.Customer_ID=Customer_Table.Customer_ID WHERE Container_Table.Container_Name=(?) AND  Customer_Table.Customer_UserName=(?)", cName, cusUname)
	
	if err != nil {
		log.Println("Error creating MySQL Query: ", err)
		cStatus := "Error Reading Database"
			return cStatus
	}
	
	defer rows.Close()
	
	for rows.Next() {
		err := rows.Scan(&qryResult)
		if err != nil {
			log.Println("Error 1 reading MySQL output: ", err)
			cStatus := "Error Reading Database"
			return cStatus
		}
		log.Println("This is the query result", qryResult)
	}
	
	err = rows.Err()
	if err != nil {
		log.Println("Error 2 reading MySQL output: ", err)
		cStatus := "Error Reading Database"
		return cStatus
	}
	
	// If it's a change to an existing containers status, check that it is at that status already 
	if action == "containerStart" || action == "containerStop" || action == "containerDelete" || action == "containerRestart" || action == "containerBackUp" /*|| action == "containerSnapShot"*/ {
		if qryResult == "" {
			log.Println("Here 500: Container Does Not Exist")
			cStatus := "Container Does Not Exist"
			return cStatus
		} else {
			log.Println("Here 600: Exit GoodToGo")
			cStatus := checkExistingContainersCurrentStatus(db, action, cName)
			return cStatus
		}
	}
	
	// We need to invert result for createNew Container as we only want to create container if it does not already exist
	if action == "createNew" {
		if qryResult == "" {
			log.Println("Here 100: Exit GoodToGo")
			cStatus := "GoodToGo"
			return cStatus
		} else {
			log.Println("Here 200: Container Already Exists")
			cStatus := "Container Already Exists"
			return cStatus
		}
	}
	
	// Container Clone and Snapshot will be evaulated here
	if qryResult == "" {
		log.Println("Here 300: Container Does Not Exist")
		cStatus := "Container Does Not Exist"
		return cStatus
	} else {
		log.Println("Here 400: Exit GoodToGo")
		cStatus := "GoodToGo"
		return cStatus
	}
}

func checkExistingContainersCurrentStatus(db *sql.DB, action string, cName string) (string) {
	var qryResult, containerStatus string
	
	switch action {
		case "containerStart": containerStatus = "Running"
		case "containerStop": containerStatus = "Stopped"
		case "containerRestart": containerStatus = "Stopped"
		case "containerDelete": containerStatus = "Running"
		case "containerBackUp": containerStatus = "Running"
		case "containerSnapShot": containerStatus = "Running" 
	}	
	rows, err := db.Query("SELECT Container_Table.Container_Name FROM Container_Table WHERE Container_Name=(?) and Container_Status=(?)", cName, containerStatus)
	
	if err != nil {
		log.Println("Error creating MySQL Query: ", err)
		cStatus := "Error Updating Database"
			return cStatus
	}
	
	defer rows.Close()
	
	for rows.Next() {
		err := rows.Scan(&qryResult)
		if err != nil {
			log.Println("Error 1 reading MySQL output: ", err)
			cStatus := "Error Updating Database"
			return cStatus
		}
		log.Println("This is the query result", qryResult)
	}
	
	log.Println("This is the query result", qryResult)
	
	err = rows.Err()
	if err != nil {
		log.Println("Error 2 reading MySQL output: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	
	if qryResult == "" {
		log.Println("Here 800: Exit GoodToGo")
		cStatus := "GoodToGo"
		return cStatus
	} else {
		log.Println("Here 700: Container Does Not Exist")
		cStatus := "Containers current staus is not compatable with your request"
		return cStatus
	}
}

func addDataBaseEntryForNewContainer (db *sql.DB, action string, cusUname string, cName string) (string) {
	var cStatus string
	var customerID string
		
	rows, err := db.Query ("select Customer_ID FROM Customer_Table WHERE Customer_UserName=(?);",cusUname)
		
	if err != nil {
		log.Println("Error creating MySQL Query: ", err)
		cStatus := "Error Updating Database"
			return cStatus
	}
	
	defer rows.Close()
	
	for rows.Next() {
		err := rows.Scan(&customerID)
		if err != nil {
			log.Println("Error 1 reading MySQL output: ", err)
			cStatus := "Error Updating Database"
			return cStatus
		}
		log.Println(customerID)
	}
	
	err = rows.Err()
	if err != nil {
		log.Println("Error 2 reading MySQL output: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	
	if customerID == "" {
		log.Println("User does not exist")
		cStatus := "User does not exist"
		return cStatus
	}
	
	if action == "createNew" {
		cStatus = dbQryInsertIntoContainer_Table(db, 1, customerID, cName, cusUname)
	}
	
	if action == "createDBentryForClone" {
		cStatus = dbQryInsertIntoContainer_Backup_Table(db, customerID, cName)
	}
	
	if action == "createDBentryForSnaphot" {
		cStatus = dbQryInsertIntoContainer_Snapshot_Table(db, customerID, cName)
	}
	return cStatus
}

func deleteContainerFromContainer_Table(db *sql.DB, cusUname string, cName string) (string) {
	var cStatus string
	
	stmt, err := db.Prepare("DELETE FROM Container_Table WHERE Container_Table.Container_Name=(?)")
	
	if err != nil {
		log.Println("Error preparing database statement: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	
	res, err := stmt.Exec(cName)
	if err != nil {
		log.Println("Error 1 Adding New Container To Database: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	} else {
		cStatus = "Deleted"
	}
	
	lastId, err := res.LastInsertId()
	if err != nil {
		log.Println("Error 2 Adding New Container To Database: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	
	rowCnt, err := res.RowsAffected()
	if err != nil {
		log.Println("Error 3 Adding New Container To Database: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	log.Printf("ID = %d, affected = %d\n", lastId, rowCnt)
	
	return cStatus	
}

func dbQryInsertIntoContainer_Snapshot_Table (db *sql.DB, custNumber string, clonedContainerName string) (string) {
	strArray := strings.Split(clonedContainerName,"-")

	fmt.Println(strArray)

	originalContainerName := fmt.Sprint(strArray[0] + "-" + strArray[1])

	fmt.Println(originalContainerName)
	
	var cStatus string
	var container_Table_ID string
		
	rows, err := db.Query ("select Container_ID FROM Container_Table WHERE Container_Name=(?);", originalContainerName)
		
	if err != nil {
		log.Println("Error creating MySQL Query: ", err)
		cStatus := "Error Updating Database"
			return cStatus
	}
	
	defer rows.Close()
	
	for rows.Next() {
		err := rows.Scan(&container_Table_ID)
		if err != nil {
			log.Println("Error 1 reading MySQL output: ", err)
			cStatus := "Error Updating Database"
			return cStatus
		}
		log.Println(container_Table_ID)
	}
	
	err = rows.Err()
	if err != nil {
		log.Println("Error 2 reading MySQL output: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	
	if container_Table_ID == "" {
		log.Println("Container does not exist")
		cStatus := "Container does not exist"
		return cStatus
	}
	
	stmt, err := db.Prepare("INSERT INTO Container_Snapshot_Table ( Container_ID, Container_Snapshot_Name ) VALUES (?,?)")
	
	if err != nil {
		log.Println("Error preparing database statement: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	
	res, err := stmt.Exec(container_Table_ID ,clonedContainerName)
	if err != nil {
		log.Println("Error 1 Adding New Container To Database: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	} else {
		cStatus = "GoodToGo"
	}
	
	lastId, err := res.LastInsertId()
	if err != nil {
		log.Println("Error 2 Adding New Container To Database: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	
	rowCnt, err := res.RowsAffected()
	if err != nil {
		log.Println("Error 3 Adding New Container To Database: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	log.Printf("ID = %d, affected = %d\n", lastId, rowCnt)
	
	return cStatus
	
}


func dbQryInsertIntoContainer_Backup_Table (db *sql.DB, custNumber string, clonedContainerName string) (string) {
	strArray := strings.Split(clonedContainerName,"-")

	fmt.Println(strArray)

	originalContainerName := fmt.Sprint(strArray[0] + "-" + strArray[1])

	fmt.Println(originalContainerName)
	
	var cStatus string
	var container_Table_ID string
		
	rows, err := db.Query ("select Container_ID FROM Container_Table WHERE Container_Name=(?);", originalContainerName)
		
	if err != nil {
		log.Println("Error creating MySQL Query: ", err)
		cStatus := "Error Updating Database"
			return cStatus
	}
	
	defer rows.Close()
	
	for rows.Next() {
		err := rows.Scan(&container_Table_ID)
		if err != nil {
			log.Println("Error 1 reading MySQL output: ", err)
			cStatus := "Error Updating Database"
			return cStatus
		}
		log.Println(container_Table_ID)
	}
	
	err = rows.Err()
	if err != nil {
		log.Println("Error 2 reading MySQL output: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	
	if container_Table_ID == "" {
		log.Println("Container does not exist")
		cStatus := "Container does not exist"
		return cStatus
	}
	
	stmt, err := db.Prepare("INSERT INTO Container_Backup_Table ( Container_ID, Container_Backup_Name ) VALUES (?,?)")
	
	if err != nil {
		log.Println("Error preparing database statement: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	
	res, err := stmt.Exec(container_Table_ID ,clonedContainerName)
	if err != nil {
		log.Println("Error 1 Adding New Container To Database: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	} else {
		cStatus = "GoodToGo"
	}
	
	lastId, err := res.LastInsertId()
	if err != nil {
		log.Println("Error 2 Adding New Container To Database: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	
	rowCnt, err := res.RowsAffected()
	if err != nil {
		log.Println("Error 3 Adding New Container To Database: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	log.Printf("ID = %d, affected = %d\n", lastId, rowCnt)
	
	return cStatus
	
}

func dbQryInsertIntoContainer_Table(db *sql.DB, baseContainerNum int, custNumber string, cName string, cusUname string) (string) {
	var cStatus string
	
	stmt, err := db.Prepare("INSERT INTO Container_Table ( Base_Container_Clone_ID, Customer_ID, Container_Name ) VALUES (?,?,?)")
	
	if err != nil {
		log.Println("Error preparing database statement: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	
	res, err := stmt.Exec(baseContainerNum ,custNumber ,cName)
	if err != nil {
		log.Println("Error 1 Adding New Container To Database: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	} else {
		cStatus = "GoodToGo"
	}
	
	lastId, err := res.LastInsertId()
	if err != nil {
		log.Println("Error 2 Adding New Container To Database: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	
	rowCnt, err := res.RowsAffected()
	if err != nil {
		log.Println("Error 3 Adding New Container To Database: ", err)
		cStatus := "Error Updating Database"
		return cStatus
	}
	log.Printf("ID = %d, affected = %d\n", lastId, rowCnt)
	
	return cStatus
}

func updateContainerStatus(db *sql.DB, containerName string, conStatus string) (string) {
	var cStatus string
	
	log.Println(containerName, " ", conStatus)
	stmt, err := db.Prepare("UPDATE Container_Table SET Container_Status=(?) WHERE Container_Name=(?)")
	
	if err != nil {
		log.Println("Error preparing database statement: ", err)
	}
	
	res, err := stmt.Exec(conStatus ,containerName)
	if err != nil {
		log.Println("Error Updating Database", err)
		cStatus = "Error Updating Database"
	} else {
		cStatus = conStatus
	}
	
	lastId, err := res.LastInsertId()
	if err != nil {
		log.Println(err)
	}
	rowCnt, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("ID = %d, affected = %d\n", lastId, rowCnt)
	
	return cStatus
}

/////////////////////////////////////////  Runs On Startup Only //////////////////////////////////////////////////

func updateDBcontainerStatusOnStartUp(db *sql.DB) (error) {
	var (
		ContainerID int
		baseContainerID int
		customerID int
		containerStatus string
		containerName string
		
	)
	log.Println("Getting update of current registered containers status from LXC-Control Server")
	rows, err := db.Query("SELECT * FROM Container_Table WHERE Container_Status != 'Deleted'")
	if err != nil {
		log.Println("Error creating MySQL query", err)
	}
	
	defer rows.Close()
	
	for rows.Next() {
		err := rows.Scan(&ContainerID, &baseContainerID, &customerID, &containerStatus, &containerName)
		if err != nil {
			log.Println("Error reading MySQL query results", err)
			return err
		}
		
		if containerStatus, err = dialLXCserver(containerName); err != nil {
			log.Println("LXC-Control-Server Dial Error", err)
			return err
		}
		if err == nil {
			containerStatus = updateContainerStatus(db, containerName, containerStatus)
			log.Println(containerName," container updated to: ", containerStatus)
		}
	}
	
	err = rows.Err()
	if err != nil {
		log.Println("Error reading MySQL query", err)
		return err
	}
		
	return nil
}

func dialLXCserver(containerName string, ) (string, error) {
	service := "192.168.1.100:8081" //"127.0.0.1:8081"
	
	conn, err := net.Dial("tcp", service)
	if err != nil {
		log.Println("LXC Server Dial error: ", err)
		return "", err
	}
	defer conn.Close()
	
	log.Printf("Dialing LXC Server")
	
	var b NewContainerJSON
	b.Action = "updateDBcontainerStatus"
	b.ContainerName = containerName
	

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(b); err != nil {
		log.Println("JSON Encode error: ", err)
		return "", err
	}

	decoder := json.NewDecoder(conn)

	var c NewContainerJSON
	if err := decoder.Decode(&c); err != nil {
		log.Println("JSON Decode error: ", err)
		return "", err
	}

	log.Println("Master server recieved JSON from Database server: ",c)
	
	return c.ContainerStatus, nil
}

