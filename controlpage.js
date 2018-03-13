// Test AJAX call mit JQuery

function ControlPageViewModel() {
    camlist = ko.observableArray();
    statusitems = ko.observableArray();
}

function Camera(cam_name, cam_image) {
    cam_name = ko.observable(cam_name);
    cam_image = ko.observable(cam_image);
}

function Config(item_name,item_status) {
    item_name = ko.observable(item_name);
    item_status = ko.observable(item_status);
}

function get_answer() {

    var config = {
        url: "http://localhost:8040/request",
        data: {},
        type: "GET",
        dataType: "json"
    };

    var done_fct = function(json) {
        console.log(json);
    }

    $.ajax(config).done(done_fct).fail(fail_fct);
}

function fail_fct(xhr, status, errorThrown) {
    console.log("Error: " + errorThrown);
    console.log("Status: " + status);
    console.log(xhr);
}

//ko.applyBindings(new ControlPageViewModel());
