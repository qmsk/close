package shell

import (
	"net/http"
	"log"
)

func DoRequest(url string, user User, path string) (*http.Response, error) {
	client := &http.Client{
	}

	if req, err := http.NewRequest("GET", url + path, nil); err != nil {
		log.Printf("http.NewRequest %v/%v: %v", url, path, err)
		return nil, err
	} else {
		req.SetBasicAuth(user.Id, user.Password)
		return client.Do(req)
	}
}
