// A simple taskmanager that operates on a channel
package taskqueue

import (
    "bitbucket.com/andrews2000/multicam/recordcontrol"
    "fmt"
)

type TaskQueue struct {
    Queue chan string
    Stopping chan bool
}

// Execute tasks until stopping channel is true
func (tq TaskQueue) executeTask(task string, rc *recordcontrol.RecordControl) {
    for {
        select {
        case <- tq.Stopping:
            fmt.Println("Stopping Task manager.")
            return
        case <- tq.Queue:
            for str := range tq.Queue {
                switch str {
                case "Pepare":
                    execPrepare(rc)
                case "Start":
                    execStartRecording(rc)
                case "Stop":
                    execStopRecording(rc)
                }
            }
        }
    }
}

// Actual tasks

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
