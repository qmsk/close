describe('WorkerCtrl', function() {

  beforeEach(module('closeApp'));

  var $httpBackend, $rootScope, createController;
  var url = '/api/workers/undefined/undefined';

  beforeEach(inject(function($injector) {
    // Set up the mock http service responses
    $httpBackend = $injector.get('$httpBackend');

    // Get hold of a scope (i.e. the root scope)
    $rootScope = $injector.get('$rootScope');
    // The $controller service is used to create instances of controllers
    var $controller = $injector.get('$controller');

    createController = function() {
      return $controller('WorkerCtrl', {'$scope' : $rootScope });
    };
  }));

//  beforeEach(inject(function(_$controller_) {
//    $controller = _$controller_;
//  }));

  afterEach(function() {
    $httpBackend.verifyNoOutstandingExpectation();
    $httpBackend.verifyNoOutstandingRequest();
  });

  it('should provide a delete function', function() {
    $httpBackend.expectGET(url).respond('200', '');

    var ctrl = createController();
    $httpBackend.flush();

    expect($rootScope.delete).toBeDefined();
  });

  describe('$scope.delete', function() {
    it('should issue a delete request to "/api/workers/undefined/undefined"', function() {
      $httpBackend.expectGET(url).respond('200', '');
      var ctrl = createController();

      $rootScope.delete();
      $httpBackend.expectDELETE(url).respond('200', '');
      $httpBackend.flush();
    });
  });
});
