package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"

	"github.com/9seconds/mtg/v2/mtglib/network"
)

func makeNetwork(conf *config) (network.Network, error) {
	tcpTimeout := conf.Network.Timeout.TCP.Value(network.DefaultTimeout)
	idleTimeout := conf.Network.Timeout.Idle.Value(network.DefaultIdleTimeout)
	dohIP := conf.Network.DOHIP.Value(net.ParseIP(network.DefaultDOHHostname)).String()
	bufferSize := conf.TCPBuffer.Value(network.DefaultBufferSize)

	baseDialer, err := network.NewDefaultDialer(tcpTimeout, int(bufferSize))
	if err != nil {
		return nil, fmt.Errorf("cannot build a default dialer: %w", err)
	}

	proxyURLs := make([]*url.URL, 0, len(conf.Network.Proxies))

	for _, v := range conf.Network.Proxies {
		if value := v.Value(nil); value != nil {
			proxyURLs = append(proxyURLs, v.Value(nil))
		}
	}

	switch len(proxyURLs) {
	case 0:
		return network.NewNetwork(baseDialer, dohIP, idleTimeout)
	case 1:
		socksDialer, err := network.NewSocks5Dialer(baseDialer, proxyURLs[0])
		if err != nil {
			return nil, fmt.Errorf("cannot build socks5 dialer: %w", err)
		}

		return network.NewNetwork(socksDialer, dohIP, idleTimeout)
	}

	socksDialer, err := network.NewLoadBalancedSocks5Dialer(baseDialer, proxyURLs)
	if err != nil {
		return nil, fmt.Errorf("cannot build socks5 dialer: %w", err)
	}

	return network.NewNetwork(socksDialer, dohIP, idleTimeout)
}

func exhaustResponse(response *http.Response) {
	io.Copy(ioutil.Discard, response.Body) // nolint: errcheck
	response.Body.Close()
}
