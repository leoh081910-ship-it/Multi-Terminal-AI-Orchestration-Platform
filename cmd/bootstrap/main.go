package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/auth"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	_ "modernc.org/sqlite"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	viper.SetDefault("database.path", "ai-orchestration.db")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("AIOP")

	dbPath := viper.GetString("database.path")
	client, err := ent.Open("sqlite", fmt.Sprintf("file:%s?_fk=1", dbPath))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open database")
	}
	defer client.Close()

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to create schema")
	}

	adminUsername := os.Getenv("ADMIN_USERNAME")
	if adminUsername == "" {
		adminUsername = "admin"
	}

	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@localhost"
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		log.Fatal().Msg("ADMIN_PASSWORD environment variable is required")
	}

	ctx := context.Background()

	existingUser, err := client.User.Query().Where().Only(ctx)
	if err == nil {
		log.Info().Str("username", existingUser.Username).Msg("Admin user already exists")
		return
	}

	user, err := client.User.Create().
		SetID(uuid.New().String()).
		SetUsername(adminUsername).
		SetEmail(adminEmail).
		SetPasswordHash(adminPassword).
		SetActive(true).
		Save(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create admin user")
	}

	log.Info().
		Str("id", user.ID).
		Str("username", user.Username).
		Str("email", user.Email).
		Msg("Admin user created")

	tokenRepo := auth.NewEntTokenRepository(client)
	tokenService := auth.NewTokenService(tokenRepo)

	token, rawToken, err := tokenService.GenerateToken(
		ctx,
		user.ID,
		"Initial Admin Token",
		"Bootstrap token for initial system access",
		[]string{"*"},
		nil,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to generate admin token")
	}

	log.Info().
		Str("token_id", token.ID).
		Str("token_name", token.Name).
		Msg("Admin token created")

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("ADMIN TOKEN (save this, it will not be shown again):")
	fmt.Println(rawToken)
	fmt.Println(strings.Repeat("=", 60) + "\n")
}
