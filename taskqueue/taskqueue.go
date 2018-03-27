// A simple taskmanager that operates on a channel
package taskqueue

import (
    "bitbucket.com/andrews2000/multicam/recordcontrol"
    "fmt"
    "strconv"
)

type TaskQueue struct {
    Queue chan Command
}

// Execute tasks until stopping channel is true
func (tq TaskQueue) ExecuteTask(rc *recordcontrol.RecordControl) {
    for {
        cmd := <-tq.Queue
        cmdType := cmd.GetType()
        cmdPayload := cmd.GetPayload()

        fmt.Println("TQ: "+cmdType, cmdPayload)

        switch cmdType {
        case "CTL":
            switch cmdPayload {
            case "PREPARE":
                execPrepare(rc)
                cmd.Respond("OK:PREPARE")
            case "START":
                execStartRecording(rc)
                cmd.Respond("OK:START")
            case "STOP":
                execStopRecording(rc)
                cmd.Respond("OK:STOP")
            default:
                fmt.Println("TQ: REQ[unknown] "+cmdPayload)
                cmd.Respond("NOTOK:ERROR")
        }
        case "REQ":
            switch cmdPayload {
            case "STATE":
                cmd.Respond(strconv.Itoa(rc.GetState()))
            default:
                fmt.Println("TQ: REQ[unknown] "+cmdPayload)
                cmd.Respond("NOTOK:ERROR")
            }
        //case "DATA":
        case "ERROR":
            fmt.Println("TQ: ERROR received. "+cmdPayload)
            cmd.Respond("NOTOK:ERROR")
        default:
            fmt.Println("TQ: TYPE[unknown] "+cmdType)
            cmd.Respond("NOTOK:ERROR")
        }
    }
}
// Actual task
func getState(rc *recordcontrol.RecordControl) (int){
    return rc.GetState()
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
        fmt.Println("Record control not idle.")
    }
    fmt.Println("Current state: ",rc.GetState())
}

// Stop recording
func execStopRecording(rc *recordcontrol.RecordControl) {
    if rc.GetState() == 2 {
        fmt.Println("Stopping the recording.")
        rc.StopRecording()
    } else {
        fmt.Println("Cannot stop recording because the server is currently not recording.")
    }
    fmt.Println("Current state: ",rc.GetState())
}

// Task Helpers

// Check current status of the server. If idle the script is executed.
func recCtrlIdle(rc *recordcontrol.RecordControl) bool {
    if rc.GetState() == 0 {
        return true
    } else {
        return false
    }
}

// The interface for command structures
type Command interface {
    // This function returns a string to the client through the appropriate channel
    Respond(Str string)
    GetType() string
    GetPayload() string
}
