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
    "github.com/spf13/viper"
)

func main() {
    // Start up Server

    // Read configuration
    viper.SetConfigName("multicam_config")
    viper.AddConfigPath(".")

    //DEBUG
    err := viper.ReadInConfig()
    if err != nil {
        panic(fmt.Errorf("Problem reading config:",err))
    }

    // Get configuration for the recording
    sid := viper.GetString("Recording.SID")
    recfolder := viper.GetString("Recording.RecFolder")
    cams_cfg := viper.Get("Recording.Cameras").([]interface{})
    cams := make([]int, len(cams_cfg))
    for i,cam := range cams_cfg {
        cams[i] = cam.(int)
    }

    // Get configuration for the server
    port := viper.GetInt("Server.Port")
    address := viper.GetString("Server.Adress")


    // Instantiate record control data configuration
    recCfg := recordcontrol.RecordConfig{Cameras: cams, Sid: sid, RecFolder: recfolder}
    //DEBUG
    fmt.Println(recCfg)
    // Instantiate record control data model 
    rec1 := recordcontrol.RecordControl{StateId: 0, Config: recCfg}
    rec1.Preflight()
    fmt.Println(rec1)
    // Instantiate task manager 
    tq1 := taskqueue.TaskQueue{Queue: make(chan taskqueue.Command)}
    //FIXME Immediately writing something on the task queue. If you do not do that the first command goes missing.
    //gtq1.Queue <- "Nada."

    // Instantiate the UDP Server
    serveUdp_addr := net.UDPAddr{Port: port, IP: net.ParseIP(address)}
    serveUdp_conn, err := net.ListenUDP("udp",&serveUdp_addr)

    if err != nil {
        panic(err)
        //FIXME Proper error handling
    }

    udpFeedback := make(chan []byte)

    serveUdp := lns.RecUdpServer{Conn: serveUdp_conn, Addr: &serveUdp_addr, Tq: tq1, UdpFeedback: udpFeedback}

    // Goroutine control channel (for ending goroutine)
    qudp := make(chan bool)

    // Instantiate the HTTP Server

    httpFeedback := make(chan []byte)

    serveHttp := lns.RecHttpServer{Tq: tq1, HttpFeedback: httpFeedback}

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
