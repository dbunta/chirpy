package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/dbunta/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	dbQueries := database.New(db)
	mux := http.NewServeMux()
	apiConfig := apiConfig{}
	apiConfig.dbQueries = dbQueries
	apiConfig.platform = os.Getenv("PLATFORM")
	//mux.Handle("/app/", apiConfig.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	//mux.Handle("/app", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	handler := handlerMain()
	mux.Handle("/app/", apiConfig.middlewareMetricsInc(handler))
	mux.HandleFunc("GET /api/healthz", handlerHealthz)
	mux.HandleFunc("GET /admin/metrics", apiConfig.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiConfig.handlerReset)
	mux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)
	mux.HandleFunc("POST /api/users", apiConfig.handlerCreateUser)

	server := &http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	//go server.ListenAndServe()

	err = server.ListenAndServe()
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
	dbQueries      *database.Queries
	platform       string
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
	if cfg.platform != "dev" {
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		rw.WriteHeader(403)
		return
	}
	err := cfg.dbQueries.DeleteUsers(req.Context())
	if err != nil {
		res := errorRes{
			Error: "Something went wrong",
		}
		dat, _ := json.Marshal(res)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(500)
		rw.Write(dat)
		return
	}
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(200)
	cfg.fileServerHits.Store(0)
}

type errorRes struct {
	Error string `json:"error"`
}

func handlerValidateChirp(rw http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
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

func (cfg *apiConfig) handlerCreateUser(rw http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}
	type successRes struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		res := errorRes{
			Error: "Something went wrong",
		}
		dat, _ := json.Marshal(res)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(500)
		rw.Write(dat)
		return
	}

	user, err := cfg.dbQueries.CreateUser(req.Context(), params.Email)
	if err != nil {
		res := errorRes{
			Error: "Something went wrong",
		}
		dat, _ := json.Marshal(res)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(500)
		rw.Write(dat)
		return
	}

	newUser := successRes{
		Id:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}
	dat, _ := json.Marshal(newUser)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(201)
	rw.Write(dat)
	return
}
