// Record control: Controls checking hardware and starting and stopping of the recording scripts
//package recordControl
package recordcontrol

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
    StateId int
    // The actual state
    State State
}

// Update the state value 
func (rc *RecordControl) setState(newstate int) {
    rc.StateId = newstate
}

// Return the state value
func (rc *RecordControl) GetState() int {
    return rc.StateId
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

//TODO Does the State of the system (video and audio hardware and saving location) match the configuration
//FIXME Oder soll das lieber oben einzeln geprüft werden?
func (rc *RecordControl) CheckConfig() {
    // If configuration matches, set state to 1 ("ready for recording")
    rc.setState(1)
}

// Prepare the recording by checking all prerequisites for the recording
func (rc *RecordControl) Preflight() {
    rc.State.Cams = rc.CheckVideoHw()
    rc.State.Mics = rc.CheckAudioHw()
    rc.State.Disk = rc.CheckDiskspace()
    rc.State.LocationOk = rc.CheckSavingLocation()
    rc.State.GStreamerOk = rc.CheckGstreamer()
    rc.setState(0)
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
type State struct {
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
