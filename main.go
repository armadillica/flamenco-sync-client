package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	stdlog "log"

	"github.com/armadillica/flamenco-sync-client/httpclient"
	"github.com/armadillica/flamenco-sync-client/rsync"
	log "github.com/sirupsen/logrus"
)

const applicationVersion = "1.0-dev"
const applicationName = "Flamenco Sync Client"

var cliArgs struct {
	version  bool
	verbose  bool
	debug    bool
	insecure bool
	url      string
}

func parseCliArgs() {
	flag.BoolVar(&cliArgs.version, "version", false, "Shows the application version, then exits.")
	flag.BoolVar(&cliArgs.verbose, "verbose", false, "Enable info-level logging.")
	flag.BoolVar(&cliArgs.debug, "debug", false, "Enable debug-level logging.")
	flag.BoolVar(&cliArgs.insecure, "insecure", false, "Skips verification of the HTTPS TLS certificate. Only use for testing.")
	flag.StringVar(&cliArgs.url, "url", "http://localhost:8084/", "URL to connect to.")
	flag.Parse()
}

func configLogging() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// Only log the warning severity or above by default.
	level := log.WarnLevel
	if cliArgs.debug {
		level = log.DebugLevel
	} else if cliArgs.verbose {
		level = log.InfoLevel
	}
	log.SetLevel(level)
	stdlog.SetOutput(log.StandardLogger().Writer())
}

func logStartup() {
	level := log.GetLevel()
	defer log.SetLevel(level)

	log.SetLevel(log.InfoLevel)
	log.WithFields(log.Fields{
		"version": applicationVersion,
	}).Infof("Starting %s", applicationName)
}

func main() {
	parseCliArgs()
	if cliArgs.version {
		fmt.Println(applicationVersion)
		return
	}

	configLogging()
	logStartup()

	// Set some more or less sensible limits & timeouts.
	http.DefaultTransport = &http.Transport{
		MaxIdleConns:          100,
		TLSHandshakeTimeout:   30 * time.Second,
		IdleConnTimeout:       15 * time.Minute,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	if cliArgs.insecure {
		log.Warning("potentially insecure HTTPS connection, only use for testing")
	}

	conn, err := httpclient.Dial(cliArgs.url, cliArgs.insecure)
	if err != nil {
		log.Fatal("dial:", err)
	}

	rsc := rsync.CreateRsyncClient(conn)
	rsc.Work()

	log.Info("Done")
}
