package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Kam1217/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type chirpResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handleHits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := fmt.Sprintf(
		`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())
	w.Write([]byte(html))
}

func (cfg *apiConfig) handleReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, 403, "Forbidden")
		return
	}
	cfg.fileserverHits.Store(0)
	err := cfg.db.DeleteUsers(r.Context())
	if err != nil {
		respondWithError(w, 500, "failed to delete users")
	}
	w.WriteHeader(200)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type returnError struct {
		Error string `json:"error"`
	}
	respError := returnError{
		Error: msg,
	}
	data, err := json.Marshal(respError)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func badWordReplacement(msg string, badWords map[string]struct{}) string {
	splitText := strings.Split(msg, " ")
	for i, word := range splitText {
		for badWord := range badWords {
			if strings.ToLower(word) == badWord {
				splitText[i] = "****"
			}
		}
	}
	cleanWord := strings.Join(splitText, " ")
	return cleanWord
}

func handleValidate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	type returnValid struct {
		BodyText string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "something went wrong")
	}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
	}

	badWordMap := map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}

	cleanedBody := badWordReplacement(params.Body, badWordMap)
	validResponse := returnValid{
		BodyText: cleanedBody,
	}

	respondWithJSON(w, 200, validResponse)
}

func (cfg *apiConfig) handleUsers(w http.ResponseWriter, r *http.Request) {
	type emailUser struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	eUser := emailUser{}
	err := decoder.Decode(&eUser)
	if err != nil {
		respondWithError(w, 500, "something went wrong")
	}

	data, err := cfg.db.CreateUser(r.Context(), eUser.Email)
	if err != nil {
		respondWithError(w, 500, "failed to create user")
	}

	user := User{
		ID:        data.ID,
		CreatedAt: data.CreatedAt,
		UpdatedAt: data.UpdatedAt,
		Email:     data.Email,
	}

	respondWithJSON(w, 201, user)
}

func (cfg *apiConfig) handleChirp(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	msg := chirp{}
	err := decoder.Decode(&msg)
	if err != nil {
		respondWithError(w, 500, "something went wrong")
	}

	if msg.Body == "" {
		respondWithError(w, 400, "chirp cannot be empty")
	}

	data, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   msg.Body,
		UserID: msg.UserID,
	})
	if err != nil {
		respondWithError(w, 500, "failed to create chirp")
	}

	response := chirpResponse{
		ID:        data.ID,
		CreatedAt: data.CreatedAt,
		UpdatedAt: data.UpdatedAt,
		Body:      data.Body,
		UserID:    data.UserID,
	}

	respondWithJSON(w, 201, response)
}

func (cfg *apiConfig) handleGetChirp(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		respondWithError(w, 500, "something went wrong")
	}

	var msg []chirpResponse

	for _, chirp := range chirps {
		response := chirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
		msg = append(msg, response)
	}
	respondWithJSON(w, 200, msg)
}

func (cfg *apiConfig) handleGetOneChirp(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(path)
	if err != nil {
		respondWithError(w, 400, "something went wrong")
		return
	}

	chirp, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err == sql.ErrNoRows {
		respondWithError(w, 404, "cannot find chirp")
		return
	}
	if err != nil {
		respondWithError(w, 500, "something went wrong")
		return
	}

	response := chirpResponse{
		ID:        chirpID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}
	respondWithJSON(w, 200, response)
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("failed to open a connection to the database:", err)
	}
	dbQueries := database.New(db)

	apiCfg := apiConfig{db: dbQueries, platform: platform}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handleGetOneChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.handleGetChirp)
	mux.HandleFunc("POST /api/chirps", apiCfg.handleChirp)
	mux.HandleFunc("POST /api/users", apiCfg.handleUsers)
	mux.HandleFunc("POST /api/validate_chirp", handleValidate)
	mux.HandleFunc("GET /api/healthz", handleHealth)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handleHits)
	mux.HandleFunc("POST /admin/reset", apiCfg.handleReset)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	server.ListenAndServe()
}
