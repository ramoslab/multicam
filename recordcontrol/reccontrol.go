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
    State int
    // The actual status of the server
    Status Status
    // The channels for checking if the recording processes are still running
    //Mutex for locking when multiple goroutines running recording commands access record control
    mux sync.Mutex
}

// Update the state value 
func (rc *RecordControl) setState(newstate int) {
    rc.State = newstate
}

// Return the state value
func (rc *RecordControl) GetStateId() int {
    return rc.State
}

// Return the status struct
func (rc *RecordControl) GetStatus() Status {
    return rc.Status
}

// Return the config struct
func (rc *RecordControl) GetConfig() RecordConfig {
    return rc.Config
}

// Set a new configuration
func (rc *RecordControl) SetConfig(config RecordConfig) {
    rc.Config = config
    fmt.Println("Setting config as:",config)
}

// Create an empty state
func CreateEmptyStatus() Status {
    var state Status
    state.Cams = []Hardware{}
    state.Mics = []Hardware{}
    state.Disk = Disk{}
    state.LocationOk = false
    state.GStreamerOk = false
    state.WebcamCaptureFilename = []string{""}

    return state
}

// Create an empty state
func CreateEmptyConfig() RecordConfig {
    var config RecordConfig
    config.Cameras = []int{}
    config.Microphones = []int{}
    config.Sid = ""
    config.RecFolder = ""

    return config
}

// Checking (preflight)

// Get cameras available
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

// Check audio hardware
func (rc *RecordControl) CheckAudioHw() []Hardware {
    rc.setState(4)
    cmd := exec.Command("")
    return []Hardware{Hardware{Id: 0, Recording: false, Hardware: "/dev/mic0", Command: cmd},Hardware{Id: 1, Recording: false, Hardware: "/dev/mic1", Command: cmd}}
}

// Check the disk space of the disk that contains the recording folder
func (rc *RecordControl) CheckDiskspace() Disk {
    rc.setState(5)
    var stat syscall.Statfs_t
    //FIXME error handling
    syscall.Statfs(rc.Config.RecFolder, &stat)

    return Disk{SpaceAvailable: stat.Bavail * uint64(stat.Bsize) / uint64(math.Pow(1024,3)),
                SpaceTotal: stat.Blocks * uint64(stat.Bsize) / uint64(math.Pow(1024,3))}
}

// Check the saving location
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

// Recording (flight)

// Start recording
func (rc *RecordControl) StartRecording() {
    rc.setState(2)

    // Generate the gstreamer command for recording the video from the webcams
    gstcommand := "gst-launch-1.0"
    argstrs := [][]string{}

    for i,cam := range rc.Status.Cams {
        argstrs = append(argstrs,[]string{
            "-e",
            "mp4mux",
            "name=filemux",
            "!",
            "filesink",
            fmt.Sprintf("location=%s%d.mp4",rc.Config.RecFolder,i),
            "v4l2src",
            fmt.Sprintf("device=%s",cam.Hardware),
            "!",
            "video/x-h264,width=1280,height=720,framerate=30/1",
            "!",
            "h264parse",
            "!",
            "filemux.video_0"})
    }

    cmd := make([]*exec.Cmd,len(rc.Config.Cameras))
    for i,_ := range cmd {
        cmd[i] = exec.Command(gstcommand,argstrs[rc.Config.Cameras[i]]...)
        //FIXME Den command hier in der Kamera-Struct speichern. Dann können die Prozesse auch wieder korrekt interrupted werden.
    }

    for i,_ := range cmd {
        cmd[i].Start()
        go waitwait(cmd[i],rc.Status.Cams[i].Id,rc)
    }

    //for i := range cmd {
    //    fmt.Printf("Process %d is done\n",i)
    //}

}

// Stop recording
func (rc *RecordControl) StopRecording() {
    rc.setState(0)
    // Find all cameras that are still recording
    for _,cam := range rc.Status.Cams {
        fmt.Println(cam)
        if cam.Recording {
            fmt.Println("Stopping process.")
            cam.Command.Process.Signal(syscall.SIGINT)
        }
    }

}

//TODO Does the Status of the system (video and audio hardware and saving location) match the configuration
//FIXME Oder soll das lieber oben einzeln geprüft werden?
//TODO Funktion, die eine gegebene Config mit dem Status testet
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

// Capture single frame of webcam and store in jpg file using fswebcam
// Return string array containing the filenames created

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

// Prepare the recording by checking all prerequisites for the recording
func (rc *RecordControl) Preflight() {
    rc.Status.Cams = rc.CheckVideoHw()
    rc.Status.Mics = rc.CheckAudioHw()
    rc.Status.Disk = rc.CheckDiskspace()
    rc.Status.LocationOk = rc.CheckSavingLocation()
    rc.Status.GStreamerOk = rc.CheckGstreamer()
    rc.setState(0)
}

// Function generating the STATE response for the client
// Returns the marshalled JSON byte array of the state struct
func (rc *RecordControl) TaskGetStatus() []byte {
    // Run Preflight to get the current state
    rc.Preflight()
    // Capture still image from all webcams
    var capture_fnames []string
    capture_fnames = rc.CaptureFrame()
    rc.Status.WebcamCaptureFilename = capture_fnames
    // Check if recording is still running
    // Marshal the state into JSON
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

// Function generating the CONFIG response for the client
// Returns the marshalled JSON byte array of the config struct
func (rc *RecordControl) TaskGetConfig() []byte {
    // Marshal the config into JSON
    retVal, err := json.Marshal(rc.GetConfig())
    // FIXME Proper error handling
    if err != nil {
        fmt.Println("Error marshalling config", err)
        // If marshalling fails, return empty state
        emptyConfig := CreateEmptyConfig()
        retVal, _ = json.Marshal(emptyConfig)
    }
    return retVal
}

// Set the config given by the client
// Generate the SETCONFIG response for the client
//func (rc *RecordControl) TaskSetConfig(config RecordConfig) []byte {
func (rc *RecordControl) TaskSetConfig(config RecordConfig) []byte {
    fmt.Println("Setting new config.")
    rc.SetConfig(config)
    fmt.Println("Checking new config.")
    rc.CheckConfig(config)
    return rc.TaskGetConfig()
}

// Start the recording
func (rc *RecordControl) TaskStartRecording() []byte {
    rc.StartRecording()
    return []byte(`{"Test": "test"}`)
}

//Stop recording
func (rc *RecordControl) TaskStopRecording() []byte {
    rc.StopRecording()
    return []byte(`{"Test": "test"}`)
}


// Function running the preflight to check the hardware status of the system
// Return the marshalled JSON byte array (including a message to the client)

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
    for _,cam := range rc.Status.Cams {
        if cam.Id == camid {
            cam.Recording = false
        }
    }
}

func interrupt_process(cmd *exec.Cmd, quit chan bool) {
    for {
        select {
        case quit <- true:
            cmd.Process.Signal(syscall.SIGINT)
        }
    }
}
