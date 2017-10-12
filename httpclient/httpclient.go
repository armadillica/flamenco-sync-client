package httpclient

import (
	"bufio"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var handshakeTimeout = 30 * time.Second

// Dial performs a HTTP call and "upgrades" it to a regular socket
func Dial(urlStr string, insecureSkipVerify bool) (net.Conn, error) {
	logger := log.WithField("url", urlStr)

	req, err := http.NewRequest("RSYNC", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Upgrade", "rsync")
	req.Header.Set("Connection", "Upgrade")
	req.SetBasicAuth("je", "moeder") // TODO(sybren): implement

	u := req.URL

	hostPort, hostNoPort := hostPortNoPort(u)

	var deadline time.Time
	deadline = time.Now().Add(handshakeTimeout)

	netDialer := &net.Dialer{Deadline: deadline}
	netConn, err := netDialer.Dial("tcp", hostPort)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "https" {
		cfg := &tls.Config{
			ServerName:         hostNoPort,
			InsecureSkipVerify: insecureSkipVerify,
		}
		tlsConn := tls.Client(netConn, cfg)
		netConn = tlsConn // Replace netConn with the TLS connection so it gets tunneled.
		if tlserr := tlsConn.Handshake(); tlserr != nil {
			return nil, tlserr
		}
		if !cfg.InsecureSkipVerify {
			if verifyerr := tlsConn.VerifyHostname(cfg.ServerName); verifyerr != nil {
				return nil, verifyerr
			}
		}
	}
	if writeErr := req.Write(netConn); writeErr != nil {
		logger.WithError(writeErr).Error("error writing request")
		return nil, writeErr
	}

	readBufferSize := 128
	br := bufio.NewReaderSize(netConn, readBufferSize)
	resp, err := http.ReadResponse(br, req)
	if err != nil {
		logger.WithError(err).Error("error reading response")
		return nil, err
	}
	logger.WithField("headers", resp.Header).Debug("response headers")

	return netConn, nil
}

func hostPortNoPort(u *url.URL) (hostPort, hostNoPort string) {
	hostPort = u.Host
	hostNoPort = u.Host
	if i := strings.LastIndex(u.Host, ":"); i > strings.LastIndex(u.Host, "]") {
		hostNoPort = hostNoPort[:i]
	} else {
		switch u.Scheme {
		case "https":
			hostPort += ":443"
		default:
			hostPort += ":80"
		}
	}
	return hostPort, hostNoPort
}
