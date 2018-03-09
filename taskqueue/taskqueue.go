// A simple taskmanager that operates on a channel
package taskqueue

import (
    "bitbucket.com/andrews2000/multicam/recordcontrol"
    "fmt"
)

type TaskQueue struct {
    Queue chan string
}

// Execute tasks until stopping channel is true
func (tq TaskQueue) ExecuteTask(rc *recordcontrol.RecordControl, cfb chan int) {
    for {
        str := <-tq.Queue
        switch str {
        case "Pepare":
            execPrepare(rc)
        case "Start":
            execStartRecording(rc)
        case "Stop":
            execStopRecording(rc)
        case "State":
            i := getState(rc)
            cfb <- i
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
