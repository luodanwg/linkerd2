package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/linkerd/linkerd2/controller/api/public"
	"github.com/linkerd/linkerd2/controller/k8s"
	"github.com/linkerd/linkerd2/controller/tap"
	"github.com/linkerd/linkerd2/pkg/admin"
	"github.com/linkerd/linkerd2/pkg/version"
	promApi "github.com/prometheus/client_golang/api"
	log "github.com/sirupsen/logrus"
)

func main() {
	addr := flag.String("addr", ":8085", "address to serve on")
	kubeConfigPath := flag.String("kubeconfig", "", "path to kube config")
	prometheusUrl := flag.String("prometheus-url", "http://127.0.0.1:9090", "prometheus url")
	metricsAddr := flag.String("metrics-addr", ":9995", "address to serve scrapable metrics on")
	tapAddr := flag.String("tap-addr", "127.0.0.1:8088", "address of tap service")
	controllerNamespace := flag.String("controller-namespace", "linkerd", "namespace in which Linkerd is installed")
	ignoredNamespaces := flag.String("ignore-namespaces", "kube-system", "comma separated list of namespaces to not list pods from")
	logLevel := flag.String("log-level", log.InfoLevel.String(), "log level, must be one of: panic, fatal, error, warn, info, debug")
	printVersion := version.VersionFlag()
	flag.Parse()

	// set global log level
	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatalf("invalid log-level: %s", *logLevel)
	}
	log.SetLevel(level)

	version.MaybePrintVersionAndExit(*printVersion)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	tapClient, tapConn, err := tap.NewClient(*tapAddr)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer tapConn.Close()

	k8sClient, err := k8s.NewClientSet(*kubeConfigPath)
	if err != nil {
		log.Fatal(err.Error())
	}
	k8sAPI := k8s.NewAPI(
		k8sClient,
		k8s.Deploy,
		k8s.NS,
		k8s.Pod,
		k8s.RC,
		k8s.RS,
		k8s.Svc,
	)

	prometheusClient, err := promApi.NewClient(promApi.Config{Address: *prometheusUrl})
	if err != nil {
		log.Fatal(err.Error())
	}

	server := public.NewServer(
		*addr,
		prometheusClient,
		tapClient,
		k8sAPI,
		*controllerNamespace,
		strings.Split(*ignoredNamespaces, ","),
	)

	ready := make(chan struct{})

	go k8sAPI.Sync(ready)

	go func() {
		log.Infof("starting HTTP server on %+v", *addr)
		server.ListenAndServe()
	}()

	go admin.StartServer(*metricsAddr, ready)

	<-stop

	log.Infof("shutting down HTTP server on %+v", *addr)
	server.Shutdown(context.Background())
}
