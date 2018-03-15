// Listen and Serve: Communication with the client

package lns

import (
    "net"
    "fmt"
    "net/http"
    "encoding/json"
    "io/ioutil"
    "strconv"
    "bitbucket.com/andrews2000/multicam/taskqueue"
)

// Defines the configuration of the server and its functions
type RecUdpServer struct {
    Addr net.UDPAddr
}

func (rudps RecUdpServer) Run(com chan string, q chan bool, conn *net.UDPConn) {

    buf := make([]byte, 1024)

    for {
        select {
            case <- q:
                fmt.Println("Stopping UDP listener.")
                close(com)
                return
            default:
                n, _, err := conn.ReadFromUDP(buf)
                com <- string(buf[0:n])
                fmt.Println("Received ", string(buf[0:n]))

                if err != nil {
                    fmt.Println("Error: ", err)
                }
        }
    }
}

// Defines the configurtion of the http feedback server and its function
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
    // Feedback channel
    Cfb chan int
    // Command channel
    Com chan string
}

type clientRequest struct {
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

    // Send command to command channel
    //FIXME add source of the command to the command
    //TODO add client time of the request
    rhttps.Com <- fmt.Sprintf("%s:%i:%s",creq.Command,0,creq.Value)

    // Start task appropriate to the request
    rhttps.Tq.Queue <- "State"

    // Wait for task execution
    i := <-rhttps.Cfb
    // Send reply as JSON
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte("{\"state\": \""+strconv.Itoa(i)+"\"}"))
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
