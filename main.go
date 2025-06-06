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

	"github.com/dbunta/chirpy/internal/auth"
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
	apiConfig.secret = os.Getenv("SECRET")
	//mux.Handle("/app/", apiConfig.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	//mux.Handle("/app", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	handler := handlerMain()
	mux.Handle("/app/", apiConfig.middlewareMetricsInc(handler))
	mux.HandleFunc("GET /api/healthz", handlerHealthz)
	mux.HandleFunc("GET /admin/metrics", apiConfig.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiConfig.handlerReset)
	mux.HandleFunc("POST /api/users", apiConfig.handlerCreateUser)
	mux.HandleFunc("PUT /api/users", apiConfig.handlerUpdateUser)
	mux.HandleFunc("POST /api/chirps", apiConfig.handlerCreateChirp)
	mux.HandleFunc("GET /api/chirps", apiConfig.handlerGetAllChirps)
	mux.HandleFunc("GET /api/chirps/{chirpId}", apiConfig.handlerGetChirp)
	mux.HandleFunc("POST /api/login", apiConfig.handlerLogin)
	mux.HandleFunc("POST /api/refresh", apiConfig.handlerRefresh)
	mux.HandleFunc("POST /api/revoke", apiConfig.handlerRevoke)

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
	secret         string
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
		log.Printf("%s", err)
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

func (cfg *apiConfig) handlerCreateUser(rw http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
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
		log.Printf("%s", err)
		res := errorRes{
			Error: "Something went wrong",
		}
		dat, _ := json.Marshal(res)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(500)
		rw.Write(dat)
		return
	}

	hashedPassword, _ := auth.HashPassword(params.Password)
	createUserParams := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
	}
	user, err := cfg.dbQueries.CreateUser(req.Context(), createUserParams)
	if err != nil {
		log.Printf("%s", err)
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

func (cfg *apiConfig) handlerCreateChirp(rw http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	type successRes struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserId    uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)

		res := errorRes{
			Error: "Something went wrong here",
		}
		dat, _ := json.Marshal(res)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(500)
		rw.Write(dat)
		return
	}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		log.Printf("authorization error: %v", err)
		res := errorRes{
			Error: "authorization error 1",
		}
		dat, _ := json.Marshal(res)
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(401)
		rw.Write(dat)
		return
	}

	fmt.Printf("\n request header: %v\n", req.Header)
	fmt.Printf("\n create chirps token: %v\n", token)
	userId, err := auth.ValidateJWT(token, cfg.secret)

	if err != nil {
		log.Printf("authorization error: %v", err)
		res := errorRes{
			Error: "authorization error 2",
		}
		dat, _ := json.Marshal(res)
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(401)
		rw.Write(dat)
		return
	}

	if len(params.Body) > 140 {
		log.Printf("Chirp is too long")
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

	newChirpParams := database.CreateChirpParams{
		Body:   params.Body,
		UserID: userId,
	}

	newChirp, err := cfg.dbQueries.CreateChirp(req.Context(), newChirpParams)
	if err != nil {
		log.Printf("%s", err)
		res := errorRes{
			Error: "Something went wrong here instead",
		}
		dat, _ := json.Marshal(res)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(500)
		rw.Write(dat)
		return
	}

	res := successRes{
		Id:        newChirp.ID,
		CreatedAt: newChirp.CreatedAt,
		UpdatedAt: newChirp.UpdatedAt,
		Body:      newChirp.Body,
		UserId:    newChirp.UserID,
	}

	dat, _ := json.Marshal(res)
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(201)
	rw.Write(dat)
}

func (cfg *apiConfig) handlerGetAllChirps(rw http.ResponseWriter, req *http.Request) {
	type successRes struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserId    uuid.UUID `json:"user_id"`
	}
	chirps, err := cfg.dbQueries.GetAllChirps(req.Context())
	if err != nil {
		log.Printf("%s", err)
		res := errorRes{
			Error: "Something went wrong here instead",
		}
		dat, _ := json.Marshal(res)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(500)
		rw.Write(dat)
		return
	}

	var retval []successRes
	for _, val := range chirps {
		res := successRes{
			Id:        val.ID,
			CreatedAt: val.CreatedAt,
			UpdatedAt: val.UpdatedAt,
			Body:      val.Body,
			UserId:    val.UserID,
		}
		retval = append(retval, res)
	}

	dat, _ := json.Marshal(retval)
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(200)
	rw.Write(dat)
}

func (cfg *apiConfig) handlerGetChirp(rw http.ResponseWriter, req *http.Request) {
	type successRes struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserId    uuid.UUID `json:"user_id"`
	}

	chirpId, _ := uuid.Parse(req.PathValue("chirpId"))

	chirp, err := cfg.dbQueries.GetChirp(req.Context(), chirpId)
	if err != nil {
		log.Printf("error fetching chirp: %w", err)
		res := errorRes{
			Error: "error fetching chirp",
		}
		dat, _ := json.Marshal(res)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(404)
		rw.Write(dat)
		return
	}

	retval := successRes{
		Id:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserId:    chirp.UserID,
	}
	dat, _ := json.Marshal(retval)
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(200)
	rw.Write(dat)
}

func (cfg *apiConfig) handlerLogin(rw http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
	}
	decoder := json.NewDecoder(req.Body)
	var params parameters
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		res := errorRes{
			Error: "Something went wrong here",
		}
		dat, _ := json.Marshal(res)
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(500)
		rw.Write(dat)
		return
	}

	if params.ExpiresInSeconds == 0 || params.ExpiresInSeconds > 3600 {
		params.ExpiresInSeconds = 3600
	}

	user, err := cfg.dbQueries.GetUser(req.Context(), params.Email)
	if err != nil {
		log.Printf("Error getting user: %w", err)
		res := errorRes{
			Error: "Something went wrong",
		}
		dat, _ := json.Marshal(res)
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(500)
		rw.Write(dat)
		return
	}

	err = auth.CheckPassword(user.HashedPassword, params.Password)
	if err != nil {
		res := errorRes{
			Error: "Incorrect password",
		}
		dat, _ := json.Marshal(res)
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(401)
		rw.Write(dat)
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.secret, time.Second*time.Duration(params.ExpiresInSeconds))
	if err != nil {
		ReturnError(rw, "Error generating token", 401)
		return
	}

	refreshToken, _ := auth.MakeRefreshToken()
	createRefreshTokenParams := database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(time.Hour * time.Duration(1)),
	}
	_, err = cfg.dbQueries.CreateRefreshToken(req.Context(), createRefreshTokenParams)
	if err != nil {
		fmt.Printf("%v\n", err)
		ReturnError(rw, "error saving new token", 500)
		return
	}

	type successRes struct {
		Id           uuid.UUID `json:"id"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		Token        string    `json:"token"`
		RefreshToken string    `json:"refresh_token"`
	}

	res := successRes{
		Id:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        token,
		RefreshToken: refreshToken,
	}
	dat, _ := json.Marshal(res)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(200)
	rw.Write(dat)
}

func ReturnError(rw http.ResponseWriter, error string, status int) {
	res := errorRes{
		Error: error,
	}
	dat, _ := json.Marshal(res)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	rw.Write(dat)
}

func (cfg *apiConfig) handlerRefresh(rw http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		fmt.Printf("%v\n", err)
		ReturnError(rw, "error getting bearer token", 401)
		return
	}
	rt, err := cfg.dbQueries.GetRefreshToken(req.Context(), token)
	if err != nil {
		fmt.Printf("%v\n", err)
		ReturnError(rw, "error validating token", 401)
		return
	}

	if time.Now().UTC().After(rt.ExpiresAt) {
		fmt.Print("refresh token expired\n")
		ReturnError(rw, "refresh token expired", 401)
		return
	}
	if rt.RevokedAt.Valid {
		fmt.Print("refresh token revoked\n")
		ReturnError(rw, "refresh token revoked", 401)
		return
	}

	newToken, err := auth.MakeJWT(rt.UserID, cfg.secret, time.Hour*time.Duration(1))
	if err != nil {
		fmt.Printf("%v\n", err)
		ReturnError(rw, "error making refresh token", 500)
		return
	}

	type successRes struct {
		Token string `json:"token"`
	}
	retval := successRes{
		Token: newToken,
	}

	fmt.Printf("\n-------refreshed token: %v------\n", retval.Token)

	dat, _ := json.Marshal(retval)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(200)
	rw.Write(dat)
}

func (cfg *apiConfig) handlerRevoke(rw http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		fmt.Printf("%v\n", err)
		ReturnError(rw, "error getting bearer token", 401)
		return
	}
	err = cfg.dbQueries.RevokeRefreshToken(req.Context(), token)
	if err != nil {
		fmt.Printf("%v\n", err)
		ReturnError(rw, "error getting bearer token", 401)
		return
	}

	rw.WriteHeader(204)
}

func (cfg *apiConfig) handlerUpdateUser(rw http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		fmt.Printf("%v\n", err)
		ReturnError(rw, "error getting bearer token", 401)
		return
	}

	userId, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		fmt.Printf("%v\n", err)
		ReturnError(rw, "error getting bearer token", 401)
		return
	}

	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(req.Body)
	var params parameters
	err = decoder.Decode(&params)
	if err != nil {
		fmt.Printf("%v\n", err)
		ReturnError(rw, "error decoding parameters", 500)
		return
	}

	pwHash, err := auth.HashPassword(params.Password)
	if err != nil {
		fmt.Printf("%v\n", err)
		ReturnError(rw, "error hashing password", 500)
		return
	}

	updateUserParams := database.UpdateUserParams{
		ID:             userId,
		Email:          params.Email,
		HashedPassword: pwHash,
	}
	updatedUser, err := cfg.dbQueries.UpdateUser(req.Context(), updateUserParams)
	if err != nil {
		fmt.Printf("%v\n", err)
		ReturnError(rw, "error updating user", 500)
		return
	}

	type SuccessRes struct {
		ID             uuid.UUID `json:"id"`
		CreatedAt      time.Time `json:"created_at"`
		UpdatedAt      time.Time `json:"updated_at"`
		Email          string    `json:"email"`
		HashedPassword string    `json:"hashed_password"`
	}
	retval := SuccessRes{
		ID:             updatedUser.ID,
		CreatedAt:      updatedUser.CreatedAt,
		UpdatedAt:      updatedUser.UpdatedAt,
		Email:          updatedUser.Email,
		HashedPassword: updatedUser.HashedPassword,
	}

	dat, _ := json.Marshal(retval)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(200)
	rw.Write(dat)
}
