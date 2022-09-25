const {GreeterRequest, Resp} = require('./api/gen/v1/greeter_pb');
const {GreeterClient} = require('./api/gen/v1/greeter_grpc_web_pb');

window.onload = function() {
    var service = new GreeterClient('http://localhost:8080', null, null);
    var req = new GreeterRequest();
    req.setMessage("grpc web request")

    service.greeter(req, {}, function(err, response) {
        if (err !== null) {
            document.getElementById("res").innerHTML = err
        } else {
            document.getElementById("res").innerHTML = response.getMessage()
        }
    })
};