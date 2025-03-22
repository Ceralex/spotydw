package service

import (
	"net/url"
)

type Service interface {
	Download(url *url.URL, concurrentN int) error
}
