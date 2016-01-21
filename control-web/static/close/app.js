var closeApp = angular.module('closeApp', [
        'ngRoute',
        'angular-flot',
        'angular-websocket',
        'luegg.directives'      // angularjs-scroll-glue
]);

closeApp.config(function($routeProvider){
    $routeProvider
        .when('/workers', {
            templateUrl: '/close/workers.html',
            controller: 'WorkersCtrl'
        })
        .when('/workers/:config/:instance', {
            templateUrl: '/close/worker.html',
            controller: 'WorkerCtrl'
        })
        .when('/docker', {
            templateUrl: '/close/docker-index.html',
            controller: 'DockerIndexCtrl',
        })
        .when('/docker/:id', {
            templateUrl: '/close/docker.html',
            controller: 'DockerCtrl',
        })
        .when('/stats', {
            templateUrl: '/close/stats.html',
            controller: 'StatsCtrl',
            reloadOnSearch: false
        })
        .otherwise({
            redirectTo: '/workers'
        });
});

function mapStatsData(series) {
    return series.points.map(function(point){
        var date = new Date(point.time);

        return [date.getTime(), point.value];
    });
}

closeApp.controller('HeaderController', function($scope, $location) {
    $scope.navActive = function(prefix) {
        return $location.path().startsWith(prefix);
    };
});

closeApp.controller('WorkersCtrl', function($scope, $routeParams, $location, $http) {
    $scope.busy = true;
    $scope.get = function(){
        $http.get('/api/').then(
            function success(r){
                $scope.config = r.data.config_text;
                $scope.clients = r.data.clients;
                $scope.workers = r.data.workers;

                $scope.busy = false;

                if (r.data.config && r.data.config.Workers) {
                    // XXX: multiple configs?!
                    $.each(r.data.config.Workers, function(configName, workerConfig){
                        if (workerConfig.RateStats) {
                            $scope.statsChart(workerConfig.RateStats);
                        }
                    });
                }
            },
            function error(r){
                $scope.configAlert = r.data;
            }
        );
    };

    $scope.postConfig = function(){
        $scope.busy = true;
        $http.post('/api/', $scope.config).then(
            function success(r){
                $scope.configAlert = "Config OK";
                $scope.get();
            },
            function error(r){
                $scope.busy = false;
                $scope.configAlert = r.data;
            }
        );
    }

    $scope.stopWorkers = function(){
        $scope.busy = true;
        $http.delete('/api/workers').then(
            function success(r){
                $scope.configAlert = "Workers stopped";
                $scope.get();
            },
            function error(r){
                $scope.configAlert = r.data;
            }
        );
    }

    // XXX: copy-pasta
    $scope.statsDuration = $routeParams.duration || "10s";
    $scope.statsChart = function(stats) {
        if (stats) {
            $scope.stats = stats;
        } else {
            stats = $scope.stats
        }
        // update view state
        $location.search('duration', $scope.statsDuration);

        // update
        var statsParams = {duration: $scope.statsDuration};

        $scope.chartData = [];
        $scope.chartAlert = null;

        $http.get('/api/stats/' + stats, {params: statsParams}).then(
            function success(r){
                if (!r.data || r.data.length == 0) {
                    $scope.chartAlert = "No Data";
                    return;
                }

                $scope.chartOptions = {
                    xaxis: { mode: "time" },
                };
                $scope.chartData = r.data.map(function(series){
                    var label = series.type + "." + series.field + "@" + series.hostname + ":" + series.instance;

                    console.log("stats: " + label);

                    return {
                        label: label,
                        data: mapStatsData(series)
                    };
                });
            },
            function error(response){
                $scope.chartAlert = r.data;
            }
        );
    }

    $scope.get();
});

closeApp.controller('WorkerCtrl', function($scope, $http, $routeParams) {
    $scope.config = $routeParams.config;
    $scope.instance = $routeParams.instance;

    $http.get('/api/workers/' + $routeParams.config + '/' + $routeParams.instance).then(
            function success(r) {
                $scope.error = null;
                $scope.worker = r.data;
                $scope.workerConfig = r.data.worker_config;
                $scope.configMap = r.data.config_map;

                $scope.getStats();
            },
            function error(r) {
                $scope.error = r.data;
            }
    );

    $scope.getConfig = function() {
        $http.get('/api/config/' + $scope.workerConfig.Type + '/' + $scope.worker.config_instance).then(
            function success (r) {
                $scope.error = null;
                $scope.configMap = r.data;
            },
            function error(r) {
                $scope.error = r.data;
            }
        );
    }

    $scope.getStats = function() {
        var statsType = $scope.workerConfig.StatsType;
        var statsInstance = $scope.worker.stats_instance;

        if (!(statsType && statsInstance)) {
            return;
        }

        $scope.chartOptions = {
            xaxis: { mode: "time" },
        };
        $http.get('/api/stats/' + statsType + '/', {params:{instance: statsInstance}}).then(
            function success(r) {
                $scope.error = null;
                if (r.data) {
                    $scope.statsData = r.data.map(function(series){
                        return [{
                            label: series.field,
                            data: mapStatsData(series)
                        }];
                    });
                }
            },
            function error(r) {
                $scope.error = r.data;
            }
        );
    }

    // shadow copy of workerConfig, used as the <input ng-model> to POST any changed fields
    $scope.postConfig = {};

    // The config is POST'd as JSON, so the type of the value must match - we cannot POST a number value as a string
    // Angular can preserve the <input ng-model> value's type, as long as we use the right <input type>
    $scope.inputType = function(value) {
        switch (typeof value) {
            case "string":  return "text";
            case "number":  return "number";
            case "boolean": return "checkbox";
            default:        return false;
        }
    }

    // POST any changed config <form> fields to the server for the worker to apply
    $scope.submitConfig = function() {
        // only changed fields
        $http.post('/api/config/' + $scope.workerConfig.Type + '/' + $scope.worker.config_instance, $scope.postConfig, {
            headers: { 'Content-Type': 'application/json' },
        });
    };
});

closeApp.controller('DockerIndexCtrl', function($scope, $http) {
    $http.get('/api/docker/').success(function(data){
        $scope.dockerContainers = data;
    });
});

closeApp.controller('DockerCtrl', function($scope, $routeParams, $http) {
    $scope.dockerID = $routeParams.id;

    $http.get('/api/docker/' + $scope.dockerID).success(function(data){
        $scope.dockerContainer = data;
    });
    $http.get('/api/docker/' + $scope.dockerID + '/logs').success(function(data){
        $scope.dockerLogs = data;
    });
});

closeApp.controller('StatsCtrl', function($scope, $location, $routeParams, $http) {
    $http.get('/api/stats').success(function(data){
        // [ {type: field:} ]
        $scope.statsMeta = $.map(data, function(meta){
            return meta.fields.map(function(field){
                return {type: meta.type, field: field};
            });
        });
    });

    $scope.statsActive = function(fieldMeta) {
        return fieldMeta.type == $scope.type && fieldMeta.field == $scope.field;
    }

    /*
     * Select given {type: field:} for viewing
     */
    $scope.statsChart = function(fieldMeta) {
        if (fieldMeta) {
            $scope.type = fieldMeta.type;
            $scope.field = fieldMeta.field;
        } else if ($scope.type && $scope.field){

        } else {
            $scope.type = $routeParams.type;
            $scope.field = $routeParams.field;
            $scope.statsDuration = $routeParams.duration || "10s";
        }

        // update view state
        $location.search('type', $scope.type);
        $location.search('field', $scope.field);
        $location.search('duration', $scope.statsDuration);

        // update
        var statsURL = '/api/stats/' + $scope.type + '/' + $scope.field;
        var statsParams = {duration: $scope.statsDuration};

        console.log("get stats: " + statsURL + "?" + statsParams);

        $scope.chartData = [];
        $scope.chartAlert = null;

        $http.get(statsURL, {params: statsParams}).then(
            function success(r){
                if (!r.data || r.data.length == 0) {
                    $scope.chartAlert = "No Data";
                    return;
                }

                $scope.chartOptions = {
                    xaxis: { mode: "time" },
                };
                $scope.chartData = r.data.map(function(series){
                    var label = series.type + "." + series.field + "@" + series.hostname + ":" + series.instance;

                    console.log("stats: " + label);

                    return {
                        label: label,
                        data: mapStatsData(series)
                    };
                });
                    
                console.log("stats length=" + $scope.chartData.length);
            },
            function error(response){
                $scope.chartAlert = r.data;
            }
        );
    }

    // init
    $scope.statsChart();
});

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
