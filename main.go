package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

func main() {
	mux := http.NewServeMux()
	apiConfig := apiConfig{}
	//mux.Handle("/app/", apiConfig.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	//mux.Handle("/app", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	handler := handlerMain()
	mux.Handle("/app/", apiConfig.middlewareMetricsInc(handler))
	mux.HandleFunc("GET /healthz", handlerHealthz)
	mux.HandleFunc("GET /metrics", apiConfig.handlerMetrics)
	mux.HandleFunc("POST /reset", apiConfig.handlerReset)

	server := &http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	//go server.ListenAndServe()

	err := server.ListenAndServe()
	if err != nil {
		fmt.Printf("%w", err)
	}
}

func handlerMain() http.Handler {
	return http.StripPrefix("/app", http.FileServer(http.Dir(".")))
}

func handlerHealthz(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(200)
	rw.Write([]byte("OK"))
}

type apiConfig struct {
	fileServerHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerMetrics(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(200)
	rw.Write([]byte(fmt.Sprintf("Hits: %v", cfg.fileServerHits.Load())))
}

func (cfg *apiConfig) handlerReset(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(200)
	cfg.fileServerHits.Store(0)
}
