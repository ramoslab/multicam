// Record control: Controls checking hardware and starting and stopping of the recording scripts
//package recordControl
package recordcontrol

// The record control class
type RecordControl struct {
    // The state of record control: 
    //0 is idle;
    //1 is recording;
    //2 is checking video hardware; 
    //3 is checking audio hardware;
    //4 is checking disk space;
    //5 is checking if the saving location exists;
    //6 is checking if other gstreamer processes are running 
    State int
    // The current configuration structure
    Config RecordConfig
    // The current state of the different aspects RecordControl is controlling
    // 0: Not yet tested; 1: false; 2: true
    VideoHwState int
    AudioHwState int
    DiskSpaceState int
    SavingLocationState int
    GstreamerState int
}

// Update the state value 
func (rc *RecordControl) setState(newstate int) {
    rc.State = newstate
}

// Return the state value
func (rc *RecordControl) GetState() int {
    return rc.State
}

// Checking (preflight)

// Check video hardware
func (rc *RecordControl) CheckVideoHw() {
    rc.setState(2)
    rc.VideoHwState = 2
}

// Check audio hardware
func (rc *RecordControl) CheckAudioHw() {
    rc.setState(3)
    rc.AudioHwState = 2
}

// Check the disk space
func (rc *RecordControl) CheckDiskspace() {
    rc.setState(4)
    rc.DiskSpaceState = 2
}

// Check the saving location
func (rc *RecordControl) CheckSavinglocation() {
    rc.setState(5)
    rc.SavingLocationState = 2
}

// Check if other gstreamer processes are running
func (rc *RecordControl) CheckGstreamer() {
    rc.setState(6)
    rc.GstreamerState = 2
}

// Recording (flight)

// Start recording
func (rc *RecordControl) StartRecording() {
    rc.setState(1)
}

// Stop recording
func (rc *RecordControl) StopRecording() {
    rc.setState(0)
}

// The config class
type RecordConfig struct {
    // Record from these cameras
    Cameras []int
    // ID of the subject
    Sid string
    // Starting date of the server
    Date string
    // Saving location
    Loc string
}
