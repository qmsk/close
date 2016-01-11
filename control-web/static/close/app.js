var closeApp = angular.module('closeApp', [
        'ngRoute',
        'angular-flot'
]);

closeApp.config(function($routeProvider){
    $routeProvider
        .when('/workers', {
            templateUrl: '/close/workers.html',
            controller: 'WorkersCtrl'
        })
        .when('/workers/:type/:id', {
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
        $http.get('/api/').success(function(data){
            $scope.config = data.config_text;
            $scope.workers = data.workers;

            $scope.busy = false;

            if (data.config.Worker) {
                $scope.statsChart(data.config.Worker.RateStats);
            }
        });
    };

    $scope.postConfig = function(){
        $scope.busy = true;
        $http.post('/api/', $scope.config).then(
            function success(r){
                $scope.configAlert = "Config OK";
                $scope.get();
            },
            function error(r){
                $scope.configAlert = r.data;
            }
        );
    }

    $scope.stop = function(){
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
    $scope.closeType = $routeParams.type;
    $scope.closeInstance = $routeParams.id;

    var statsType = $routeParams.type;
    var statsInstance = $routeParams.id;

    switch ($scope.closeType) {
    case "udp":
        statsType = "udp_send";
    }
    
    $http.get('/api/config/' + $routeParams.type + '/' + $routeParams.id).success(function(data){
        $scope.workerConfig = data;
    });

    $scope.chartOptions = {
        xaxis: { mode: "time" },
    };
    $http.get('/api/stats/' + statsType + '/', {params:{instance: statsInstance}}).success(function(data){
        if (data) {
            $scope.statsData = data.map(function(series){
                return [{
                    label: series.field,
                    data: mapStatsData(series)
                }];
            });
        }
    });

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
        $http.post('/api/config/' + $routeParams.type + '/' + $routeParams.id, $scope.postConfig, {
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


