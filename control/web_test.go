package control

import (
    "github.com/ant0ine/go-json-rest/rest"
    "github.com/ant0ine/go-json-rest/rest/test"
    "testing"
)

func TestDeleteWorkers(t *testing.T) {
    api := rest.NewApi()
    api.Use(rest.DefaultDevStack...)

    var manager JsonApp = &Manager {
    }

    if app, err := manager.RestApp(); err != nil {
        t.Fatalf("manager.RestApp: %v\n", err)
    } else {
        api.SetApp(app)
    }

    recorded := test.RunRequest(t, api.MakeHandler(),
        test.MakeSimpleRequest("DELETE", "http://0.0.0.0/workers", nil))
    recorded.CodeIs(200)
    recorded.ContentTypeIsJson()

    recorded2 := test.RunRequest(t, api.MakeHandler(),
        test.MakeSimpleRequest("DELETE", "http://0.0.0.0/", nil))
    recorded2.CodeIs(405)
    recorded2.ContentTypeIsJson()
}
    
