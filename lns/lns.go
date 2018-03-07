// Listen and Serve: Communication with the client

package lns

import (
    "net"
    "fmt"
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
