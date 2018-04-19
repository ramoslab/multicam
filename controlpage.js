// Client side of the graphical web interface for the multicam server

// ViewModel specification for knockout bindings
function ControlPageViewModel() {
    var self = this;
    self.ServerState = ko.observable();
    self.RecordingConfig = ko.observable();
}

// The actual state of the server
function ServerState() {
    var self = this;
    self.StateId = ko.observable(-1);
    self.CamList = ko.observableArray();
    self.MicList = ko.observableArray();
    self.Disk = ko.observable(new Disk(0,0));
    self.SavingLocation = ko.observable(false);
    self.GStreamer = ko.observable(false);
    
    //self.serverstateBg = ko.computed(function() {
    //    switch (self.ServerState().StateId()) {
    //        case "-1": return '#333'; 
    //        case "0": return '#ccc'; 
    //        case "1": return '#8f9'; 
    //        case "2": return '#fe8'; 
    //    }
    //});

    self.serverstateBg = ko.computed(function() {
        return '#ccc';
        });
}

function Camera(cam_id, cam_path, cam_image) {
    var self = this;
    self.cam_id = ko.observable(cam_id);
    self.cam_path = ko.observable(cam_path);
    self.cam_image = ko.observable(cam_image);
}

function Microphone(mic_id, mic_path) {
    var self = this;
    self.mic_id = ko.observable(mic_id);
    self.mic_path = ko.observable(mic_path);
}

function Disk(space_av, space_tot) {
    var self = this;
    self.space_av = ko.observable(space_av);
    self.space_tot = ko.observable(space_tot);
}

// The configuration (if untouched it represents the current config of the server; if touched, it can be used to set a new configuration)
function RecordingConfig() {
    var self = this;
    self.RecordCams = ko.observableArray();
    self.RecordMics = ko.observableArray();
    self.SavingLocation = ko.observable();
    self.Sid = ko.observable();
}

// Find a camera id in the ServerState (e.g. for checking if a camera of the configuation is actually available)
function findCam(serverState,cam_id) {
    retVal = false;
    $.each(serverState().CamList(),function(i,item) {
        if (item.cam_id() == cam_id) {
            retVal = true;
        }
    });
    return retVal;
}

// Find a microphone id in the ServerState (e.g. for checking if a camera of the configuation is actually available)
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
function get_state() {

    var config = {
        url: "http://localhost:8040/request",
        data: JSON.stringify({"Command": "REQ", "Data": {"CmdType":"GETSTATE"}}),
        type: "POST",
        contentType: "application/json", // Request
        dataType: "json" // Response
    };

    var done_fct = function(json) {

        CPVM.ServerState(new ServerState());
        CPVM.ServerState().StateId(0);
        $.each(json['Content']['Cams'], function() {
            CPVM.ServerState().CamList.push(new Camera(this.Id,this.Hardware,"image_path"));
        });
        $.each(json['Content']['Mics'], function() {
            CPVM.ServerState().MicList.push(new Microphone(this.Id,this.Hardware));
        });
        
        CPVM.ServerState().Disk(new Disk(json['Content']['Disk']['SpaceAvailable'], json['Content']['Disk']['SpaceTotal']));
        CPVM.ServerState().SavingLocation(json['Content']['LocationOk']);
        CPVM.ServerState().GStreamer(json['Content']['GStreamerOk']);
        console.log(json);
    }

    $.ajax(config).done(done_fct).fail(fail_fct);
}

// Get current configuration of the server
function get_config() {

    var config = {
        url: "http://localhost:8040/request",
        data: JSON.stringify({"Command": "REQ", "Data": {"CmdType":"GETCONFIG"}}),
        type: "POST",
        contentType: "application/json", // Request
        dataType: "json" // Response
    };

    var done_fct = function(json) {

        CPVM.RecordingConfig(new RecordingConfig());
        $.each(json['Content']['Cameras'], function(i,item) {
            camExists = findCam(CPVM.ServerState,item);
            CPVM.RecordingConfig().RecordCams.push(camExists);
        });
        $.each(json['Content']['Microphones'], function(i,item) {
            micExists = findMic(CPVM.ServerState,item);
            console.log(micExists)
            CPVM.RecordingConfig().RecordMics.push(micExists);
        });
        
        CPVM.RecordingConfig().SavingLocation(json['Content']['RecFolder']);
        CPVM.RecordingConfig().Sid(json['Content']['Sid']);
        console.log(json);
    }

    $.ajax(config).done(done_fct).fail(fail_fct);
}

// Set current configuration on the server
function set_config() {

    var config = {
        url: "http://localhost:8040/request",
        data: JSON.stringify({"Command": "REs", "value": "CONFIG"}),
        type: "POST",
        contentType: "application/json", // Request
        dataType: "json" // Response
    };

    var done_fct = function(json) {

        CPVM.RecordingConfig(new RecordingConfig());
        $.each(json['Content']['Cameras'], function(i,item) {
            camExists = findCam(CPVM.ServerState,item);
            CPVM.RecordingConfig().RecordCams.push(camExists);
        });
        $.each(json['Content']['Microphones'], function(i,item) {
            micExists = findMic(CPVM.ServerState,item);
            console.log(micExists)
            CPVM.RecordingConfig().RecordMics.push(micExists);
        });
        
        CPVM.RecordingConfig().SavingLocation(json['Content']['RecFolder']);
        CPVM.RecordingConfig().Sid(json['Content']['Sid']);
        console.log(json);
    }

    $.ajax(config).done(done_fct).fail(fail_fct);
}

function fail_fct(xhr, status, errorThrown) {
    console.log("Error: " + errorThrown);
    console.log("Status: " + status);
    console.log(xhr);
}

// Instantiating of the viewmodel and application of the bindings

CPVM = new ControlPageViewModel();
CPVM.ServerState(new ServerState());
CPVM.RecordingConfig(new RecordingConfig());

ko.applyBindings(CPVM);
