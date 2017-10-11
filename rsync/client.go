package rsync

import (
	"bytes"
	"net"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Manages a single client.
type rsyncClient struct {
	conn net.Conn
}

func CreateRsyncClient(conn net.Conn) *rsyncClient {
	daemon := rsyncClient{conn}
	return &daemon
}

func (rsc *rsyncClient) Work() {
	defer rsc.cleanup()

	logger := log.WithFields(log.Fields{
		"remote_addr": rsc.conn.RemoteAddr(),
	})
	logger.Debug("rsync daemon: starting")

	tcpconn, ok := rsc.conn.(*net.TCPConn)
	if !ok {
		logger.Error("connection is not a TCP/IP connection")
		return
	}
	connfile, err := tcpconn.File()
	if err != nil {
		logger.WithError(err).Error("unable to get file descriptor from TCP/IP connection")
		return
	}

	// Start the RSync process, connecting it to the network connection.
	cmd := exec.Command("./rsync-bin", "/path-one", "&3::flamenco")
	// logger.WithField("cmd", cmd).Debug("running")
	cmd.Stdout = os.Stdout
	cmd.ExtraFiles = []*os.File{connfile}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	cmd.Stderr = os.Stdout

	if err := cmd.Run(); err != nil {
		stderr := string(stderrBuf.Bytes())
		trimmed := strings.TrimSpace(stderr)
		logger.WithError(err).WithField("stderr", trimmed).Error("Error running rsync")
		return
	}
	logger.Info("rsync ran OK, closing connection")
	// rsd.conn.Write([]byte("je moeder"))
}

func (rsd *rsyncClient) cleanup() {
	logger := log.WithFields(log.Fields{
		"remote_addr": rsd.conn.RemoteAddr(),
	})

	if err := rsd.conn.Close(); err != nil {
		logger.WithError(err).Warning("rsync client cleanup: unable to close connection")
	} else {
		logger.Debug("Connection closed")
	}

	// TODO: remove this daemon from the server list of daemons
}
