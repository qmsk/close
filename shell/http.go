package shell

import (
	"fmt"
	"net/http"
	"io"
	"log"
)

type JSONResponseParser interface {
	ParseJSON(io.ReadCloser) error
}

func MakeHttpRequest(command GenericCommand) error {
	url := command.Url()
	path := command.Path()
	user := command.User()
	client := &http.Client{
	}

	if req, err := http.NewRequest("GET", url + path, nil); err != nil {
		return fmt.Errorf("http.NewRequest %v %v: %v", url, path, err)
	} else {
		req.SetBasicAuth(user.Id, user.Password)

		if resp, err := client.Do(req); err != nil {
			return fmt.Errorf("http.client.Do %v: %v", url, err)
		} else {
			defer resp.Body.Close()
			log.Printf("Response %v, %v, content length %v\n", resp.Status, resp.Proto, resp.ContentLength)
			return command.ParseJSON(resp.Body)
		}
	}
}
