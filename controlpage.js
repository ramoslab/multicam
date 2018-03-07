// Test AJAX call mit JQuery

function get_answer() {
    console.log("Script gestartet")

    var config = {
        url: "localhost:8040/request",
        data: {},
        type: "GET",
        dataType: "json"
    };

    var done_fct = function(json) {
        console.log(json);
    }

function fail_fct(xhr, status, errorThrown) {
    console.log("Error: " + errorThrown);
    console.log("Status: " + status);
    console.log(xhr);
}
