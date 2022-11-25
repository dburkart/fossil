package server

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

type Server struct {
	log     zerolog.Logger
	metrics MetricsStore

	collectionPort int
	databasePort   int
	metricsPort    int
}

func New(log zerolog.Logger, collectionPort, databasePort, metricsPort int) Server {
	return Server{
		log,
		NewMetricsStore(),
		collectionPort,
		databasePort,
		metricsPort,
	}
}

func (s Server) ServeDatabase() {

}

func (s Server) ServeMetrics() {
	s.log.Info().Int("port", s.metricsPort).Msg("/metrics endpoint started")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(fmt.Sprintf(":%d", s.metricsPort), nil)
}
