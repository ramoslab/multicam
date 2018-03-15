// Client side of the graphical web interface for the multicam server

// ViewModel specification for knockout bindings
function ControlPageViewModel() {
    var self = this;
    self.serverstate = ko.observable("-1");
    self.serverstateBg = ko.computed(function() {
        switch (self.serverstate()) {
            case "-1": return '#333'; 
            case "0": return '#ccc'; 
            case "1": return '#8f9'; 
            case "2": return '#fe8'; 
        }
    });
    self.camlist = ko.observableArray();
    self.statusitems = ko.observableArray();
}

function Camera(cam_name, cam_image) {
    var self = this;
    self.cam_name = ko.observable(cam_name);
    self.cam_image = ko.observable(cam_image);
}

function StatusItem(item_name,item_status) {
    var self = this;
    self.item_name = ko.observable(item_name);
    self.item_status = ko.observable(item_status);
}

// Request and control functions via the http side of the server

// Get state of the server
function get_answer() {

    var config = {
        url: "http://localhost:8040/request",
        data: JSON.stringify({"request": "state", "value": 100}),
        type: "POST",
        contentType: "application/json", // Request
        dataType: "json" // Response
    };

    var done_fct = function(json) {
        console.log(json);
        CPVM.serverstate(json['state']);
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

CPVM.camlist.push(new Camera("Cam 1","Cam 1 Image"));
CPVM.camlist.push(new Camera("Cam 2","Cam 2 Image"));
CPVM.camlist.push(new Camera("Cam 3","Cam 3 Image"));
CPVM.statusitems.push(new StatusItem("Disk space","ok"));
CPVM.statusitems.push(new StatusItem("Saving location","ok"));
CPVM.statusitems.push(new StatusItem("GStreamer","ok"));

ko.applyBindings(CPVM);
