// Record control: Controls checking hardware and starting and stopping of the recording scripts
//package recordControl
package recordcontrol

import (
    "encoding/json"
    "fmt"
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
func (rc *RecordControl) SetConfig(Cams []int ) {

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
    return []Hardware{Hardware{Id: 0, Hardware: "/dev/video0"}, Hardware{Id: 1, Hardware: "/dev/video1"}}
}

// Check audio hardware
func (rc *RecordControl) CheckAudioHw() []Hardware {
    rc.setState(4)
    return []Hardware{Hardware{Id: 0, Hardware: "/dev/mic0"},Hardware{Id: 1, Hardware: "/dev/mic1"}}
}

// Check the disk space
func (rc *RecordControl) CheckDiskspace() Disk {
    rc.setState(5)
    return Disk{SpaceAvailable: 100, SpaceTotal: 1000}
}

// Check the saving location
func (rc *RecordControl) CheckSavingLocation() bool {
    rc.setState(6)
    // Check if saving location as specified in RecordConfig is available, if not create it. Return true if the location is not available and could not be created.
    return true
}

// Check if other gstreamer processes are running
func (rc *RecordControl) CheckGstreamer() bool {
    rc.setState(7)
    //Check if dead and unkillable GStreamer processes are running. Return "true" if no.
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
func (rc *RecordControl) TaskSetConfig(config RecordConfig) []byte {
   return []byte{} 
}


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
    SpaceAvailable int
    SpaceTotal int
}

//TODO implement function: Return error
