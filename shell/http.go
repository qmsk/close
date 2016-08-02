package shell

import (
	"net/http"
	"log"
)

func DoRequest(options Options, path string) (*http.Response, error) {
	client := &http.Client{
	}

	if req, err := http.NewRequest("GET", options.URL + path, nil); err != nil {
		log.Printf("http.NewRequest %v/%v: %v", options.URL, path, err)
		return nil, err
	} else {
		req.SetBasicAuth(options.User.Id, options.User.Password)
		return client.Do(req)
	}
}
