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
    // The channels for checking if the recording processes are still running
    //Mutex for locking when multiple goroutines running recording commands access record control
    mux sync.Mutex
}

// Setters

// Updates the state value 
func (rc *RecordControl) setState(newstate int) {
    rc.Status.Stateid = newstate
    fmt.Println("State: ",newstate)
}

// Sets a new configuration
func (rc *RecordControl) SetConfig(config RecordConfig) {
    rc.Config = config
    fmt.Println("Setting config as:",config)
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
    //FIXME error handling
    if err != nil {
        fmt.Println("Error",err)
    }

    // Retrieve all available cameras
    var cams []string

    fmt.Println("Available Webcams:")
    for _, f := range files {
        if strings.HasPrefix(f.Name(), "video") {
            fmt.Println("/dev/"+f.Name())
            cams = append(cams, "/dev/"+f.Name())
        }
    }
    //TODO Retrieve all available microphones
    var mics []string

    hardware := make([]Hardware,len(cams)+len(mics))

    // Add all available cams to the hardware list
    for i, cam := range cams {
        cmd := exec.Command("")
        hardware[i] = Hardware{Id: i, Recording: false, Hardware: cam, Command: cmd}
    }

    //TODO Add all available mics to the hardware list

    return hardware
}

//TODO Not yet implemented
// Checks audio hardware
func (rc *RecordControl) CheckAudioHw() []Hardware {
    rc.setState(4)
    cmd := exec.Command("")
    return []Hardware{Hardware{Id: 0, Recording: false, Hardware: "/dev/mic0", Command: cmd},Hardware{Id: 1, Recording: false, Hardware: "/dev/mic1", Command: cmd}}
}

// Returns the disk space of the disk that contains the recording folder
func (rc *RecordControl) CheckDiskspace() Disk {
    rc.setState(5)
    var stat syscall.Statfs_t
    //FIXME error handling
    syscall.Statfs(rc.Config.RecFolder, &stat)

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
        fmt.Printf("Starting recording on camera: CamId: %d, Index: %d, Hardware: %s\n",camid,index,rc.Status.Cams[index].Hardware)

        cmd := rc.Status.Cams[index].Command
        rc.Status.Cams[index].Recording = true

        err := cmd.Start()
        fmt.Println(cmd.Args,err)
        go waitwait(cmd,camid,rc)
    }
}

// Stop recording
func (rc *RecordControl) StopRecording() {
    rc.setState(0)
    // Find all cameras that are still recording
    for i,cam := range rc.Status.Cams {
        fmt.Println(cam)
        if cam.Recording {
            fmt.Printf("Stopping process of camera %s\n", cam.Hardware)
            fmt.Println(cam.Command.Args)
            err := rc.Status.Cams[i].Command.Process.Signal(syscall.SIGINT)
            fmt.Println(err)
        }
    }
}

// Checks if the given configuration matches the current status
func (rc *RecordControl) CheckConfig(config RecordConfig) bool {
    var retVal bool
    // Check cameras
    for _,n := range config.Cameras {
        if n < len(rc.Status.Cams) {
            retVal = true
        }
    }

    // Check microphones
    for _,n := range config.Microphones {
        if n < len(rc.Status.Mics) {
            retVal = retVal && true
        } else {
            retVal = retVal && false
        }
    }

    // Check saving location
    retVal = retVal && rc.CheckSavingLocation()

    return retVal
}

// Captures a single frame of all available webcams and stores them in a jpg file using fswebcam
// Returns string array containing the filenames created

func (rc RecordControl) CaptureFrame() []string {
    // Delete all webcam captures files
    captures, err := ioutil.ReadDir("static/captures")
    if err != nil {
        //FIXME error handling
        fmt.Println("Error with file")
    }
    for _, file := range captures {
        if strings.HasPrefix(file.Name(),"captmpv") {
            os.Remove("static/captures/"+file.Name())
        }
    }

    // Capture
    output := make([]string, len(rc.Status.Cams))
    for i,cam := range rc.Status.Cams {
        fmt.Println(i)


        // Generate file name with time
        t := time.Now()
        fname := fmt.Sprintf("static/captures/captmpv%d_%s.jpg",i,t.Format("060102_150405"))

        // Capture frame 
        argstr := []string{"--jpeg","80","--save",fname,"--device",cam.Hardware}
        cmd := exec.Command("fswebcam",argstr...)
        _, err = cmd.Output()
        if err != nil {
            fmt.Println(err)
            output[i] = ""
        } else {
            fmt.Println("Captured frame:",fname)
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
        fmt.Println("Error marshalling state", err)
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
    if err != nil { fmt.Println("Error marshalling config", err)
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
        fmt.Println("Setting new config.")
        rc.SetConfig(config)
        fmt.Println("Checking new config.")
        rc.CheckConfig(config)
    }
    //TODO Log new config not accepted because server is not idle
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
func waitwait(cmd *exec.Cmd, camid int, rc *RecordControl) {
    fmt.Printf("Waiting for camid %d\n",camid)
    // Wait for process to die
    err := cmd.Wait()
    if err != nil {
        fmt.Println(err)
    }
    fmt.Printf("Process of camid %d died.\n",camid)
    // Notify record control that the process has died
    rc.mux.Lock()
    defer rc.mux.Unlock()
    for i,cam := range rc.Status.Cams {
        if cam.Id == camid {
            rc.Status.Cams[i].Recording = false
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
