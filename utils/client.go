package utils

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var (
	Client           *http.Client
	ClientNoRedirect *http.Client
	dialTimout       = 10 * time.Second
	keepAlive        = 10 * time.Second
)

func InitHttp() {
	//PocInfo.Proxy = "http://127.0.0.1:8080"
	// err := InitHttpClient(
	// 	PocInfo.Num, PocInfo.Proxy, time.Duration(PocInfo.Timeout)*time.Second)
	InitHttpClient(1, "", dialTimout)
}

func InitHttpClient(ThreadsNum int, DownProxy string, Timeout time.Duration) error {
	// type DialContext = func(ctx context.Context, network, addr string) (net.Conn, error)
	dialer := &net.Dialer{
		Timeout:   dialTimout,
		KeepAlive: keepAlive,
	}

	tr := &http.Transport{
		DialContext:         dialer.DialContext,
		MaxConnsPerHost:     5,
		MaxIdleConns:        ThreadsNum,
		MaxIdleConnsPerHost: ThreadsNum * 2,
		IdleConnTimeout:     keepAlive,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		TLSHandshakeTimeout: 5 * time.Second,
		DisableKeepAlives:   false,
	}

	if viper.GetString("proxy") != "" {
		DownProxy := viper.GetString("proxy")
		if !strings.HasPrefix(DownProxy, "socks") && !strings.HasPrefix(DownProxy, "http") {
			return errors.New("do not support the proxy of this type")
		}
		u, err := url.Parse(DownProxy)
		if err != nil {
			return err
		}
		tr.Proxy = http.ProxyURL(u)
	}
	Client = &http.Client{
		Transport: tr,
		Timeout:   Timeout,
	}
	ClientNoRedirect = &http.Client{
		Transport:     tr,
		Timeout:       Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}
	return nil
}
