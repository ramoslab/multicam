// Main control program
package main

import (
    "net"
    "strconv"
    "bitbucket.com/andrews2000/multicam/recordcontrol"
    "bitbucket.com/andrews2000/multicam/lns"
    "bitbucket.com/andrews2000/multicam/taskqueue"
    "net/http"
    "github.com/rs/cors"
    "github.com/spf13/viper"
    "log"
    "os"
)

func main() {
    // Set up log
    f, err := os.OpenFile("multicam.log",os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)

    if err != nil {
        log.Fatal(err)
    }

    defer f.Close()

    log.SetOutput(f)
    log.SetFlags(log.Ldate|log.Ltime|log.Lshortfile)

    log.Print("###### STARTING UP ######")
    log.Print("INFO: Reading config.")

    // Start up Server

    // Read configuration
    viper.SetConfigName("multicam_config")
    viper.AddConfigPath(".")

    err = viper.ReadInConfig()
    if err != nil {
        log.Fatalf("Problem reading config: %s",err)
    }

    // Get configuration for the recording
    sid := viper.GetString("Recording.SID")
    recfolder := viper.GetString("Recording.RecFolder")
    cams_cfg := viper.Get("Recording.Cameras").([]interface{})
    cams := make([]int, len(cams_cfg))
    for i,cam := range cams_cfg {
        var ok bool
        cams[i], ok = cam.(int)

        if !ok {
            log.Print("ERROR: Type assertion failed during parsing of configuration.")
        }
    }

    mics_cfg := viper.Get("Recording.Microphones").([]interface{})
    mics := make([]int, len(mics_cfg))
    for i,mic := range mics_cfg {
        var ok bool
        mics[i], ok = mic.(int)

        if !ok {
            log.Print("ERROR: Type assertion failed during parsing of configuration.")
        }

    }

    // Get configuration for the microphones
    searchStringAudio := viper.GetString("Hardware.SearchStringAudio")

    // Get configuration for the server
    port := viper.GetInt("Server.Port")
    address := viper.GetString("Server.Address")

    log.Print("INFO: Starting server.")

    // Instantiate record control data configuration
    recCfg := recordcontrol.RecordConfig{Cameras: cams, Microphones: mics, Sid: sid, RecFolder: recfolder}
    // Instantiate record control data model 
    rec1 := recordcontrol.RecordControl{Config: recCfg, SearchStringAudio: searchStringAudio}
    rec1.Preflight()
    // Instantiate task manager 
    tq1 := taskqueue.TaskQueue{Queue: make(chan taskqueue.Task)}

    // Instantiate the TCP Server

    l, err := net.Listen("tcp",address+":"+strconv.Itoa(port))

    if err != nil {
        log.Fatalf("FATAL: Could not create TCP server. Message: %s",err)
    }

    defer l.Close()

    tcpFeedback := make(chan []byte)

    //serveUdp := lns.RecUdpServer{Conn: serveUdp_conn, Addr: &serveUdp_addr, Tq: tq1, UdpFeedback: udpFeedback}
    serveTcp := lns.RecTcpServer{Conn: l, Tq: tq1, TcpFeedback: tcpFeedback}

    // Goroutine control channel (for ending goroutine)
    qtcp := make(chan bool)

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

    // Start the routine that listens over UDP
    go serveTcp.Run(qtcp)
    // Start the routine that serves HTTP
    go http.ListenAndServe(":8040", handler)
    // Start the task management routine
    tq1.ExecuteTask(&rec1)
}
