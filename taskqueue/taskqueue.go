// A simple taskmanager that operates on a channel
package taskqueue

import (
    "bitbucket.com/andrews2000/multicam/recordcontrol"
    "log"
    "time"
)

type TaskQueue struct {
    Queue chan Task
}

// Execute tasks until stopping channel is true
func (tq TaskQueue) ExecuteTask(rc *recordcontrol.RecordControl) {
    for {
        cmd := <-tq.Queue
        cmdType := cmd.Command

        log.Printf("INFO: Executing task on taskqueue: %s",cmdType)

        switch cmdType  {
        case "GetStatus":
            cmd.FeedbackChannel <- rc.TaskGetStatus()
        case "GetConfig":
            cmd.FeedbackChannel <- rc.TaskGetConfig()
        case "SetConfig":
            data := cmd.Data.(map[string]interface{})
            cams_cfg, cams_ok := data["Cameras"].([]interface{})
            mics_cfg, mics_ok := data["Microphones"].([]interface{})
            if !cams_ok || !mics_ok  {
                log.Print("WARNING: Error running type assertion (TQ:SetConfig).")
                cmd.FeedbackChannel <- []byte("")
            } else {
                cams := make([]int, len(cams_cfg))
                mics := make([]int, len(mics_cfg))
                for i,cam := range cams_cfg {
                    cam_float, cams_ok := cam.(float64)
                    if cams_ok {
                        cams[i] = int(cam_float)
                    }
                }
                for j,mic := range mics_cfg {
                    mic_float, mics_ok := mic.(float64)
                    if mics_ok {
                        mics[j] = int(mic_float)
                    }
                }

                if !cams_ok || !mics_ok {
                    log.Print("WARNING Error running type assertion (TQ:SetConfig).")
                    cmd.FeedbackChannel <- []byte("")
                }
            recConfig := recordcontrol.RecordConfig{Cameras: cams, Microphones: mics, Sid: data["Sid"].(string), RecFolder: data["RecFolder"].(string)}
            cmd.FeedbackChannel <- rc.TaskSetConfig(recConfig)
            }
        case "StartRecording":
            cmd.FeedbackChannel <- rc.TaskStartRecording()
        case "StopRecording":
            cmd.FeedbackChannel <- rc.TaskStopRecording()
        case "Data":
            data, data_ok := cmd.Data.(map[string]interface{})

            if data_ok {
                trigger, trigger_ok := data["Trigger"].(string)
                recvTime, recvTime_ok := data["recvTime"].(time.Time)
                if !trigger_ok || !recvTime_ok {
                    cmd.FeedbackChannel <- []byte("")
                } else {
                    cmd.FeedbackChannel <- rc.TaskSaveSubtitleEntry(trigger, recvTime)
                }
            } else {
                cmd.FeedbackChannel <- []byte("")
            }
        case "ReturnError":
            cmd.FeedbackChannel <- []byte("")
        default:
            cmd.FeedbackChannel <- []byte("")
        }
    }
}

// A command generated by one of the servers
type Task struct {
    // The type of the command
    Command string
    // The "content" of the command
    // nil for control commands and state requests
    // Everything else is asserted into maps with string keys and interface values
    Data interface{}
    // Feedback channel for the response of the client
    FeedbackChannel chan []byte
}

// Data representing the configuration as set by the user
type DataConfig struct {
    // Cameras to record from
    Camera_ids []int
    // Microphones to record from
    Microphone_ids []int
    // Location to record to
    Recording_location string
    // Id of the subject
    Subject_id string
}
