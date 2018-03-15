// Main control program
package main

import (
    "fmt"
    "net"
    "bitbucket.com/andrews2000/multicam/recordcontrol"
    "bitbucket.com/andrews2000/multicam/lns"
    "bitbucket.com/andrews2000/multicam/taskqueue"
    "strings"
    "net/http"
    "github.com/rs/cors"
)

func main() {
    // Start up Server


    // Instantiate record control data configuration
    cfg1 := recordcontrol.RecordConfig{Cameras: []int{1,2}, Sid: "AR", Date: "180305", Loc: "recordings"}
    // Instantiate record control data model 
    rec1 := recordcontrol.RecordControl{State: 0, Config: cfg1, VideoHwState: 0, AudioHwState: 0, DiskSpaceState: 0, SavingLocationState: 0, GstreamerState: 0}
    // Instantiate task manager 
    tq1 := taskqueue.TaskQueue{Queue: make(chan string)}
    //FIXME Immediately writing something on the task queue. If you do not do that the first command goes missing.
    //gtq1.Queue <- "Nada."

    // Instantiate the UDP Server
    serv1 := lns.RecUdpServer{Addr: net.UDPAddr{Port: 9999, IP: net.ParseIP("127.0.0.1")}}
    conn, err := net.ListenUDP("udp",&serv1.Addr)
    if err != nil {
        panic(err)
    }
    // Communication from UDP to parser 
    com := make(chan string)
    // Goroutine control channel
    qudp := make(chan bool)

    // Instantiate the HTTP Server
    // Communication from task manager to HTTP server
    cfb := make(chan int)
    serv2 := lns.RecHttpServer{Tq: tq1, Cfb: cfb, Com: com}

    mux := http.NewServeMux()

    mux.HandleFunc("/request", serv2.RequestHandler)
    mux.HandleFunc("/", lns.PageHandler)
    mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

    co := cors.New(cors.Options{
        AllowedOrigins: []string{"*"},
        AllowCredentials: true,
        AllowedMethods: []string{"GET","POST"},
    })

    handler := co.Handler(mux)

    //DEBUG
    fmt.Println("Current state: ",rec1.GetState())

    // Start the routine that listens over UDP
    go serv1.Run(com,qudp, conn)
    // Start the routine that serves HTTP
    go http.ListenAndServe(":8040", handler)
    // Start the task management routine
    go tq1.ExecuteTask(&rec1, cfb)

    // Parse commands that are written to the command channel
    for str := range com {
        parseCommand(str,&rec1,qudp,conn,tq1)
    }

    //DEBUG
    fmt.Println("Current state: ",rec1.GetState())
}

// Parse commands being received via UDP and initiate execution of the commands
//FIXME How do we know the source of the command: Maybe by instead of using strings to store the commands using some sort of a struct
func parseCommand(cmd string, rc *recordcontrol.RecordControl, qudp chan bool, conn *net.UDPConn, tq taskqueue.TaskQueue) {
    spl := strings.Split(cmd, ":")
    if len(spl) == 3 {
        switch spl[0] {
        case "CTL":
            fmt.Println("Control command received: "+spl[2])
            switch spl[2] {
            case "START":
                fmt.Println("Writing \"Start\" to queue")
                tq.Queue <- "Start"
            case "STOP":
                tq.Queue <- "Stop"
            case "PREPARE":
                tq.Queue <- "Prepare"
            default:
                fmt.Println("Ignoring invalid command: "+spl[2])
            }
        case "DAT":
            fmt.Println("Data received.")
        case "REQ":
            fmt.Println("Request received.")
        default:
            fmt.Println("Invalid command received.")
        }
    } else {
        if spl[0] == "EXIT" {
            stopAndExit(qudp,conn)
        } else {
            fmt.Println("Invalid command received.")
        }
    }
}

//TODO It remains an open question of the stopAndExit procedure should remain here or go to the TaskQueue

// Stop UDP Server and exit
//FIXME Needs to stop the recording as well
//FIXME Needs to stop the task manager as well (or does it?)
func stopAndExit(qudp chan bool, conn *net.UDPConn) {
    fmt.Println("Closing shutdown channel.")
    close(qudp)
    fmt.Println("Closing UDP connection.")
    conn.Close()
    //fmt.Println("Stopping the recording.")
}
