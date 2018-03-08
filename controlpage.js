// Test AJAX call mit JQuery

function ControlPageViewModel() {
    test = ko.observable("a");
}

function get_answer() {
    console.log("Script gestartet")

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
