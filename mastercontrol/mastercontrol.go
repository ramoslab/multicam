// Main control program
package main

import (
    "fmt"
    "net"
    "bitbucket.com/andrews2000/multicam/recordcontrol"
    "bitbucket.com/andrews2000/multicam/lns"
    "strings"
    "net/http"
    "github.com/rs/cors"
)

func main() {
    // Start up Server
    // Instantiate configuration struct
    cfg1 := recordcontrol.RecordConfig{Cameras: []int{1,2}, Sid: "AR", Date: "180305", Loc: "recordings"}
    // Instantiate RecordControl struct 
    rec1 := recordcontrol.RecordControl{State: 0, Config: cfg1, VideoHwState: 0, AudioHwState: 0, DiskSpaceState: 0, SavingLocationState: 0, GstreamerState: 0}
    // Instantiate RecUdpServer struct and start listening to UDP connection
    serv1 := lns.RecUdpServer{Addr: net.UDPAddr{Port: 9999, IP: net.ParseIP("127.0.0.1")}}
    conn, err := net.ListenUDP("udp",&serv1.Addr)
    if err != nil {
        panic(err)
    }
    // Data channel
    c := make(chan string)
    //defer close(c)
    // Goroutine control channel
    q := make(chan bool)

    fmt.Println("Current state: ",rec1.GetState())

    // Start the routine that listens over UDP
    go serv1.Run(c,q, conn)

    // Configure and start the routine that listens over HTTP
    serv2 := lns.RecHttpServer{Rec: &rec1}

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

    go http.ListenAndServe(":8040", handler)

    // Parse commands that are written to the command channel
    for str := range c {
        parseCommand(str,&rec1,q,conn)
    }

    fmt.Println("Current state: ",rec1.GetState())
}

// Parse commands being received via UDP and initiate execution of the commands
func parseCommand(cmd string, rc *recordcontrol.RecordControl, q chan bool, conn *net.UDPConn) {
    spl := strings.Split(cmd, ":")
    if len(spl) == 3 {
        switch spl[0] {
        case "CTL":
            fmt.Println("Control command received.")
            switch spl[2] {
            case "START":
                execStartRecording(rc)
            case "STOP":
                execStopRecording(rc)
            case "PREPARE":
                execPrepare(rc)
            }
        case "DAT":
            fmt.Println("Data received.")
        default:
            fmt.Println("Invalid command received.")
        }
    } else {
        if spl[0] == "EXIT" {
            stopAndExit(q,conn)
        } else {
            fmt.Println("Invalid command received.")
        }
    }
}

// Check current status of the server. If idle the script is executed.
func recCtrlIdle(rc *recordcontrol.RecordControl) bool {
    if rc.GetState() == 0 {
        return true
    } else {
        return false
    }
}

// Execute preparation command (Perform all necessary checks of record control
func execPrepare(rc *recordcontrol.RecordControl) {
    if recCtrlIdle(rc) {
        fmt.Println("Running preflight...")
    } else {
        fmt.Println("Record control not ready for preparation.")
    }
}

// Set configuration of record control
func setRecordControlConfig(rc *recordcontrol.RecordControl) {

}

// Start recording
func execStartRecording(rc *recordcontrol.RecordControl) {
    if recCtrlIdle(rc) {
        fmt.Println("Starting the recording")
        rc.StartRecording()
    } else {
        fmt.Println("Record control not ready for recording.")
    }
    fmt.Println("Current state: ",rc.GetState())
}

// Stop recording
func execStopRecording(rc *recordcontrol.RecordControl) {
    rc.StopRecording()

}

// Stop UDP Server and exit
//FIXME Needs to stop the recording as well
func stopAndExit(q chan bool, conn *net.UDPConn) {
    fmt.Println("Closing shutdown channel.")
    close(q)
    fmt.Println("Closing UDP connection.")
    conn.Close()
    //fmt.Println("Stopping the recording.")
}
