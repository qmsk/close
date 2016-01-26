var closeApp = angular.module('closeApp', [
        'close.stats',
        'ngRoute',
        'angular-flot',
        'angular-websocket',
        'luegg.directives'      // angularjs-scroll-glue
]);

closeApp.config(function($routeProvider){
    $routeProvider
        .when('/workers', {
            templateUrl: '/close/workers.html',
            controller: 'WorkersCtrl',
            reloadOnSearch: false
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

closeApp.controller('HeaderController', function($scope, $location) {
    $scope.navActive = function(prefix) {
        return $location.path().startsWith(prefix);
    };
});

/*
 * Parse a "[<type>/]<field>" expression into a statsMeta object ({type: field:}), or null.
 */
function parseWorkerStats(statsType, statsExpr){
    if (!statsExpr) {
        return null;
    }

    var match = statsExpr.match(/((\w+)\/)?(\w+)/);

    if (!match) {
        return null;
    } else if (match[1]) {
        return {type: match[2], field: match[3]};
    } else if (match[3]) {
        return {type: statsType, field: match[3]};
    } else {
        return null;
    }
}

closeApp.controller('WorkersCtrl', function($scope, $routeParams, $location, $http, Stats) {
    $scope.busy = true;
    $scope.get = function(){
        $http.get('/api/').then(
            function success(r){
                $scope.config = r.data.config_text;
                $scope.clients = r.data.clients;
                $scope.workers = r.data.workers;

                $scope.busy = false;

                if (r.data.config && r.data.config.Workers) {
                    // XXX: need to merge workers with identical statsMetas into one chart, since without ?instance= each such chart will render all workers...
                    $scope.workerStats = $.map(r.data.config.Workers, function(workerConfig, configName){
                        var workerStats = [];

                        if ((rateStats = parseWorkerStats(workerConfig.StatsType, workerConfig.RateStats))) {
                            workerStats.push({
                                workerConfig:   configName,
                                title:          "Rate",
                                statsMeta:      rateStats,
                                statsUnit:      "/s",
                                ylabel:         "Rate",
                            });
                        }
                        if ((latencyStats = parseWorkerStats(workerConfig.StatsType, workerConfig.LatencyStats))) {
                            workerStats.push({
                                workerConfig:   configName,
                                title:          "Latency",
                                statsMeta:      latencyStats,
                                statsUnit:      "s",
                                ylabel:         "Latency",
                            });
                        }

                        return workerStats
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

    $scope.get();
});

closeApp.controller('WorkerCtrl', function($scope, $http, $routeParams, Stats) {
    $scope.config = $routeParams.config;
    $scope.instance = $routeParams.instance;

    $http.get('/api/workers/' + $routeParams.config + '/' + $routeParams.instance).then(
            function success(r) {
                $scope.error = null;
                $scope.worker = r.data;
                $scope.workerConfig = r.data.worker_config;
                $scope.configMap = r.data.config_map;
                $scope.statsMeta = {
                    type:       $scope.workerConfig.StatsType,
                    instance:   $scope.worker.stats_instance,
                };
            },
            function error(r) {
                $scope.error = r.data;
            }
    );

    /* ConfigController */
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

closeApp.controller('StatsCtrl', function($scope, $location, $routeParams, Stats) {
    Stats.index().then(
        function success(statsIndex) {
            // [ {type: field:} ]
            $scope.statsIndex = statsIndex;
        },
        function error(err) {
            $scope.chartAlert = err;
        }
    );

    $scope.statsMeta = null;
    $scope.statsActive = function(meta) {
        return $scope.statsMeta && meta.type == $scope.statsMeta.type && meta.field == $scope.statsMeta.field;
    }

    /*
     * Select given {type: field:} for viewing
     */
    $scope.select = function(meta) {
        if (meta) {
            $scope.statsMeta = meta;
        } else if ($scope.statsMeta){
            meta = $scope.statsMeta;
        } else if ($routeParams.type && $routeParams.field) {
            meta = $scope.statsMeta = {type: $routeParams.type, field: $routeParams.field};
        } else {
            return;
        }

        // update view state
        $location.search('type', meta.type);
        $location.search('field', meta.field);
    }

    // init
    $scope.select();
});
