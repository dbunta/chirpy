package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync/atomic"
)

func main() {
	mux := http.NewServeMux()
	apiConfig := apiConfig{}
	//mux.Handle("/app/", apiConfig.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	//mux.Handle("/app", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	handler := handlerMain()
	mux.Handle("/app/", apiConfig.middlewareMetricsInc(handler))
	mux.HandleFunc("GET /api/healthz", handlerHealthz)
	mux.HandleFunc("GET /admin/metrics", apiConfig.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiConfig.handlerReset)
	mux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)

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
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.WriteHeader(200)
	html := fmt.Sprintf(`
		<html>
		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>
		</html>	
	`, cfg.fileServerHits.Load())
	rw.Write([]byte(html))
}

func (cfg *apiConfig) handlerReset(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(200)
	cfg.fileServerHits.Store(0)
}

func handlerValidateChirp(rw http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	type errorRes struct {
		Error string `json:"error"`
	}
	type successRes struct {
		CleanedBody string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)

		res := errorRes{
			Error: "Something went wrong",
		}
		dat, _ := json.Marshal(res)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(500)
		rw.Write(dat)
		return
	}

	if len(params.Body) > 140 {
		res := errorRes{
			Error: "Chirp is too long",
		}
		dat, _ := json.Marshal(res)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(400)
		rw.Write(dat)
		return
	}

	replace := "****"
	re := regexp.MustCompile("(?i)" + "kerfuffle|sharbert|fornax")
	params.Body = re.ReplaceAllString(params.Body, replace)
	res := successRes{
		CleanedBody: params.Body,
	}
	dat, _ := json.Marshal(res)
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(200)
	rw.Write(dat)
}
