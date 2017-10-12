package rsync

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Client manages a single client connection to an rsync server.
type Client struct {
	conn net.Conn
}

// CreateRsyncClient sets up an rsync client for a specific network connection.
func CreateRsyncClient(conn net.Conn) *Client {
	daemon := Client{conn}
	return &daemon
}

// Work starts the rsync binary and lets it communicate with the server over the network.
func (rsc *Client) Work() {
	defer rsc.cleanup()
	var err error

	logger := log.WithFields(log.Fields{
		"remote_addr": rsc.conn.RemoteAddr(),
	})
	logger.Debug("rsync daemon: starting")

	port, err := rsc.startTCP()
	if err != nil {
		logger.WithError(err).Fatal("unable to start local TCP tunnel server")
	}
	logger = logger.WithField("tunnel_port", port)

	// Start the RSync process, connecting it to the network connection.
	cmd := exec.Command("rsync", "./LICENSE.txt", fmt.Sprintf("--port=%d", port), "localhost::", "--verbose")
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		logger.WithError(err).Error("Error running rsync")
		return
	}
	logger.Info("rsync ran OK, closing connection")
}

// Starts a local TCP/IP server that proxies between rsc.conn and whatever connects to it.
func (rsc *Client) startTCP() (int, error) {
	listener, err := net.Listen("tcp", "localhost:0") // port 0 means "choose automatically"
	if err != nil {
		return 0, fmt.Errorf("unable to open local port: %s", err)
	}

	_, portStr, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		return 0, fmt.Errorf("error getting port number from tunnel address: %s", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("error converting port nr to integer: %s", err)
	}

	go func() {
		rsyncConn, accepterr := listener.Accept()
		if accepterr != nil {
			log.WithError(accepterr).Fatal("unable to accept local connection")
		}
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			io.Copy(rsc.conn, rsyncConn)
			log.Debug("tunnel: rsc.conn ← rsyncConn done")
			wg.Done()
		}()
		go func() {
			io.Copy(rsyncConn, rsc.conn)
			log.Debug("tunnel: rsyncConn ← rsc.conn done")

			// Close the TCP/IP connection to our local rsync when the HTTP connection dies.
			rsyncConn.Close()
			wg.Done()
		}()
		wg.Wait()
		log.Debug("tunnel: finished")
	}()

	return port, nil
}

func (rsc *Client) cleanup() {
	logger := log.WithFields(log.Fields{
		"remote_addr": rsc.conn.RemoteAddr(),
	})

	if err := rsc.conn.Close(); err != nil {
		logger.WithError(err).Warning("rsync client cleanup: unable to close connection")
	} else {
		logger.Debug("Connection closed")
	}
}
