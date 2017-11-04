package backend

import "net/http"

type Backend interface {
	http.FileSystem
}

type CIDer interface {
	CID() string
}