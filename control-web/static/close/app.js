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
                    // XXX: multiple configs?!
                    $.each(r.data.config.Workers, function(configName, workerConfig){
                        var match = workerConfig.RateStats.match(/((\w+)\/)?(\w+)/);

                        if (!match) {

                        } else if (match[1]) {
                            $scope.statsChart(match[2], match[3]);

                        } else if (match[1]) {
                            $scope.statsChart(workerConfig.StatsType, match[3]);
                        } else {

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
    $scope.statsChart = function(statsType, statsField) {
        if (statsType && statsField) {
            $scope.statsType = statsType;
            $scope.statsField = statsField;
        } else {
            statsType = $scope.statsType;
            statsField = $scope.statsField;
        }
        // update view state
        $location.search('duration', $scope.statsDuration);

        // update
        var statsParams = {duration: $scope.statsDuration};

        $scope.chartData = [];
        $scope.chartAlert = null;

        Stats.get(statsType, statsField, {duration: $scope.statsDuration}).then(
            function success(stats){
                if (!stats || stats.length == 0) {
                    $scope.chartAlert = "No Data";
                    return;
                }

                $scope.chartOptions = {
                    xaxis: { mode: "time" },
                };
                $scope.chartData = stats;
            },
            function error(err){
                $scope.chartAlert = err;
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

        // all fields
        Stats.get(statsType, null, {instance: statsInstance}).then(
            function success(stats) {
                $scope.error = null;

                $scope.chartOptions = {
                    xaxis: { mode: "time" },
                };
                $scope.statsData = stats;
            },
            function error(err) {
                $scope.error = err;
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
