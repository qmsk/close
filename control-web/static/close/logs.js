closeApp.factory('Logs', function LogsFactory($websocket) {
    var Logs = {
        url:        'ws://' + window.location.host + '/logs',
        Error:      null,
        Messages:   [],   
    };
    var ws;

    try {
        ws = $websocket(Logs.url);
    } catch (err) {
        Logs.Error = "WS failed: " + err;
        return Logs;
    }

    Logs.message = function(msg){
        msg.id = "log-" + Logs.Messages.length;

        Logs.Messages.push(msg)  
    };
    Logs.log = function(line){
        Logs.message({line: line});
    };

    ws.onOpen(function(e){
        Logs.log("WebSocket Open");
        Logs.Error = null;
    });
    ws.onClose(function(e){
        Logs.log("WebSocket Close: " + e.reason);
        Logs.Error = e; // XXX: .reason;
    });
    ws.onError(function(e){
        Logs.log("WebSocket Error");
    });
    ws.onMessage(function(msg){
        Logs.message(JSON.parse(msg.data));
    });

    Logs.status = function() {
        switch (ws.readyState) {
            case WebSocket.CONNECTING:
                return "Connecting..";
            case WebSocket.OPEN:
                return "Open";
            case WebSocket.CLOSING:
                return "Closing...";
            case WebSocket.CLOSED:
                return "Closed";
            default:
                return "unknown " + ws.readyState + "?";
        }
    }

    return Logs;
});

closeApp.controller('LogsController', function($scope, Logs) {
    $scope.Logs = Logs;
});
