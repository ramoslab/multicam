// Listen and Serve: Communication with the client

package lns

import (
    "net"
    "fmt"
    "log"
    "time"
    "net/http"
    "encoding/json"
    "io/ioutil"
    "bitbucket.com/andrews2000/multicam/taskqueue"
)

//// TCP SERVER

// Defines the configuration of the server and its functions
type RecTcpServer struct {
    // TCP listener
    Conn net.Listener
    // Task manager
    Tq taskqueue.TaskQueue
    // Feedback channel from task queue
    TcpFeedback chan []byte
}

// Handles connections to clients until something is written to the quit channel
func (rtcps RecTcpServer) Run(q chan bool) {

    for {
        select {
        case <- q:
            log.Printf("INFO: Stopping TCP listener.")
        default:
            conn, errTcp := rtcps.Conn.Accept()
            if errTcp != nil {
                log.Printf("ERROR: Error accepting client via TCP. Message: %s",errTcp)
            } else {
                go handleTcpConnection(rtcps, conn)
            }
        }
    }
}

// Handles a TCP connection
func handleTcpConnection(rtcps RecTcpServer, conn net.Conn) {
    // Unmarshal what is on the buffer
    buf := make([]byte, 1024)
    for {
        n,err := conn.Read(buf)
        if err != nil {
            // When the client closes the connection an EOF error occurs. In this case the connection will be closed.
            fmt.Println("Error reading from buffer", err)
            break
        }
        var creq map[string]interface{}
        errJson := json.Unmarshal(buf[0:n], &creq)

        if errJson != nil {
            log.Printf("ERROR: Could not unmarshal udp pacakge to json; Message: %s", errJson)
        }

        // Parse command and put it on the task queue
        com := parseCommand(creq, rtcps.TcpFeedback)

        rtcps.Tq.Queue <- com

        var response []byte
        response = <-rtcps.TcpFeedback

        conn.Write(response)
    }
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
        log.Printf("ERROR: Error loading static page; Message: %s",err)
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
    Data interface{}
}

// Handle Ajax requests to the /request page
func (rhttps *RecHttpServer) RequestHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("INFO: Request received.")

    // Decode request from client
    decoder := json.NewDecoder(r.Body)

    //var creq clientRequest
    var creq map[string]interface{}
    err := decoder.Decode(&creq)

    if err != nil {
        log.Printf("ERROR: Error decoding json; Message: %s",err)
    }

    // Parse command and put it on the task queue
    currCmd := parseCommand(creq, rhttps.HttpFeedback)
    rhttps.Tq.Queue <- currCmd
    // This is used to send the http response back to the client before the client requestHandler returns
    var feedback []byte
    feedback = <-rhttps.HttpFeedback
    if len(feedback) == 0 {
        // Error state because parsing failed
        http.Error(w, "Error parsing command sent by client.", http.StatusBadRequest)
    } else {
        // Send json message back to client
        w.Header().Set("Content-Type", "application/json")
        w.Write(feedback)
    }
}

// Parse commands received via HTTP
func parseCommand(creq map[string]interface{}, httpFeedback chan []byte) taskqueue.Task {
    var retVal taskqueue.Task
    switch creq["Command"] {
    case "REQ":
        // Type assertion for Data as map
        creqData, ok := creq["Data"].(map[string]interface{})
        if !ok {
            log.Print("WARNING: Error running type assertion (REQ).")
            retVal = taskqueue.Task{Command: "ReturnError", Data: nil, FeedbackChannel: httpFeedback}
            return retVal
        }
        switch creqData["CmdType"] {
        case "GETSTATUS":
            retVal = taskqueue.Task{Command: "GetStatus", Data: nil, FeedbackChannel: httpFeedback}
        case "GETCONFIG":
            retVal = taskqueue.Task{Command: "GetConfig", Data: nil, FeedbackChannel: httpFeedback}
        default:
            log.Print("WARNING: Command not understood (REQ).")
            retVal = taskqueue.Task{Command: "ReturnError", Data: nil, FeedbackChannel: httpFeedback}
        }
    case "POST":
        // Type assertion for Data as map
        creqData, ok := creq["Data"].(map[string]interface{})
        if !ok {
            log.Print("WARNING: Error running type assertion (POST).")
            retVal = taskqueue.Task{Command: "ReturnError", Data: nil, FeedbackChannel: httpFeedback}
            return retVal
        }

        switch creqData["CmdType"] {
            case "SETCONFIG":
                payload, ok := creqData["Values"].(map[string]interface{})
                if !ok {
                    log.Printf("WARNING: Error running type assertion (POST:SETCONFIG).")
                    retVal = taskqueue.Task{Command: "ReturnError", Data: nil, FeedbackChannel: httpFeedback}
                    return retVal
                }
                retVal = taskqueue.Task{Command: "SetConfig", Data: payload, FeedbackChannel: httpFeedback}
            default:
                log.Print("WARNING: Command not understood (POST).")
                retVal = taskqueue.Task{Command: "ReturnError", Data: nil, FeedbackChannel: httpFeedback}
        }

    case "CTL":
        creqData, ok := creq["Data"].(map[string]interface{})
        if !ok {
            log.Printf("ERROR: Error running type assertion (CTL).")
            retVal = taskqueue.Task{Command: "ReturnError", Data: nil, FeedbackChannel: httpFeedback}
            return retVal
        }
        switch creqData["CmdType"] {
            case "START":
                retVal = taskqueue.Task{Command: "StartRecording", Data: nil, FeedbackChannel: httpFeedback}
            case "STOP":
                retVal = taskqueue.Task{Command: "StopRecording", Data: nil, FeedbackChannel: httpFeedback}
            default:
                log.Print("WARNING: Command not understood (CTL).")
                retVal = taskqueue.Task{Command: "ReturnError", Data: nil, FeedbackChannel: httpFeedback}
        }
    case "DATA":
        creqData, ok := creq["Data"].(map[string]interface{})
        if !ok {
            log.Printf("ERROR: Error running type assertion (DATA).")
            retVal = taskqueue.Task{Command: "ReturnError", Data: nil, FeedbackChannel: httpFeedback}
            return retVal
        }
        payload, ok := creqData["Values"].(map[string]interface{})
        if !ok {
            log.Printf("ERROR: Error running type assertion (DATA:Values)")
            retVal = taskqueue.Task{Command: "ReturnError", Data: nil, FeedbackChannel: httpFeedback}
            return retVal
        }
        // Add current time to payload
        payload["recvTime"] = time.Now()
        retVal = taskqueue.Task{Command: "Data", Data: payload, FeedbackChannel: httpFeedback}
    default:
        retVal = taskqueue.Task{Command: "ReturnError", Data: nil, FeedbackChannel: httpFeedback}
    }
    return retVal
}

// A message received from the client
type ClientMessage struct {
    Command string
    Data interface{}
}

// Data containing only request type
type DataRequest struct {
    CmdType map[string]interface{}
}

// Data containing only control type
type DataControl struct {
    CmdType string
}
