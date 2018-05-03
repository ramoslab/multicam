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
    // The actual state
    Status Status
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
        hardware[i] = Hardware{Id: i, Hardware: cam}
    }

    //TODO Add all available mics to the hardware list

    return hardware
}

// Check audio hardware
func (rc *RecordControl) CheckAudioHw() []Hardware {
    rc.setState(4)
    return []Hardware{Hardware{Id: 0, Hardware: "/dev/mic0"},Hardware{Id: 1, Hardware: "/dev/mic1"}}
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
}

// Stop recording
func (rc *RecordControl) StopRecording() {
    rc.setState(0)
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
}

type Hardware struct {
    Id int
    Hardware string
}

type Disk struct {
    // Disk space in GB
    SpaceAvailable uint64
    SpaceTotal uint64
}

//TODO implement function: Return error
