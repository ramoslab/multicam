// Listen and Serve: Communication with the client

package lns

import (
    "net"
    "fmt"
    "strings"
    "net/http"
    "encoding/json"
    "io/ioutil"
    "bitbucket.com/andrews2000/multicam/taskqueue"
)

////UDP SERVER

// Defines the configuration of the server and its functions
type RecUdpServer struct {
    // UDP connection
    Conn *net.UDPConn
    // UDP address
    Addr *net.UDPAddr
    // Task manager
    Tq taskqueue.TaskQueue
}

func (rudps RecUdpServer) Run(q chan bool) {

    buf := make([]byte, 1024)

    for {
        select {
            case <- q:
                fmt.Println("Stopping UDP listener.")
                //close(com)
                return
            default:
                n, _, err := rudps.Conn.ReadFromUDP(buf)
                // Parse command and put it on the task queue
                com := parseUdpCommand(string(buf[0:n]), rudps.Conn, rudps.Addr)
                rudps.Tq.Queue <- com

                fmt.Println("Received ", string(buf[0:n]))

                if err != nil {
                    fmt.Println("Error: ", err)
                }
        }
    }
}

// Parse commands being received via UDP and initiate execution of the commands
//FIXME How do we know the source of the command: Maybe by instead of using strings to store the commands using some sort of a struct
func parseUdpCommand(cmd string, conn *net.UDPConn, addr *net.UDPAddr) UdpCommand {
    var retVal UdpCommand
    spl := strings.Split(cmd, ":")
    if len(spl) == 3 {
        switch spl[0] {
        case "CTL":
            fmt.Println("Control command received: "+spl[2])
            switch spl[2] {
            case "START":
                fmt.Println("Writing \"Start\" to queue")
                retVal = UdpCommand{Type: "CTL", Payload: "START", Timestamp: "000", conn: conn, addr: addr}
            case "STOP":
                retVal = UdpCommand{Type: "CTL", Payload: "STOP", Timestamp: "000", conn: conn, addr: addr}
            case "PREPARE":
                retVal = UdpCommand{Type: "CTL", Payload: "PREPARE", Timestamp: "000", conn: conn, addr: addr}
            default:
                retVal = UdpCommand{Type: "ERROR", Payload: "ERROR", Timestamp: "000", conn: conn, addr: addr}
                fmt.Println("Ignoring invalid command: "+spl[2])
            }
        case "DAT":
            fmt.Println("Data received.")
            retVal = UdpCommand{Type: "DATA", Payload: "xxx", Timestamp: "000", conn: conn, addr: addr}
        case "REQ":
            fmt.Println("Request received.")
            retVal = UdpCommand{Type: "REQ", Payload: "REQ", Timestamp: "000", conn: conn, addr: addr}
            // Ausformulieren
        default:
            //FIXME Proper error handling needed
            fmt.Println("Invalid command received.")
            retVal = UdpCommand{Type: "ERROR", Payload: "ERROR", Timestamp: "000", conn: conn, addr: addr}
        }
    } else {
        if spl[0] == "EXIT" {
            //stopAndExit(qudp,conn)
            //TODO Exit as a command
            retVal = UdpCommand{Type: "EXIT", Payload: "EXIT", Timestamp: "000", conn: conn, addr: addr}
        } else {
            //FIXME Proper error handling needed
            retVal = UdpCommand{Type: "ERROR", Payload: "ERROR", Timestamp: "000", conn: conn, addr: addr}
        }
    }
    return retVal
}

////HTTP SERVER

// Defines the configuration of the http feedback server and its function
type Page struct {
    Title string
    Body []byte
}

// Load a page file from disk
func loadPage() (*Page, error) {
    filename := "static/controlpage.html"
    body, err := ioutil.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    return &Page{Title: "Controlpage", Body: body}, nil
}

// Defines the http server and its functions
type RecHttpServer struct {
    // Task manager struct
    Tq taskqueue.TaskQueue
}

type clientRequest struct {
    command string
    value string
}

// Handle Ajax requests to the /request page
func (rhttps *RecHttpServer) RequestHandler(w http.ResponseWriter, r *http.Request) {
    //Debug
    fmt.Println("Request received.")

    // Decode request from client
    decoder := json.NewDecoder(r.Body)

    var creq clientRequest
    err := decoder.Decode(&creq)

    if err != nil {
        fmt.Println(err)
    }

    // Parse command and put it on the task queue
    currCmd := parseHttpCommand(creq, w)
    rhttps.Tq.Queue <- currCmd
}

// Parse commands received via HTTP
func parseHttpCommand(creq clientRequest, hRespWriter http.ResponseWriter) HttpCommand {
    var retVal HttpCommand
    switch creq.command {
    case "req":
        switch creq.value {
        case "state":
            retVal = HttpCommand{Type: "REQ", Payload: "STATE", Timestamp: "000", HRespWriter: hRespWriter}
        default:
            //FIXME Proper error handling (using error type)
            retVal =  HttpCommand{Type: "ERROR", Payload: "ERROR", Timestamp: "000", HRespWriter: hRespWriter}
        }
    }
    return retVal
}

// Handle the static main html page
func PageHandler(w http.ResponseWriter, r *http.Request) {
    p, err := loadPage()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    fmt.Fprintf(w, "%s", p.Body)
}


// A command generated by the http server
type HttpCommand struct {
    // The type of the command
    Type string
    // The "content" of the command
    Payload string
    // The timestamp
    //TODO Use timestamp object
    Timestamp string
    // Variable that specifies the response: httpResponseWriter
    HRespWriter http.ResponseWriter
}

// The implementation of "Respond" of the Command interface for HTTP
func (cmd HttpCommand) Respond(res string) {
    // Send reply as JSON
    cmd.HRespWriter.Header().Set("Content-Type", "application/json")
    //cmd.HRepsWriter.Write([]byte("{\"state\": \""+strconv.Itoa(i)+"\"}"))
    cmd.HRespWriter.Write([]byte("{\"state\": \""+res+"\"}"))
    //TODO Proper JSON encoding
}

// The implementation of "GetPayload" of the Command interface for HTTP
func (cmd HttpCommand) GetPayload() string {
    return cmd.Payload
}

// A command generated by the udp server
type UdpCommand struct {
    // The type of the command
    Type string
    // The "content" of the command
    Payload string
    // The timestamp
    //TODO Use timestamp object
    Timestamp string
    // Variables that specifies the response: UDP connection and address
    conn *net.UDPConn
    addr *net.UDPAddr
}

// The implementation of "Respond" of the Command interface for UDP
func (cmd UdpCommand) Respond(res string) {
    _,err := cmd.conn.WriteToUDP([]byte("message"), cmd.addr)
    if err != nil {
        fmt.Printf("Error with UDP: %v",err)
    }
}

// The implementation of "GetPayload" of the Command interface for UDP
func (cmd UdpCommand) GetPayload() string {
    return cmd.Payload
}
