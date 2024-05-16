package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	// Para que no de error al utilizar LoadLocation(Europe/Madrid)
	_ "time/tzdata"
)

type key int

const (
	requestIDKey key = 0
)

var (
	listenAddr string
	healthy    int32
	Version    string = "dev"
	Fecha      string
	Commit     string
	Source     string
)

var (
	//go:embed static/*
	files embed.FS
)

type renderContext struct {
	Headers   map[string]string
	Version   string
	Fecha     string
	Commit    string
	Source    string
	RequestID string
	CallerIP  string
	HostName  string
}

func main() {
	flag.StringVar(&listenAddr, "binding", "0.0.0.0:3100", "Server listen address")
	flag.Parse()

	cancel := make(chan os.Signal)
	signal.Notify(cancel, os.Interrupt, syscall.SIGTERM)

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Printf("Servidon levantando en %s...\n", listenAddr)

	router := http.NewServeMux()
	router.Handle("/favicon.ico", http.FileServer(http.Dir("./static")))
	router.HandleFunc("/", handler)

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      logging(logger)(router),
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		logger.Println("Servidor apagando...")
		atomic.StoreInt32(&healthy, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logger.Fatalf("No se ha podido parar el servidor de manera limpia: %v\n", err)
		}
		close(done)
	}()

	logger.Println("Servidor preparado para recibir solicitudes en ", listenAddr)
	atomic.StoreInt32(&healthy, 1)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("No se puede escuchar en %s: %v\n", listenAddr, err)
	}

	<-done
	logger.Println("Servidor detenido")
}

func handler(w http.ResponseWriter, r *http.Request) {
	requestID := fmt.Sprintf("%d", time.Now().UnixNano())
	w.Header().Set("X-Request-Id", requestID)

	var err error
	templ, err := template.New("index.html").ParseFS(files, "static/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for name, values := range r.Header {
		for _, value := range values {
			fmt.Println(name, value)
		}
	}

	headers := make(map[string]string)

	for name, values := range r.Header {
		headers[name] = strings.Join(values, ",")
	}

	tmpFecha, _ := strconv.ParseInt(Fecha, 10, 64)

	location, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hostname, err := os.Hostname()
	err = templ.Execute(w, renderContext{
		Headers:   headers,
		Version:   Version,
		Fecha:     time.Unix(tmpFecha, 0).Local().In(location).Format(time.RFC3339),
		Commit:    Commit,
		Source:    Source,
		RequestID: requestID,
		CallerIP:  r.RemoteAddr,
		HostName:  hostname,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}
