// Package health implements an HTTP handler that responds to health checks.
package health

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/plugin/pkg/reuseport"
)

var log = clog.NewWithPlugin("health")

// Health implements healthchecks by exporting a HTTP endpoint.
type health struct {
	Addr      string
	lameduck  time.Duration
	lookup    []string
	healthURI *url.URL

	ln      net.Listener
	nlSetup bool
	mux     *http.ServeMux

	stop context.CancelFunc
}

func (h *health) OnStartup() error {
	if h.Addr == "" {
		h.Addr = ":8080"
	}

	var err error
	h.healthURI, err = url.Parse("http://" + h.Addr)
	if err != nil {
		return err
	}

	h.healthURI.Path = "/health"
	if h.healthURI.Host == "" {
		// while we can listen on multiple network interfaces, we need to pick one to poll
		h.healthURI.Host = "localhost"
	}

	ln, err := reuseport.Listen("tcp", h.Addr)
	if err != nil {
		return err
	}

	h.ln = ln
	h.mux = http.NewServeMux()
	h.nlSetup = true

	h.mux.HandleFunc(h.healthURI.Path, func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK

		if len(h.lookup) > 0 {
			for _, l := range h.lookup {
				log.Infof("Looking up %s", l)
				_, err := net.LookupHost(l)
				if err != nil {
					log.Errorf("Lookup for %s failed: %s", l, err)
					status = http.StatusInternalServerError
					break
				}
				log.Infof("Lookup for %s succeeded", l)
			}
		}

		w.WriteHeader(status)
		io.WriteString(w, http.StatusText(status))
	})

	ctx := context.Background()
	ctx, h.stop = context.WithCancel(ctx)

	go func() { http.Serve(h.ln, h.mux) }()
	go func() { h.overloaded(ctx) }()

	return nil
}

func (h *health) OnFinalShutdown() error {
	if !h.nlSetup {
		return nil
	}

	if h.lameduck > 0 {
		log.Infof("Going into lameduck mode for %s", h.lameduck)
		time.Sleep(h.lameduck)
	}

	h.stop()

	h.ln.Close()
	h.nlSetup = false
	return nil
}

func (h *health) OnReload() error {
	if !h.nlSetup {
		return nil
	}

	h.stop()

	h.ln.Close()
	h.nlSetup = false
	return nil
}
