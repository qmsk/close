angular.module('close.stats', [

])

/*
 * /api/stats
 */
.factory('Stats', function($http){
    return {
        /*
         * Fetch index of available stats types-fields.
         * 
         * 
         * Returns: [ {type field} ]
         */
        index: function() {
            return $http.get('/api/stats').then(
                function success(r) {
                    // flattened array
                    return $.map(r.data, function(meta){
                        return $.map(meta.fields, function(field){
                            return {type: meta.type, field: field};
                        });
                    });
                },
                function error(r) {
                    return r.data.Error;
                }
            );
        },

        /*
         * Fetch stats data in the form of a <flot dataset>.
         *
         * Error: string message
         * Returns: [{label data}].
         */
        get: function(type, field, params){
            if (!type) {
                throw("Missing type=");
            }

            var url = '/api/stats/' + type + '/';

            if (field) {
                url += field;
            }

            // promise chaining
            return $http.get(url, {params:params}).then(
                function success(r) {
                    if (!r.data) {
                        return [];
                    }

                    // flattened array
                    return $.map(r.data, function(series){
                        // TODO: format label contextually, based on parameters given
                        return {
                            series: series,
                            label:  series.type + "." + series.field + "@" + series.hostname + ":" + series.instance,
                            data:   series.points.map(function(point){
                                var date = new Date(point.time);

                                return [date.getTime(), point.value];
                            }),
                        };
                    });
                },
                function error(r) {
                    if (r.data && r.data.Error) {
                        return r.data.Error;
                    } else {
                        return r;
                    }
                }
            );
        },
    };
})

.directive('statsChart', function(){
    return {
        scope: {
            statsMeta:  '=statsMeta',
            height:     '@height',
        },
        templateUrl:    '/close/stats/chart.html',
        controller:     function($scope, $location, Stats) {
            //$scope.height = "400px";

            // XXX: use a global value
            $scope.duration = $location.search()['duration'];
            if (!$scope.duration) {
                $scope.duration = "1m";
            }

            $scope.changeDuration = function() {
                $location.search('duration', $scope.duration);

                $scope.update();
            }

            /*
             * Select given {type: field:} for viewing
             */
            $scope.update = function() {
                var meta = $scope.statsMeta;

                // view state
                $scope.chartOptions = {};
                $scope.chartData = [];
                $scope.chartAlert = null;

                if (!meta) {
                    return
                };

                Stats.get(meta.type, meta.field, {instance: meta.instance, duration: $scope.duration}).then(
                    function success(stats){
                        if (!stats || stats.length == 0) {
                            $scope.chartAlert = "No Data";
                            return;
                        } else {
                            $scope.chartAlert = false;
                        }
                    
                        $scope.chartMap = {};
                        $scope.chartCount = 0;
                        $.each(stats, function(i, stat){
                            var chartData = $scope.chartMap[stat.series.field];

                            if (chartData == undefined) {
                                $scope.chartCount++;
                                chartData = $scope.chartMap[stat.series.field] = [];
                            }
                            
                            chartData.field = stat.series.field;
                            chartData.push(stat);
                        });
                        $scope.chartOptions = {
                            legend: {
                                show: $scope.chartCount == 1,
                            },
                            xaxis: { mode: "time" },
                        };
                    },
                    function error(err){
                        $scope.chartAlert = err;
                    }
                );
            };

            $scope.$watch('statsMeta', $scope.update);
        },
    };
})

;
