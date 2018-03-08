// Listen and Serve: Communication with the client

package lns

import (
    "net"
    "fmt"
    "net/http"
    "io/ioutil"
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

func Handler(w http.ResponseWriter, r *http.Request) {
    // allow cross domain AJAX requests
    p, err := loadPage()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    fmt.Fprintf(w, "%s", p.Body)
}

func RequestHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte("{\"hello\": \"world\"}"))
}

