// Listen and Serve: Communication with the client

package lns

import (
    "net"
    "fmt"
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
                n, _, errUdp := rudps.Conn.ReadFromUDP(buf)

                //FIXME Error handling
                if errUdp != nil {
                    fmt.Println("Error: ", errUdp)
                }

                // Unmarshal what is on the buffer
                var creq map[string]interface{}
                errJson := json.Unmarshal(buf[0:n], &creq)

                //FIXME Error handling
                if errJson != nil {
                    fmt.Println("Error: ", errJson)
                }

                // Parse command and put it on the task queue
                com := parseUdpCommand(creq, rudps.UdpFeedback)
                rudps.Tq.Queue <- com

                //DEBUG
                fmt.Println("Received ", string(buf[0:n]))
                //rudps.Conn.WriteToUDP([]byte("Great"), rudps.Addr)

                // Send response to client
                //NOTE The response should usually go unheard because if the package gets lost the client will likely block while waiting for the package to arrive
                //response := <-rudps.UdpFeedback
                //DEBUG
                //fmt.Println("Response:",response)

                //_,err = rudps.Conn.WriteToUDP([]byte(response), addr)

                //if err != nil {
                //    fmt.Println(err)
                //}
        }
    }
}

//FIXME One function for parseCommands insted of UDP and HTTP separately
// Parse commands being received via UDP and initiate execution of the commands
func parseUdpCommand(creq map[string]interface{}, udpFeedback chan []byte) taskqueue.Task {
    var retVal taskqueue.Task
    switch creq["Command"] {
    case "REQ":
        // Type assertion for Data as map
        creqData, ok := creq["Data"].(map[string]interface{})
        if !ok {
            //FIXME Error handling
            fmt.Println("Error running type assertion.")
        }
        switch creqData["CmdType"] {
        case "GETSTATE":
            retVal = taskqueue.Task{Command: "GetState", Data: nil, FeedbackChannel: udpFeedback}
        case "GETCONFIG":
            retVal = taskqueue.Task{Command: "GetConfig", Data: nil, FeedbackChannel: udpFeedback}
                    //var data taskqueue.ConfigStruct
                    //if err := json.Unmarshal(cmData, &data); err != nil {
                    //    fmt.Println(err)
                    //    //FIXME Proper error handling
                    //    retVal = taskqueue.Task{Command: "ReturnError", Data: err, FeedbackChannel: httpFeedback}
                    //} else {
                    //    //TODO properly set data
                    //    retVal = taskqueue.Task{Command: "SetConfig", Data: nil, FeedbackChannel: httpFeedback}
                    //}
                default:
                //FIXME Proper error handling (using error type)
                retVal = taskqueue.Task{Command: "ReturnError", Data: nil, FeedbackChannel: udpFeedback}
            }
        case "CTL":
            creqData, ok := creq["Data"].(map[string]interface{})
            if !ok {
                //FIXME Error handling
                fmt.Println("Error running type assertion.")
            }
            switch creqData["CmdType"] {
                //TODO
                case "PREPARE":
                    retVal = taskqueue.Task{Command: "Preflight", Data: nil, FeedbackChannel: udpFeedback}
                case "START":
                    retVal = taskqueue.Task{Command: "StartRecording", Data: nil, FeedbackChannel: udpFeedback}
                case "STOP":
                    retVal = taskqueue.Task{Command: "StopRecording", Data: nil, FeedbackChannel: udpFeedback}
            } //TODO Implement data
        case "DAT":
            fmt.Println("Data received.")
            //TODO Implement
        default:
            //FIXME Proper error handling needed
            fmt.Println("Invalid command received.")
        }
    return retVal
    //TODO Implement exit (and maybe even shutdown commands
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
    Data interface{}
}

// Handle Ajax requests to the /request page
func (rhttps *RecHttpServer) RequestHandler(w http.ResponseWriter, r *http.Request) {
    //Debug
    fmt.Println("Request received.")

    // Decode request from client
    decoder := json.NewDecoder(r.Body)

    //var creq clientRequest
    var creq map[string]interface{}
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
//func parseHttpCommand(creq clientRequest, hRespWriter *http.ResponseWriter, httpFeedback chan []byte) taskqueue.Task {
func parseHttpCommand(creq map[string]interface{}, hRespWriter *http.ResponseWriter, httpFeedback chan []byte) taskqueue.Task {
    var retVal taskqueue.Task
    switch creq["Command"] {
    case "REQ":
        // Type assertion for Data as map
        creqData, ok := creq["Data"].(map[string]interface{})
        if !ok {
            //FIXME Error handling
            fmt.Println("Error running type assertion.")
        }
        switch creqData["CmdType"] {
        case "GETSTATUS":
            retVal = taskqueue.Task{Command: "GetStatus", Data: nil, FeedbackChannel: httpFeedback}
        case "GETCONFIG":
            //var data taskqueue.ConfigStruct
            //if err := json.Unmarshal(creq.Data[1], &data); err != nil {
            //    fmt.Println(err)
            //    //FIXME Proper error handling
            //    retVal = taskqueue.Task{Command: "ReturnError", Data: err, FeedbackChannel: httpFeedback}
            //}
            retVal = taskqueue.Task{Command: "GetConfig", Data: nil, FeedbackChannel: httpFeedback}
        default:
            //FIXME Proper error handling (using error type)
                retVal = taskqueue.Task{Command: "ReturnError", Data: nil, FeedbackChannel: httpFeedback}
        }
    case "POST":
        // Type assertion for Data as map
        creqData, ok := creq["Data"].(map[string]interface{})
        if !ok {
            //FIXME Error handling
            fmt.Println("Error running type assertion.")
        }
        switch creqData["CmdType"] {
            case "SETCONFIG":
                payload, ok := creqData["Values"].(map[string]interface{})
                fmt.Println(payload["Cameras"])
                if ok {
                    fmt.Println(payload)
                } else {
                    fmt.Println("CONFIG not ok")
                }
                retVal = taskqueue.Task{Command: "SetConfig", Data: payload, FeedbackChannel: httpFeedback}
                //FIXME Enter proper command here
        }

    case "CTL":
        creqData, ok := creq["Data"].(map[string]interface{})
        if !ok {
            //FIXME Error handling
            fmt.Println("Error running type assertion.")
        }
        switch creqData["CmdType"] {
            //TODO
            case "PREPARE":
                retVal = taskqueue.Task{Command: "Preflight", Data: nil, FeedbackChannel: httpFeedback}
            case "START":
                retVal = taskqueue.Task{Command: "StartRecording", Data: nil, FeedbackChannel: httpFeedback}
            case "STOP":
                retVal = taskqueue.Task{Command: "StopRecording", Data: nil, FeedbackChannel: httpFeedback}
        }
        //TODO Implement data
    }
    return retVal
}

// The implementation of "RespondMessage" of the Command interface for HTTP
//TODO Remove this
//func (cmd HttpCommand) RespondMessage(msg taskqueue.Message) {
//    // Marshal message into byte array
//    res, _ := json.Marshal(msg)
//    // Write response string to the channel
//    cmd.HttpFeedback <- res
//    //TODO Make response structs
//    fmt.Println(res)
//}

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

// Data containing configuration

