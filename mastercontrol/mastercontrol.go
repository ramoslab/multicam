// Main control program
package main

import (
    "fmt"
    "net"
    "bitbucket.com/andrews2000/multicam/recordcontrol"
    "bitbucket.com/andrews2000/multicam/lns"
    "bitbucket.com/andrews2000/multicam/taskqueue"
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
    tq1 := taskqueue.TaskQueue{Queue: make(chan taskqueue.Command)}
    //FIXME Immediately writing something on the task queue. If you do not do that the first command goes missing.
    //gtq1.Queue <- "Nada."

    // Instantiate the UDP Server
    serveUdp_addr := net.UDPAddr{Port: 9999, IP: net.ParseIP("127.0.0.1")}
    serveUdp_conn, err := net.ListenUDP("udp",&serveUdp_addr)

    if err != nil {
        panic(err)
        //FIXME Proper error handling
    }

    serveUdp := lns.RecUdpServer{Conn: serveUdp_conn, Addr: &serveUdp_addr, Tq: tq1}

    // Communication from UDP to parser 
    //com := make(chan string)
    // Goroutine control channel (for ending goroutine)
    qudp := make(chan bool)

    // Instantiate the HTTP Server
    serveHttp := lns.RecHttpServer{Tq: tq1}

    mux := http.NewServeMux()

    mux.HandleFunc("/request", serveHttp.RequestHandler)
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
    go serveUdp.Run(qudp)
    // Start the routine that serves HTTP
    go http.ListenAndServe(":8040", handler)
    // Start the task management routine
    tq1.ExecuteTask(&rec1)

    //DEBUG
    fmt.Println("Current state: ",rec1.GetState())
}

//TODO It remains an open question if the stopAndExit procedure should remain here or go to the TaskQueue

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
