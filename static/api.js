// TODO: functions for getting and setting stuff

var initPage = function() {
  $("#statusDiv")[0].textContent = "...";

  $.getJSON( "/api/status", function(data) {
    console.log(data)
    $("#statusDiv")[0].textContent = data.response;
  }, function(success) {
    console.log(success)
  })
}