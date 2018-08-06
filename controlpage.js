// Client side of the graphical web interface for the multicam server

// Server IP
serverip = location.hostname;

// ViewModel specification for knockout bindings
function ControlPageViewModel() {
    var self = this;
    self.ServerStatus = ko.observable();
    self.RecordingConfig = ko.observable();
    self.TriggerValue = ko.observable("");

    self.pushTrigger = function() {
        send_trigger(self.TriggerValue());
    }
}

// The actual status of the server
function ServerStatus() {
    var self = this;
    self.StateId = ko.observable(-1);
    self.CamList = ko.observableArray();
    self.MicList = ko.observableArray();
    self.Disk = ko.observable(new Disk(0,0));
    self.SavingLocationOk = ko.observable(false);
    self.GStreamerOk = ko.observable(false);

    self.SavingLocation = ko.observable("");
    self.Sid = ko.observable("(unknown subject)");

    self.serverstateBg = ko.computed(function() {
        switch (self.StateId()) {
            case -1: return '#333'; 
            case 0: return '#eee'; 
            case 1: return '#8f9'; 
            case 2: return '#fe8'; 
        }
    });
}

function Camera(cam_id, cam_path, cam_image, cfg_record, recording) {
    var self = this;
    self.cam_id = ko.observable(cam_id);
    self.cam_path = ko.observable(cam_path);
    self.cam_image = ko.observable(cam_image);
    self.recording = ko.observable(recording);
    self.cfg_record = ko.observable(cfg_record);
}

function Microphone(mic_id, mic_path, cfg_record, recording) {
    var self = this;
    self.mic_id = ko.observable(mic_id);
    self.mic_path = ko.observable(mic_path);
    self.recording = ko.observable(recording);
    self.cfg_record = ko.observable(cfg_record);
}

function Disk(space_av, space_tot) {
    var self = this;
    self.space_av = ko.observable(space_av);
    self.space_tot = ko.observable(space_tot);
}

// The configuration setting of the client 
function RecordingConfig() {
    var self = this;
    self.RecordCams = ko.observableArray([]);
    self.RecordMics = ko.observableArray([]);
    self.SavingLocation = ko.observable("");
    self.Sid = ko.observable("");
}

// Find a camera id in the ServerStatus (e.g. for checking if a camera of the configuation is actually available)
function findCam(serverState,cam_id) {
    retVal = false;
    $.each(serverState().CamList(),function(i,item) {
        if (item.cam_id() == cam_id) {
            retVal = true;
        }
    });
    return retVal;
}

// Find a microphone id in the ServerStatus (e.g. for checking if a camera of the configuation is actually available)
function findMic(serverState,mic_id) {
    retVal = false;
    $.each(serverState().MicList(),function(i,item) {
        if (item.mic_id() == mic_id) {
            retVal = true;
        }
    });
    return retVal;
}

// Request and control functions via the http side of the server

// Get state of the server
// Handle response depending on the state of the server
function get_status() {

    var config = {
        url: "http://"+serverip+":8040/request",
        data: JSON.stringify({"Command": "REQ", "Data": {"CmdType":"GETSTATUS"}}),
        type: "POST",
        contentType: "application/json", // Request
        dataType: "json" // Response
    };

    var done_fct = function(json) {
        set_client_status(json);
    }

    $.ajax(config).done(done_fct).fail(fail_fct);
}

// Get current configuration of the server
function get_config() {

    var config = {
        url: "http://"+serverip+":8040/request",
        data: JSON.stringify({"Command": "REQ", "Data": {"CmdType":"GETCONFIG"}}),
        type: "POST",
        contentType: "application/json", // Request
        dataType: "json" // Response
    };

    var done_fct = function(json) {
        set_client_config(json);
    }

    $.ajax(config).done(done_fct).fail(fail_fct);
}

// Set current configuration on the server
function set_config() {

    // Extract camera indices
    var record_from_cam = []
    $.each(CPVM.RecordingConfig().RecordCams(), function(i, item) {
        if (item) {
            record_from_cam.push(CPVM.ServerStatus().CamList()[i].cam_id());
        }
    });
    
    // Extract microphone indices
    var record_from_mic = []
    $.each(CPVM.RecordingConfig().RecordMics(), function(i, item) {
        if (item) {
            record_from_mic.push(CPVM.ServerStatus().MicList()[i].mic_id());
        }
    });

    var config = {
        url: "http://"+serverip+":8040/request",
        data: JSON.stringify({"Command": "POST", "Data": {"CmdType":"SETCONFIG", "Values": {"Cameras" : record_from_cam, "Microphones": record_from_mic, "RecFolder": CPVM.RecordingConfig().SavingLocation(), "Sid": CPVM.RecordingConfig().Sid()}}}),
        type: "POST",
        contentType: "application/json", // Request
        dataType: "json" // Response
    };

    // Response is the current config of the server
    var done_fct = function(json) {
        set_client_config(json);
        get_status();
        get_config();
    }

    $.ajax(config).done(done_fct).fail(fail_fct);
}

function start_recording() {

    var config = {
        url: "http://"+serverip+":8040/request",
        data: JSON.stringify({"Command": "CTL", "Data": {"CmdType": "START"}}),
        type: "POST",
        contentType: "application/json", // Request
        dataType: "json" // Response
    };

    // Response is the current config of the server
    var done_fct = function(json) {
        set_client_status(json);
    }

    $.ajax(config).done(done_fct).fail(fail_fct);
}

function stop_recording() {

    var config = {
        url: "http://"+serverip+":8040/request",
        data: JSON.stringify({"Command": "CTL", "Data": {"CmdType": "STOP"}}),
        type: "POST",
        contentType: "application/json", // Request
        dataType: "json" // Response
    };

    // Response is the current config of the server
    var done_fct = function(json) {
        set_client_status(json);
        get_config();
    }

    $.ajax(config).done(done_fct).fail(fail_fct);
}

// Sends trigger data to the server
function send_trigger(trigger_value) {

    var config = {
        url: "http://"+serverip+":8040/request",
        data: JSON.stringify({"Command": "DATA", "Data": {"Values": {"Trigger" : trigger_value}}}),
        type: "POST",
        contentType: "application/json", // Request
        dataType: "json" // Response
    };

    $.ajax(config).fail(fail_fct);
}


function fail_fct(xhr, status, errorThrown) {
    console.log("Error: " + errorThrown);
    console.log("Status: " + status);
    console.log(xhr);
}

// Sets the config of the server in the view
function set_client_config(json) {
    // Reset cameras to "not recording"
    $.each(CPVM.ServerStatus().CamList(), function(i,item) {
        item.cfg_record(false);
    });
    // Set recording state to cameras according to config from server
    $.each(json['Cameras'], function(i,item) {
        camExists = findCam(CPVM.ServerStatus,item);
        if (camExists) {
            CPVM.ServerStatus().CamList()[item].cfg_record(true);
        }
    });
    // Reset microphones to "not recording"
    $.each(CPVM.ServerStatus().MicList(), function(i,item) {
        item.cfg_record(false);
    });
    // Set recording state to microphones according to config from server
    $.each(json['Microphones'], function(i,item) {
        micExists = findMic(CPVM.ServerStatus,item);
        if (micExists) {
            CPVM.ServerStatus().MicList()[item].cfg_record(true);
        }
    });
    
    CPVM.ServerStatus().SavingLocation(json['RecFolder']);
    CPVM.RecordingConfig().SavingLocation(json['RecFolder']);
    CPVM.ServerStatus().Sid(json['Sid']);
    CPVM.RecordingConfig().Sid(json['Sid']);
}

// Sets the status of the server in the view
function set_client_status(json) {
    CPVM.ServerStatus(new ServerStatus());
    CPVM.ServerStatus().StateId(json['Stateid']);
    $.each(json['Cams'], function(i,item) {
        CPVM.ServerStatus().CamList.push(new Camera(this.Id,this.Hardware,json['WebcamCaptureFilename'][i], false, item['Recording']));
    });
    $.each(json['Mics'], function(i,item) {
        CPVM.ServerStatus().MicList.push(new Microphone(this.Id,this.Hardware,false, item['Recording']));
    });
    
    CPVM.ServerStatus().Disk(new Disk(json['Disk']['SpaceAvailable'], json['Disk']['SpaceTotal']));
    CPVM.ServerStatus().SavingLocationOk(json['LocationOk']);
    CPVM.ServerStatus().GStreamerOk(json['GStreamerOk']);
    CPVM.ServerStatus().SavingLocation(CPVM.RecordingConfig().SavingLocation());
    CPVM.ServerStatus().Sid(CPVM.RecordingConfig().Sid());
}


// Instantiating of the viewmodel and application of the bindings

CPVM = new ControlPageViewModel();
CPVM.ServerStatus(new ServerStatus());
CPVM.RecordingConfig(new RecordingConfig());

ko.applyBindings(CPVM);
