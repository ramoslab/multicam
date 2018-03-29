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
    // Feedback channel from task queue
    UdpFeedback chan []byte
}

func (rudps RecUdpServer) Run(q chan bool) {

    buf := make([]byte, 1024)

    for {
        select {
            case <- q:
                fmt.Println("Stopping UDP listener.")
            default:
                // Get number of bytes on the buffer and client address
                //n, addr, err := rudps.Conn.ReadFromUDP(buf)
                n, _, err := rudps.Conn.ReadFromUDP(buf)

                if err != nil {
                    fmt.Println("Error: ", err)
                }
                // Parse command and put it on the task queue
                com := parseUdpCommand(string(buf[0:n]), rudps.UdpFeedback)
                rudps.Tq.Queue <- com

                //DEBUG
                fmt.Println("Received ", string(buf[0:n]))

                // Send response to client
                //NOTE The response should usually go unheard because if the package gets lost the client will likely block while waiting for the package to arrive
                response := <-rudps.UdpFeedback
                //DEBUG
                fmt.Println("Response:",response)

                //_,err = rudps.Conn.WriteToUDP([]byte(response), addr)

                if err != nil {
                    fmt.Println(err)
                }
        }
    }
}

// Parse commands being received via UDP and initiate execution of the commands
func parseUdpCommand(cmd string, udpFeedback chan []byte) UdpCommand {
    var retVal UdpCommand
    spl := strings.Split(cmd, ":")
    if len(spl) == 3 {
        switch spl[0] {
        case "CTL":
            fmt.Println("Control command received: "+spl[2])
            switch spl[2] {
            case "START":
                fmt.Println("Writing \"Start\" to queue")
                retVal = UdpCommand{Type: "CTL", Payload: "START", Timestamp: "000", UdpFeedback: udpFeedback}
            case "STOP":
                retVal = UdpCommand{Type: "CTL", Payload: "STOP", Timestamp: "000", UdpFeedback: udpFeedback}
            case "PREPARE":
                retVal = UdpCommand{Type: "CTL", Payload: "PREPARE", Timestamp: "000", UdpFeedback: udpFeedback}
            default:
                retVal = UdpCommand{Type: "ERROR", Payload: "ERROR", Timestamp: "000", UdpFeedback: udpFeedback}
                fmt.Println("Ignoring invalid command: "+spl[2])
            }
        case "DAT":
            fmt.Println("Data received.")
            retVal = UdpCommand{Type: "DATA", Payload: "xxx", Timestamp: "000", UdpFeedback: udpFeedback}
        case "REQ":
            fmt.Println("Request received.")
            retVal = UdpCommand{Type: "REQ", Payload: "REQ", Timestamp: "000", UdpFeedback: udpFeedback}
            // Ausformulieren
        default:
            //FIXME Proper error handling needed
            fmt.Println("Invalid command received.")
            retVal = UdpCommand{Type: "ERROR", Payload: "ERROR", Timestamp: "000", UdpFeedback: udpFeedback}
        }
    } else {
        if spl[0] == "EXIT" {
            //stopAndExit(qudp,conn)
            //TODO Exit as a command
            retVal = UdpCommand{Type: "EXIT", Payload: "EXIT", Timestamp: "000", UdpFeedback: udpFeedback}
        } else {
            //FIXME Proper error handling needed
            retVal = UdpCommand{Type: "ERROR", Payload: "ERROR", Timestamp: "000", UdpFeedback: udpFeedback}
        }
    }
    return retVal
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
    // Feedback channel for the response of the client
    UdpFeedback chan []byte
}

// The implementation of "RespondMessage" of the Command interface for HTTP
func (cmd UdpCommand) RespondMessage(msg taskqueue.Message) {
    // Marshal message into byte array
    res, _ := json.Marshal(msg)
    // Write response string to the channel
    cmd.UdpFeedback <- res
    //TODO Make response structs
    fmt.Println(res)
}

func (cmd UdpCommand) RespondState(state taskqueue.State) {
    // Marshal message into byte array
    res, _ := json.Marshal(state)
    // Write response string to the channel
    cmd.UdpFeedback <- res
    //TODO Make response structs
    fmt.Println(res)
}

func (cmd UdpCommand) RespondError(msgerr taskqueue.Error) {
    // Marshal message into byte array
    res, _ := json.Marshal(msgerr)
    // Write response string to the channel
    cmd.UdpFeedback <- res
    //TODO Make response structs
    fmt.Println(res)
}

func (cmd UdpCommand) RespondConfig(msgcfg taskqueue.Config) {
    // Marshal message into byte array
    res, _ := json.Marshal(msgcfg)
    // Write response string to the channel
    cmd.UdpFeedback <- res
    //TODO Make response structs
    fmt.Println(res)
}

// The implementation of "GetPayload" of the Command interface for UDP
func (cmd UdpCommand) GetPayload() string {
    return cmd.Payload
}

// The implementation of "GetType" of the Command interface forUDP
func (cmd UdpCommand) GetType() string {
    return cmd.Type
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

// Handle the static main html page
func PageHandler(w http.ResponseWriter, r *http.Request) {
    p, err := loadPage()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    fmt.Fprintf(w, "%s", p.Body)
}

// Defines the http server and its functions
type RecHttpServer struct {
    // Task manager struct
    Tq taskqueue.TaskQueue
    // Feedback channel from task queue
    HttpFeedback chan []byte
}

type clientRequest struct {
    //Remember: JSON decoder only fills exported fields of struct (upper-case first letter)
    Command string
    Value string
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
    currCmd := parseHttpCommand(creq, &w, rhttps.HttpFeedback)
    rhttps.Tq.Queue <- currCmd
    // This is used to send the http response back to the client before the client requestHandler returns
    var feedback []byte
    feedback = <-rhttps.HttpFeedback
    //TODO Timeout for HTTP respones if nothing is on the channel after a while (e.g. if parsing the command fails or so)
    w.Header().Set("Content-Type", "application/json")
    w.Write(feedback)

}

// Parse commands received via HTTP
func parseHttpCommand(creq clientRequest, hRespWriter *http.ResponseWriter, httpFeedback chan []byte) HttpCommand {
    var retVal HttpCommand
    switch creq.Command {
    case "REQ":
        switch creq.Value {
        case "STATE":
            retVal = HttpCommand{Type: "REQ", Payload: "STATE", Timestamp: "000", HttpFeedback: httpFeedback}
        case "CONFIG":
            retVal = HttpCommand{Type: "REQ", Payload: "CONFIG", Timestamp: "000", HttpFeedback: httpFeedback}
        default:
            //FIXME Proper error handling (using error type)
            retVal = HttpCommand{Type: "ERROR", Payload: "ERROR", Timestamp: "000", HttpFeedback: httpFeedback}
        }
    }
    return retVal
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
    // Feedback channel for the response to the client
    HttpFeedback chan []byte
}

// The implementation of "RespondMessage" of the Command interface for HTTP
func (cmd HttpCommand) RespondMessage(msg taskqueue.Message) {
    // Marshal message into byte array
    res, _ := json.Marshal(msg)
    // Write response string to the channel
    cmd.HttpFeedback <- res
    //TODO Make response structs
    fmt.Println(res)
}

func (cmd HttpCommand) RespondState(state taskqueue.State) {
    // Marshal message into byte array
    res, _ := json.Marshal(state)
    // Write response string to the channel
    cmd.HttpFeedback <- res
    //TODO Make response structs
    fmt.Println(res)
}

func (cmd HttpCommand) RespondError(msgerr taskqueue.Error) {
    // Marshal message into byte array
    res, _ := json.Marshal(msgerr)
    // Write response string to the channel
    cmd.HttpFeedback <- res
    //TODO Make response structs
    fmt.Println(res)
}

func (cmd HttpCommand) RespondConfig(msgcfg taskqueue.Config) {
    // Marshal message into byte array
    res, _ := json.Marshal(msgcfg)
    // Write response string to the channel
    cmd.HttpFeedback <- res
    //TODO Make response structs
    fmt.Println(res)
}

// The implementation of "GetPayload" of the Command interface for HTTP
func (cmd HttpCommand) GetPayload() string {
    return cmd.Payload
}

// The implementation of "GetType" of the Command interface for HTTP
func (cmd HttpCommand) GetType() string {
    return cmd.Type
}

