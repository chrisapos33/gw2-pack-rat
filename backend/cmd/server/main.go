package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"

	"gw2packrat/internal/auth"
	"gw2packrat/internal/db"
	"gw2packrat/internal/handler"
)

func main() {
	dbURL := mustEnv("DATABASE_URL")
	jwtSecret := []byte(mustEnv("JWT_SECRET"))

	encKeyHex := mustEnv("ENCRYPTION_KEY")
	encKey, err := hex.DecodeString(encKeyHex)
	if err != nil || len(encKey) != 32 {
		log.Fatal("ENCRYPTION_KEY must be a 64-character hex string (32 bytes)")
	}

	pool, err := db.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	usersH := handler.NewUsersHandler(pool, jwtSecret)
	apiKeysH := handler.NewAPIKeysHandler(pool, encKey)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/signup", usersH.Signup)
	mux.HandleFunc("POST /auth/login", usersH.Login)
	mux.Handle("POST /api-keys", auth.Middleware(jwtSecret, http.HandlerFunc(apiKeysH.AddKey)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("listening on :%s\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
