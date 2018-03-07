// Main control program
package main

import (
    "fmt"
    "net"
    "bitbucket.com/andrews2000/recordcontrol"
    "bitbucket.com/andrews2000/lns"
    "strings"
    "time"
)

func main() {
    // Start up Server
    // Instantiate configuration struct
    cfg1 := recordcontrol.RecordConfig{Cameras: []int{1,2}, Sid: "AR", Date: "180305", Loc: "recordings"}
    // Instantiate RecordControl struct 
    rec1 := recordcontrol.RecordControl{State: 0, Config: cfg1, VideoHwState: 0, AudioHwState: 0, DiskSpaceState: 0, SavingLocationState: 0, GstreamerState: 0}
    // Instantiate RecUdpServer struct
    serv1 := lns.RecUdpServer{Addr: net.UDPAddr{Port: 9999, IP: net.ParseIP("127.0.0.1")}}
    // Data channel
    c := make(chan string)
    // Goroutine control channel
    q := make(chan bool)

    conn, err := net.ListenUDP("udp",&serv1.Addr)
    conn.SetReadDeadline(time.Now().Add(1 * time.Second))

    if err != nil {
        panic(err)
    }


    fmt.Println(rec1.GetState())

    go serv1.Run(c,q, conn)
    go serv1.Test(c,q)

    for str := range c {
        parseCommand(str,rec1,q,conn)
    }

    fmt.Println(rec1.GetState())
}

// Parse commands being received via UDP and initiate execution of the commands
func parseCommand(cmd string, rc recordcontrol.RecordControl, q chan bool, conn *net.UDPConn) {
    spl := strings.Split(cmd, ":")
    if len(spl) == 3 {
        switch spl[0] {
            case "CTL":
                fmt.Println("Control command received.")
                if spl[2] == "START" {
                    execStartRecording(rc)
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

// Execute preparation command (Perform all necessary checks of record control
func execPrepare(rc recordcontrol.RecordControl) {

}

// Set configuration of record control
func setRecordControlConfig(rc *recordcontrol.RecordControl) {

}

// Start recording
func execStartRecording(rc recordcontrol.RecordControl) {
    rc.StartRecording()
    fmt.Println(rc.GetState())
}

// Stop recording
func execStopRecording(rc recordcontrol.RecordControl) {

}

// Stop UDP Server and exit
func stopAndExit(q chan bool, conn *net.UDPConn) {
    fmt.Println("Shutting down server.")
    close(q)
    conn.Close()
    //q <- true
    fmt.Println("Kill signal sent.")
}
