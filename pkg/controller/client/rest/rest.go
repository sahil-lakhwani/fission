package rest

import (
	"net/http"
)

type (
	Interface interface {
		Create(relativeUrl string, contentType string, payload []byte) (*http.Response, error)
		Put(relativeUrl string, contentType string, payload []byte) (*http.Response, error)
		Get(relativeUrl string) (*http.Response, error)
		Delete(relativeUrl string) error
		Proxy(method string, relativeUrl string, payload []byte) (*http.Response, error)
	}
)
