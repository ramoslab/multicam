// Record control: Controls checking hardware and starting and stopping of the recording scripts
//package recordControl
package recordcontrol

import (
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "syscall"
    "io/ioutil"
    "math"
    "strings"
    "time"
    "sync"
    "log"
)

// The record control class
type RecordControl struct {
    // The current configuration structure
    Config RecordConfig
    // The state of record control: 
    //0 is idle;
    //1 is ready for recording;
    //2 is recording
    //3 is checking video hardware; 
    //4 is checking audio hardware;
    //5 is checking disk space;
    //6 is checking if the saving location exists;
    //7 is checking if other gstreamer processes are running 
    // The actual status of the server (including stateid)
    Status Status
    // Configuration items
    SearchStringAudio string
    //Mutex for locking when multiple goroutines running recording commands access record control
    mux sync.Mutex
}

// Setters

// Updates the state value 
func (rc *RecordControl) setState(newstate int) {
    rc.Status.Stateid = newstate
}

// Sets a new configuration
func (rc *RecordControl) SetConfig(config RecordConfig) {
    rc.Config = config
}

// Getters

// Returns the state value
func (rc *RecordControl) GetStateId() int {
    return rc.Status.Stateid
}

// Returns the status struct
func (rc *RecordControl) GetStatus() Status {
    return rc.Status
}

// Returns the config struct
func (rc *RecordControl) GetConfig() RecordConfig {
    return rc.Config
}



// Checking (preflight)

// Gets all available cameras
// Returns an array of Hardware
func (rc *RecordControl) CheckVideoHw() []Hardware {
    rc.setState(3)
    files, err := ioutil.ReadDir("/dev/")
    if err != nil {
        log.Print("ERROR: Video hardware did not check out; Message: ",err)
    }

    // Retrieve all available cameras
    var cams []string

    for _, f := range files {
        if strings.HasPrefix(f.Name(), "video") {
            cams = append(cams, "/dev/"+f.Name())
        }
    }

    // Log available cameras
    var cam_info string
    cam_info = "INFO: Available webcams:"
    for _, cam := range cams {
        cam_info = cam_info+" "+cam
    }

    log.Print(cam_info)

    hardware := make([]Hardware,len(cams))

    // Add all available cams to the hardware list
    for i, cam := range cams {
        cmd := exec.Command("")
        hardware[i] = Hardware{Id: i, Recording: false, Hardware: cam, Command: cmd}
    }

    return hardware
}

// Checks audio hardware
func (rc *RecordControl) CheckAudioHw() []Hardware {
    rc.setState(4)
    var retVal []Hardware
    // Search for available microphones using search string of config
    searchCmd := exec.Command("/bin/sh","-c",fmt.Sprintf("pactl list | grep -A2 'Source #' | grep 'Name: ' | cut -d\" \" -f2 | grep %s",rc.SearchStringAudio))
    out, err := searchCmd.Output()
    var temp []string
    if err != nil {
        log.Print("ERROR: Could not call command for finding the available microphones; Message: ",err)
        retVal = []Hardware{}
    } else {
        temp = strings.Split(strings.TrimSpace(string(out)),"\n")
        for i,mic := range temp {
            retVal = append(retVal, Hardware{Id: i, Recording: false, Hardware: mic, Command: exec.Command("")})

        }
    }
    return retVal
}

// Returns the disk space of the disk that contains the recording folder
func (rc *RecordControl) CheckDiskspace() Disk {
    rc.setState(5)
    var stat syscall.Statfs_t
    err := syscall.Statfs(rc.Config.RecFolder, &stat)

    if err != nil {
        log.Print("ERROR: Could not stat filesystem; Message:", err)
    }

    return Disk{SpaceAvailable: stat.Bavail * uint64(stat.Bsize) / uint64(math.Pow(1024,3)),
                SpaceTotal: stat.Blocks * uint64(stat.Bsize) / uint64(math.Pow(1024,3))}
}

// Checks the saving location
// If the saving location does not exists, it will be created
// If the saving location can neither be accessed nor created false is returned
// Returns true if the saving location exists
func (rc *RecordControl) CheckSavingLocation() bool {
    rc.setState(6)
    var retVal bool
    // Check if saving location as specified in RecordConfig is available, if not create it. Return false if the location is not available and could not be created.
    _,err := os.Stat(rc.Config.RecFolder)
    if err == nil {
        retVal = true
    }
    if os.IsNotExist(err) {
        err = os.MkdirAll(rc.Config.RecFolder, os.ModePerm)
        if err != nil {
            retVal = false
            log.Print("ERROR: Could not create saving location; Message:", err)
        } else {
            retVal = true
        }
    } else {
        retVal = true
    }
    //rc.CheckGstreamer()
    rc.setState(0)
    return retVal
}

//FIXME Necessary? If so, implement correctly
// Check if other gstreamer processes are running
func (rc *RecordControl) CheckGstreamer() bool {

    rc.setState(7)
    //Check if dead and unkillable GStreamer processes are running. Return "true" if no.
    //TODO Implement properly later
    _, err := exec.Command("sh", "-c", "ps -aux | grep GStreamer").Output()

    //fmt.Println("Result:",string(out))
    if err != nil {
        fmt.Println(err)
    }


    return true
}

// Start recording
func (rc *RecordControl) StartRecording() {
    rc.setState(2)

    // Disable rightlight (auto exposure) before starting to record
    for _,cam := range rc.Status.Cams {
        rightlight_cmd := exec.Command("v4l2-ctl","-c","exposure_auto_priority=0","-d",cam.Hardware)
        err := rightlight_cmd.Run()
        if err != nil {
            log.Printf("WARNING: Could not disable RightLight for %s; Message: %s",cam.Hardware,err)
        } else {
            log.Printf("INFO: Disabling RightLight for %s",cam.Hardware)
        }
    }

    // Generate the gstreamer command for recording the video from the webcams
    gstcommand := "gst-launch-1.0"
    t := time.Now()
    filename_part := fmt.Sprintf("%s",t.Format("060102_150405"))
    argstrs := [][]string{}

    // Iterate over available cameras and assign commands
    for i,cam := range rc.Status.Cams {
        argstrs = append(argstrs,[]string{
            "-e",
            "mp4mux",
            "name=filemux",
            "!",
            "filesink",
            fmt.Sprintf("location=%s%s_%s_v%d.mp4",rc.Config.RecFolder,filename_part,rc.Config.Sid,i),
            "v4l2src",
            fmt.Sprintf("device=%s",cam.Hardware),
            "!",
            "video/x-h264,width=1280,height=720,framerate=30/1",
            "!",
            "h264parse",
            "!",
            "filemux.video_0"})

        rc.Status.Cams[i].Command = exec.Command(gstcommand,argstrs[i]...)
    }

    // Start commands
    for _,camid := range rc.Config.Cameras {
        index := find_camera(rc, camid)
        if index < 0 {
            log.Print("ERROR: Error finding camera")
        }


        cmd := rc.Status.Cams[index].Command
        rc.Status.Cams[index].Recording = true

        err := cmd.Start()
        if err != nil {
            log.Printf("ERROR: Could not start recording on camera %s; Message: %s",camid,err)
        } else {
            log.Printf("INFO: Started recording on camera: CamId: %d, Index: %d, Hardware: %s\n",camid,index,rc.Status.Cams[index].Hardware)
        }

        go waitCamRecording(cmd,camid,rc)
    }

    // Generate the gstreamer command for recording the audio from the webcams
    gstcommand = "gst-launch-1.0"
    t = time.Now()
    filename_part = fmt.Sprintf("%s",t.Format("060102_150405"))
    argstrs = [][]string{}

    // Iterate over available cameras and assign commands
    for i,mic := range rc.Status.Mics {
        argstrs = append(argstrs,[]string{
            "-e",
            "pulsesrc",
            fmt.Sprintf("device=%s",mic.Hardware),
            "!",
            "audioconvert",
            "!",
            "lamemp3enc",
            "target=1",
            "bitrate=192",
            "cbr=true",
            "!",
            "filesink",
            fmt.Sprintf("location=%s%s_%s_v%d.mp3",rc.Config.RecFolder,filename_part,rc.Config.Sid,i),
            })

        rc.Status.Mics[i].Command = exec.Command(gstcommand,argstrs[i]...)
    }

    // Start commands
    for _,micid := range rc.Config.Microphones {
        index := find_microphone(rc, micid)
        if index < 0 {
            log.Print("ERROR: Error finding microphone")
        }

        cmd := rc.Status.Mics[index].Command
        rc.Status.Mics[index].Recording = true

        err := cmd.Start()
        if err != nil {
            log.Printf("ERROR: could not start recording on microphone %s; Message: %s",micid,err)
        } else {
            log.Printf("INFO: Started recording on camera: CamId: %d, Index: %d, Hardware: %s\n",micid,index,rc.Status.Mics[index].Hardware)
        }
        go waitMicRecording(cmd,micid,rc)
    }
}

// Stop recording
func (rc *RecordControl) StopRecording() {
    rc.setState(0)
    // Find all cameras that are still recording
    for i,cam := range rc.Status.Cams {
        if cam.Recording {
            log.Printf("INFO: Stopping process of camera %s\n", cam.Hardware)
            err := rc.Status.Cams[i].Command.Process.Signal(syscall.SIGINT)
            if err != nil {
                log.Printf("ERROR: Error stopping process of camera %s",cam.Hardware)
            }
        }
    }

    for i,mic := range rc.Status.Mics {
        if mic.Recording {
            log.Printf("INFO: Stopping process of camera %s\n", mic.Hardware)
            err := rc.Status.Mics[i].Command.Process.Signal(syscall.SIGINT)
            if err != nil {
                log.Printf("ERROR: Error stopping process of microphone %s",mic.Hardware)
            }
        }
    }
}

// Checks if the given configuration matches the current status
// Changes the configuration to contain only those cameras and microphones that are currently connected and accessible 
// Returns the non-altered or altered configuration
func (rc *RecordControl) CheckConfig(config RecordConfig) RecordConfig {
    var cameras_existing []int
    var microphones_existing []int

    // Check cameras
    for _,n := range config.Cameras {
        // Try to find the camera specified in the config
        if find_camera(rc, n) > -1 {
            // Add the camera to the list of existing cameras
            cameras_existing = append(cameras_existing, n)
        }
    }
    // Set new list of existing cameras as config
    config.Cameras = cameras_existing

    // Check microphones
    for _,n := range config.Microphones {
        // Try to find the microphone specified in the config
        if find_microphone(rc, n) > -1 {
            // Add the camera to the list of existing cameras
            microphones_existing = append(microphones_existing, n)
        }
    }
    // Set new list of existing microphones as config
    config.Microphones = microphones_existing

    return config
}

// Captures a single frame of all available webcams and stores them in a jpg file using fswebcam
// Returns string array containing the filenames created

func (rc RecordControl) CaptureFrame() []string {
    // Delete all webcam captures files
    captures, err := ioutil.ReadDir("static/captures")
    if err != nil {
        //FIXME error handling
        log.Printf("ERROR: Could not read capture directory; Message: %s",err)
    }
    for _, file := range captures {
        if strings.HasPrefix(file.Name(),"captmpv") {
            os.Remove("static/captures/"+file.Name())
        }
    }

    // Capture
    output := make([]string, len(rc.Status.Cams))
    for i,cam := range rc.Status.Cams {

        // Generate file name with time
        t := time.Now()
        fname := fmt.Sprintf("static/captures/captmpv%d_%s.jpg",i,t.Format("060102_150405"))

        // Capture frame 
        argstr := []string{"--jpeg","80","--save",fname,"--device",cam.Hardware}
        cmd := exec.Command("fswebcam",argstr...)
        _, err = cmd.Output()
        if err != nil {
            log.Printf("ERROR: Could not capture from for camera %s; Message: %s",cam.Hardware,err)
            output[i] = ""
        } else {
            log.Printf("INFO: Captured frame: %s",fname)
            output[i] = fname
        }
    }
    return output
}

// Prepares the recording by checking all prerequisites for the recording
func (rc *RecordControl) Preflight() {
    rc.Status.Cams = rc.CheckVideoHw()
    rc.Status.Mics = rc.CheckAudioHw()
    rc.Status.Disk = rc.CheckDiskspace()
    rc.Status.LocationOk = rc.CheckSavingLocation()
    rc.Status.GStreamerOk = rc.CheckGstreamer()
    rc.Config = rc.CheckConfig(rc.Config)
    rc.setState(0)
}

// The tasks

// Generate the STATUS response for the client
// Returns the marshalled JSON byte array of the state struct
func (rc *RecordControl) TaskGetStatus() []byte {
    // Check if server is idle
    if rc.GetStateId() <= 1 {
        // Run Preflight to get the current status
        rc.Preflight()
        // Capture still image from all webcams
        var capture_fnames []string
        capture_fnames = rc.CaptureFrame()
        rc.Status.WebcamCaptureFilename = capture_fnames
    }
    // Marshal the current status into JSON
    retVal, err := json.Marshal(rc.GetStatus())
    // FIXME Proper error handling
    if err != nil {
        log.Printf("ERROR: Error marshalling state to json; Message: %s", err)
        // If marshalling fails, return empty state
        emptyStatus := CreateEmptyStatus()
        retVal, _ = json.Marshal(emptyStatus)
    }
    return retVal
}

// Generates the CONFIG response for the client
// Returns the marshalled JSON byte array of the config struct
func (rc *RecordControl) TaskGetConfig() []byte {
    // Marshal the config into JSON
    retVal, err := json.Marshal(rc.GetConfig())
    // FIXME Proper error handling 
    if err != nil {
        log.Printf("ERROR: Error marshalling config to json. Message: %s", err)
        // If marshalling fails, return empty state
        emptyConfig := CreateEmptyConfig()
        retVal, _ = json.Marshal(emptyConfig)
    }
    return retVal
}

// Sets The configuration given by the client
// In case the server is idle the new config is set and checked and the new config is returned to the client
// The case the server is not idle the previous config is returned to the client
func (rc *RecordControl) TaskSetConfig(config RecordConfig) []byte {
    // Check if server is IDLE
    if rc.GetStateId() <= 1 {
        log.Print("INFO: Setting new config.")
        rc.SetConfig(config)
        log.Print("INFO: Checking new config.")
        rc.CheckConfig(config)
    } else {
        log.Print("WARNING: New config not accepted, because server was not idle.")
    }
    // If not idle send previous config
    return rc.TaskGetConfig()
}

// Starts the recording if the server is idle
// Sends the current status of the server as reply to the client
func (rc *RecordControl) TaskStartRecording() []byte {
    if rc.GetStateId() <= 1 {
        rc.StartRecording()
    }
    return rc.TaskGetStatus()
}

// Stops the recording if the server is recording
// Sends the current status of the server as reply to the client
func (rc *RecordControl) TaskStopRecording() []byte {
    if rc.GetStateId() == 2 {
        rc.StopRecording()
    }
    // Sleep 500ms before returning the status because otherwise the webcams from which video was recorded are not ready to capture the preview frame
    duration := time.Duration(500)*time.Millisecond
    time.Sleep(duration)
    return rc.TaskGetStatus()
}

// Structure definitions

// The configuration for the recording
type RecordConfig struct {
    // Record from these cameras
    Cameras []int
    // Record from these microphones
    Microphones []int
    // ID of the subject
    Sid string
    // Saving location
    RecFolder string
}

// The state of the recording server
type Status struct {
    Cams []Hardware
    Mics []Hardware
    Disk Disk
    LocationOk bool
    GStreamerOk bool
    WebcamCaptureFilename []string
    Stateid int
}

type Hardware struct {
    Id int
    Recording bool
    Hardware string
    Command *exec.Cmd
}

type Disk struct {
    // Disk space in GB
    SpaceAvailable uint64
    SpaceTotal uint64
}

//TODO implement function: Return error

// Helper functions

// Creates an empty status
func CreateEmptyStatus() Status {
    var state Status
    state.Cams = []Hardware{}
    state.Mics = []Hardware{}
    state.Disk = Disk{}
    state.LocationOk = false
    state.GStreamerOk = false
    state.WebcamCaptureFilename = []string{""}
    state.Stateid = -1

    return state
}

// Creates an empty config
func CreateEmptyConfig() RecordConfig {
    var config RecordConfig
    config.Cameras = []int{}
    config.Microphones = []int{}
    config.Sid = ""
    config.RecFolder = ""

    return config
}

// Waits for a process to end
// Sets the Recording to false in the Hardware item the command corresponds to
func waitCamRecording(cmd *exec.Cmd, camid int, rc *RecordControl) {
    // Wait for process to die
    err := cmd.Wait()
    if err != nil {
        log.Printf("ERROR: Error waiting for process of camera %d to finish; Message: %s",camid,err)
    } else {
        log.Printf("INFO: Process of camid %d died.\n",camid)
    }
    // Notify record control that the process has died
    rc.mux.Lock()
    defer rc.mux.Unlock()
    for i,cam := range rc.Status.Cams {
        if cam.Id == camid {
            rc.Status.Cams[i].Recording = false
        }
    }
}

// Waits for a process to end
// Sets the Recording to false in the Hardware item the command corresponds to
func waitMicRecording(cmd *exec.Cmd, micid int, rc *RecordControl) {
    // Wait for process to die
    err := cmd.Wait()
    if err != nil {
        log.Printf("ERROR: Error waiting for process of microphone %d to finish; Message: %s",err)
    } else {
        log.Printf("INFO: Process of micid %d died.\n",micid)
    }
    // Notify record control that the process has died
    rc.mux.Lock()
    defer rc.mux.Unlock()
    for i,mic := range rc.Status.Mics {
        if mic.Id == micid {
            rc.Status.Mics[i].Recording = false
        }
    }
}

// Finds camera with given Id in the status of recording control
// Returns index of this camera
func find_camera(rc *RecordControl, camid int) int {
    retval := -1
    for i,cam := range rc.Status.Cams {
        if camid == cam.Id {
            retval = i
        }
    }
    return retval
}

// Finds microphone with given Id in the status of recording control
// Returns index of this microphone
func find_microphone(rc *RecordControl, micid int) int {
    retval := -1
    for i,mic := range rc.Status.Mics {
        if micid == mic.Id {
            retval = i
        }
    }
    return retval
}
