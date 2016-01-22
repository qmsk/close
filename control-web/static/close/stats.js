closeApp.factory('Stats', function($http){
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
                    return r.data.Error;
                }
            );
        },
    };
});

closeApp.controller('StatsCtrl', function($scope, $location, $routeParams, Stats) {
    Stats.index().then(
        function success(statsMeta) {
            // [ {type: field:} ]
            $scope.statsMeta = statsMeta;
        },
        function error(err) {
            $scope.chartAlert = err;
        }
    );

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
        $scope.chartData = [];
        $scope.chartAlert = null;

        if ($scope.type && $scope.field) {
            Stats.get($scope.type, $scope.field, {duration: $scope.statsDuration}).then(
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
    }

    // init
    $scope.statsChart();
});
