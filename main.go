package main

import (
	"os"
	"os/signal"
	"syscall"
	"flag"
	"time"
	"github.com/artd-status-updater/statusupdater"
)

func parseArguments() (string, *status_updater.EtcdConnectionParams, *status_updater.KeyUpdaterParameters) {
	var unixSocketPath string
	etcdParams := status_updater.EtcdConnectionParams{}
	keyUpdaterParams := status_updater.KeyUpdaterParameters{}

	// etcd connection params
	flag.StringVar(&etcdParams.CertFile, "cert-file", "", "identify HTTPS client using this SSL certificate file")
	flag.StringVar(&etcdParams.KeyFile, "key-file", "", "identify HTTPS client using this SSL key file")
	flag.StringVar(&etcdParams.CaFile, "ca-file", "", "verify certificates of HTTPS-enabled servers using this CA bundle")
	// StringVarP(Name: "username, u", Value: "", Usage: "provide username[:password] and prompt if password is not supplied.")
	flag.DurationVar(&etcdParams.ConnectionTimeout, "timeout", time.Second, "connection timeout per request")
	flag.DurationVar(&etcdParams.RequestTimeout, "total-timeout", 5 * time.Second, "timeout for the command execution")

	// Server socket path
	flag.StringVar(&unixSocketPath, "socket", "/tmp/socket", "Path to unix socket liten on")

	// Key updater parameters
	flag.StringVar(&keyUpdaterParams.Key, "key", "/artifact-downloader/status", "Key where to push status")
	flag.DurationVar(&keyUpdaterParams.KeyTTL, "key-ttl", 10 * time.Second, "TTL for status key")
	flag.DurationVar(&keyUpdaterParams.RetryFreq, "key-retry-freq", 1 * time.Second, "Key update retry update freq")
	flag.DurationVar(&keyUpdaterParams.UpdateFreq, "key-update-freq", 5 * time.Second, "Key update freq")

	flag.Parse()

	return unixSocketPath, &etcdParams, &keyUpdaterParams
}

func main() {
	unixSocketPath, etcdParams, keyUpdaterParams := parseArguments()

	etcdKApi, err := status_updater.MakeNewEtcdKApi(etcdParams)
	if err != nil {
		panic(err)
	}

	dataChan := make(chan string)
	errorChan := make(chan error)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill, syscall.SIGTERM)

	dataListener := status_updater.NewDataListener(unixSocketPath, dataChan, errorChan)
	// start server
	err = dataListener.Start()
	if err != nil {
		panic(err)
	}

	// start key updater
	keyUpdater := status_updater.NewKeyUpdater(keyUpdaterParams, etcdKApi, dataChan, make(chan error));
	keyUpdater.Start()


	select {
	case err = <-errorChan:
		panic(err)
	case <-signalChan:
		dataListener.Stop()
		keyUpdater.Stop()
	}
}
