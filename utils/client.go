package utils

import (
	"crypto/tls"
	"errors"
	"main/common"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

var (
	Client           *http.Client
	ClientNoRedirect *http.Client
	dialTimout       = 5 * time.Second
	keepAlive        = 5 * time.Second
)

func InitHttp() {
	//PocInfo.Proxy = "http://127.0.0.1:8080"
	// err := InitHttpClient(
	// 	PocInfo.Num, PocInfo.Proxy, time.Duration(PocInfo.Timeout)*time.Second)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	InitHttpClient(1, "", time.Second*5)
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
		MaxIdleConns:        0,
		MaxIdleConnsPerHost: ThreadsNum * 2,
		IdleConnTimeout:     keepAlive,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		TLSHandshakeTimeout: 5 * time.Second,
		DisableKeepAlives:   false,
	}

	if common.Proxy != "" {
		dialSocksProxy, err := common.Socks5Dailer(dialer)
		if err != nil {
			return err
		}
		if contextDialer, ok := dialSocksProxy.(proxy.ContextDialer); ok {
			tr.DialContext = contextDialer.DialContext
		} else {
			return errors.New("failed type assertion to DialContext")
		}
	} else if DownProxy != "" {
		if DownProxy == "1" {
			DownProxy = "http://127.0.0.1:8080"
		} else if DownProxy == "2" {
			DownProxy = "socks5://127.0.0.1:1080"
		} else if !strings.Contains(DownProxy, "://") {
			DownProxy = "http://127.0.0.1:" + DownProxy
		}
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
