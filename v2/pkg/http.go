package pkg

import (
	"crypto/tls"
	"github.com/chainreactors/logs"
	"net"
	"net/http"
	"net/url"
	"time"
)

var ProxyUrl *url.URL
var Proxy func(*http.Request) (*url.URL, error)
var maxRedirects = 5

func HttpConn(delay int) *http.Client {
	tr := &http.Transport{
		Proxy: Proxy,
		//TLSHandshakeTimeout : delay * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS10,
			Renegotiation:      tls.RenegotiateOnceAsClient,
			InsecureSkipVerify: true,
		},
		DialContext: (&net.Dialer{
			//Timeout:   time.Duration(delay) * time.Second,
			//KeepAlive: time.Duration(delay) * time.Second,
			//DualStack: true,
		}).DialContext,
		MaxIdleConnsPerHost: 1,
		MaxIdleConns:        4000,
		IdleConnTimeout:     time.Duration(delay) * time.Second,
		DisableKeepAlives:   false,
	}

	conn := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(delay) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			//if !followRedirects {
			//	return http.ErrUseLastResponse
			//}
			//if req.URL.Host == "localhost" || req.URL.Host == "127.0.0.1" {
			//	return http.ErrUseLastResponse
			//}
			logs.Log.Debugf("redirect to %s from %s", req.URL.String(), via[len(via)-1].URL.String())

			if len(via) > maxRedirects {
				return http.ErrUseLastResponse
			}

			return nil
		},
	}

	return conn
}

func HttpConnWithNoRedirect(delay int) *http.Client {
	tr := &http.Transport{
		Proxy: Proxy,
		//TLSHandshakeTimeout : delay * time.Second,
		TLSClientConfig: &tls.Config{
			Renegotiation:      tls.RenegotiateOnceAsClient,
			InsecureSkipVerify: true,
		},
		DialContext: (&net.Dialer{
			//Timeout:   time.Duration(delay) * time.Second,
			//KeepAlive: time.Duration(delay) * time.Second,
			//DualStack: true,
		}).DialContext,
		MaxIdleConnsPerHost: 1,
		MaxIdleConns:        4000,
		IdleConnTimeout:     time.Duration(delay) * time.Second,
		DisableKeepAlives:   false,
	}

	conn := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(delay) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return conn
}
