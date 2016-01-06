var closeApp = angular.module('closeApp', [
        'ngRoute',
        'angular-flot'
]);

closeApp.config(function($routeProvider){
    $routeProvider
        .when('/workers', {
            templateUrl: '/close/workers.html',
            controller: 'WorkersCtrl',
        })
        .when('/workers/:type/:id', {
            templateUrl: '/close/worker.html',
            controller: 'WorkerCtrl',
        })
        .when('/stats', {
            templateUrl: '/close/stats.html',
            controller: 'StatsCtrl',
        })
        .otherwise({
            redirectTo: '/workers',
        });
});

closeApp.controller('StatsCtrl', function($scope, $http) {
    $http.get('/api/stats/').success(function(data){
        $scope.statsSeries = data;
    });

    $scope.chartOptions = {
        xaxis: { mode: "time" },
    };
    $scope.select = function(statsInfo) {
        $scope.statsInfo = statsInfo;

        $http.get('/api/stats/' + statsInfo.type + '/', {params:{hostname: statsInfo.hostname, instance: statsInfo.instance}}).success(function(data){
            $scope.statsData = data.map(function(series){
                var dataPoints = series.points.map(function(point){
                    var date = new Date(point.time);

                    return [date.getTime(), point.value];
                });

                return [{
                    label: series.field,
                    data: dataPoints
                }];
            });
        });
    }
});

closeApp.controller('WorkersCtrl', function($scope, $http) {
    $scope.configWorkers = {};
    
    $http.get('/api/config/').success(function(data){
        $scope.configWorkers = data;
    });
});

function mapStatsData(series) {
    return series.points.map(function(point){
        var date = new Date(point.time);

        return [date.getTime(), point.value];
    });
}

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
