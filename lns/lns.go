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

func (rudps RecUdpServer) Run(c chan string, q chan bool, conn *net.UDPConn) {
    //if err != nil {
     //   panic(err)
    //}

    buf := make([]byte, 1024)

    for {
        select {
            case <- q:
                fmt.Println("Stopping UDP listener.")
                close(c)
                return
            default:
                n, _, err := conn.ReadFromUDP(buf)
                c <- string(buf[0:n])
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
}

type clientRequest struct {
    Request string
    Value int
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

    fmt.Printf("Decoded: %s:%d\n",creq.Request,creq.Value)

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
