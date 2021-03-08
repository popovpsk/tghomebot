package utils

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

//StartSystemServer http server with metrics/healtcheks
func StartSystemServer(logger *logrus.Logger, systemPort int) {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", func(wr http.ResponseWriter, req *http.Request) {
		wr.WriteHeader(200)
		wr.Write([]byte("Ok."))
	})
	if err := http.ListenAndServe(fmt.Sprintf(":%v", systemPort), nil); err != nil {
		logger.Panic(err)
	}
}
