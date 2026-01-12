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
	threads := 0
	if viper.IsSet("dirbrute-threads") {
		threads = viper.GetInt("dirbrute-threads")
	}
	if threads <= 0 && viper.IsSet("spider-threads") {
		threads = viper.GetInt("spider-threads")
	}
	if threads <= 0 {
		threads = 20
	}
	InitHttpClient(threads, viper.GetString("proxy"), dialTimout)
}

func InitHttpClient(threadsNum int, downProxy string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = dialTimout
	}
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: keepAlive,
	}

	tr := &http.Transport{
		DialContext:         dialer.DialContext,
		MaxConnsPerHost:     max(5, threadsNum),
		MaxIdleConns:        threadsNum * 2,
		MaxIdleConnsPerHost: threadsNum * 2,
		IdleConnTimeout:     keepAlive,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: viper.GetBool("insecure")},
		TLSHandshakeTimeout: 5 * time.Second,
		DisableKeepAlives:   false,
	}
	if v := viper.GetInt("conn-per-host"); v > 0 {
		tr.MaxConnsPerHost = v
	}

	if downProxy == "" {
		downProxy = viper.GetString("proxy")
	}
	if downProxy != "" {
		if !strings.HasPrefix(downProxy, "socks") && !strings.HasPrefix(downProxy, "http") {
			return errors.New("do not support the proxy of this type")
		}
		u, err := url.Parse(downProxy)
		if err != nil {
			return err
		}
		tr.Proxy = http.ProxyURL(u)
	}
	httpTimeoutSec := viper.GetInt("http-timeout")
	if httpTimeoutSec <= 0 {
		httpTimeoutSec = 10
	}
	Client = &http.Client{
		Transport: tr,
		Timeout:   time.Duration(httpTimeoutSec) * time.Second,
	}
	ClientNoRedirect = &http.Client{
		Transport:     tr,
		Timeout:       time.Duration(httpTimeoutSec) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}
	return nil
}

// CloneDefaultTransport returns a shallow clone of the default transport used by InitHttp/InitHttpClient.
func CloneDefaultTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   dialTimout,
		KeepAlive: keepAlive,
	}
	tr := &http.Transport{
		DialContext:         dialer.DialContext,
		MaxConnsPerHost:     10,
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     keepAlive,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: viper.GetBool("insecure")},
		TLSHandshakeTimeout: 5 * time.Second,
		DisableKeepAlives:   false,
	}
	if v := viper.GetInt("conn-per-host"); v > 0 {
		tr.MaxConnsPerHost = v
		tr.MaxIdleConnsPerHost = v
	}
	if proxy := viper.GetString("proxy"); proxy != "" {
		if u, err := url.Parse(proxy); err == nil {
			tr.Proxy = http.ProxyURL(u)
		}
	}
	return tr
}
