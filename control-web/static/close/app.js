var closeApp = angular.module('closeApp', [
        'ngRoute'
]);

closeApp.config(function($routeProvider){
    $routeProvider
        .when('/workers', {
            templateUrl: '/close/workers.html',
            controller: 'WorkersCtrl',
        })
        .when('/workers/:workerId', {
            templateUrl: '/close/worker.html',
            controller: 'WorkerCtrl',
        })
        .otherwise({
            redirectTo: '/workers',
        });
});

closeApp.controller('WorkersCtrl', function($scope, $http) {
    $scope.workers = {};
    
    $http.get('/api/workers/').success(function(data){
        $scope.workers = data;
    });
});

closeApp.controller('WorkerCtrl', function($scope, $http, $routeParams) {
    $scope.workerId = $routeParams.workerId
    
    $http.get('/api/workers/' + $routeParams.workerId).success(function(data){
        $scope.workerConfig = data;
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
        $http.post('/api/workers/' + $routeParams.workerId, $scope.postConfig, {
            headers: { 'Content-Type': 'application/json' },
        });
    };
});
