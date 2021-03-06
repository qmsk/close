angular.module('close.stats', [

])

/*
 * /api/stats
 */
.factory('Stats', function($http, $q){
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
                    if (!r.data) {
                        return [];
                    }

                    // flattened array
                    return $.map(r.data, function(meta){
                        return $.map(meta.fields, function(field){
                            return {type: meta.type, field: field};
                        });
                    });
                },
                function error(r) {
                    $q.reject(r.data.Error);
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
                        $q.reject(r.data.Error);
                    } else {
                        $q.reject(r);
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
            unit:       '=?',
            ylabel:     '@?',
        },
        templateUrl:    '/close/stats/chart.html',
        controller:     function($scope, $location, Stats) {
            //$scope.height = "400px";

            /* Duration */
            // XXX: use a global value, shared across all charts in the view..
            $scope.duration = $location.search()['duration'];
            if (!$scope.duration) {
                $scope.duration = "1m";
            }

            $scope.changeDuration = function() {
                $location.search('duration', $scope.duration);

                $scope.update();
            }

            /* Tick formatter */
            if ($scope.unit) { switch ($scope.unit) {
            case "s":
                $scope.tickDecimals = 5; // .xx ms
                $scope.tickFormatter = function durationFormatter(val, axis) {
                    if (axis.max > 1.0) {
                        return val.toFixed(axis.tickDecimals - 3) + " s";
                    } else {
                        return (val * 1000.0).toFixed(axis.tickDecimals - 3) + " ms";
                    }
                };
                break;
            default:
                $scope.tickDecimals = 2;
                $scope.tickFormatter = function unitFormatter(val, axis) {
                    return val.toFixed(axis.tickDecimals) + " " + $scope.unit;
                };
                break;
            } }

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
                    
                        /* Group by field; render a separate chart for each field */
                        $scope.chartMap = {};
                        $.each(stats, function(i, stat){
                            var chartData = $scope.chartMap[stat.series.field];

                            if (chartData == undefined) {
                                chartData = $scope.chartMap[stat.series.field] = [];
                            }
                            
                            chartData.field = stat.series.field;
                            chartData.push(stat);
                        });

                        $scope.chartOptions = {
                            axisLabels: {

                            },
                            legend: {
                                show: !$scope.statsMeta.instance,
                            },
                            xaxis: {
                                mode: "time",
                            },
                            yaxis: {
                                axisLabel:      $scope.ylabel,
                                tickFormatter:  $scope.tickFormatter,
                                tickDecimals:   $scope.tickDecimals,
                            },
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
