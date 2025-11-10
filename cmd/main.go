package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	discoverycontroller "odin-service-discovery-controller/internal/discovery-controller"
	"odin-service-discovery-controller/internal/rest"
	"odin-service-discovery-controller/internal/util"
)

func main() {
	stopCh := make(chan struct{})
	defer close(stopCh)

	kubeconfig := flag.String("kubeconfig", "", "path to the kubeconfig file")
	workers := flag.Int("workers", 5, "number of processing workers")
	discoverybackend := flag.String("discoverybackend", "", "odin discovery service address")
	batchsize := flag.Int("batchsize", 5, "request batch size")
	orgid := flag.String("orgid", "", "organisation id")
	accountname := flag.String("accountname", "", "account name")
	batchpushtime := flag.Duration("batchpushtimeout", 500*time.Millisecond, "batch push timeout in milliseconds")
	flag.Parse()
	log.Infof("Starting discovery controller with args: kubeconfig=%s, workers=%d, discoverybackend=%s, accountname=%s , batchsize=%s ,batchPushTimeout=%s ", *kubeconfig, *workers, *discoverybackend, *accountname, *batchsize, *batchpushtime)

	config, err := util.GetKubeConfig(*kubeconfig)
	if err != nil {
		log.Error(err)
		panic(err.Error())
	}

	namespace, err := util.GetNamespaceFromEnv()
	if err != nil {
		namespace = corev1.NamespaceAll
	}
	webclient := rest.CreateWebClient(*discoverybackend, *batchpushtime, *batchsize, *orgid, *accountname)
	controller, err := discoverycontroller.NewDiscoveryController(config, namespace, webclient)
	if err != nil {
		log.Fatal(err)
	}

	webclient.Run()
	controller.Run(*workers, stopCh)

	// Wait for termination signals (CTRL+C)
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalCh
		close(stopCh)
	}()
	<-signalCh

	log.Infoln("Terminating...")
}
