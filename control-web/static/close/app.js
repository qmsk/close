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
});
