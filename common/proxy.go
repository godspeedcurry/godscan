package common

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

func WrapperTcpWithTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	d := &net.Dialer{Timeout: timeout / 2}
	return WrapperTCP(network, address, d)
}

func WrapperTCP(network, address string, forward *net.Dialer) (net.Conn, error) {
	//get conn
	var conn net.Conn
	if Proxy == "" {
		var err error
		conn, err = forward.Dial(network, address)
		if err != nil {
			return nil, err
		}
	} else {
		dailer, err := Socks5Dailer(forward)
		if err != nil {
			return nil, err
		}
		conn, err = dailer.Dial(network, address)
		if err != nil {
			return nil, err
		}
	}
	return conn, nil

}

func Socks5Dailer(forward *net.Dialer) (proxy.Dialer, error) {
	u, err := url.Parse(Proxy)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(u.Scheme) != "socks5" {
		return nil, errors.New("only support socks5")
	}
	address := u.Host
	var auth proxy.Auth
	var dailer proxy.Dialer
	if u.User.String() != "" {
		auth = proxy.Auth{}
		auth.User = u.User.Username()
		password, _ := u.User.Password()
		auth.Password = password
		dailer, err = proxy.SOCKS5("tcp", address, &auth, forward)
	} else {
		dailer, err = proxy.SOCKS5("tcp", address, nil, forward)
	}

	if err != nil {
		return nil, err
	}
	return dailer, nil
}
