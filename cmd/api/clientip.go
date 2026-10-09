package main

import (
	"errors"
	"net"
	"net/http"
	"strings"
)

func parseTrustedProxies(value string) ([]*net.IPNet, error) {
	var networks []*net.IPNet

	for _, entry := range strings.Fields(value) {
		if strings.Contains(entry, "/") {
			_, network, err := net.ParseCIDR(entry)
			if err != nil {
				return nil, err
			}
			networks = append(networks, network)
			continue
		}

		ip := net.ParseIP(entry)
		if ip == nil {
			return nil, errors.New("invalid trusted proxy address: " + entry)
		}

		bits := 32
		if ip.To4() == nil {
			bits = 128
		}
		networks = append(networks, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
	}

	return networks, nil
}

func (app *application) isTrustedProxy(ip net.IP) bool {
	for _, network := range app.config.trustedProxies {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

func (app *application) clientIP(r *http.Request) string {
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		peer = r.RemoteAddr
	}

	peerIP := net.ParseIP(peer)
	if len(app.config.trustedProxies) == 0 || peerIP == nil || !app.isTrustedProxy(peerIP) {
		return peer
	}

	var hops []string
	for _, header := range r.Header.Values("X-Forwarded-For") {
		for _, hop := range strings.Split(header, ",") {
			hops = append(hops, strings.TrimSpace(hop))
		}
	}

	for i := len(hops) - 1; i >= 0; i-- {
		ip := net.ParseIP(hops[i])
		if ip == nil {
			break
		}
		if !app.isTrustedProxy(ip) {
			return hops[i]
		}
	}

	return peer
}
