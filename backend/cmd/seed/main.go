// Package main is the seeder command. It inserts one user per role
// (requester, agent, admin) with the shared password "password" hashed
// with bcrypt (cost 12), matching the auth service. It bypasses register
// validation since "password" lacks a digit.
//
// Run inside the backend container:
//
//	docker compose exec backend /app/seed
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	"simplehelp/src/config"
	"simplehelp/src/db"
	"simplehelp/src/models"
	"simplehelp/src/repositories"
)

const seedPassword = "password"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, cfg.DSN(), 20*time.Second)
	if err != nil {
		slog.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	hash, err := bcrypt.GenerateFromPassword([]byte(seedPassword), 12)
	if err != nil {
		slog.Error("bcrypt hash failed", "error", err)
		os.Exit(1)
	}

	repo := repositories.NewUserRepository(database)

	seedUsers := []struct {
		name, email, role string
	}{
		{"Requester User", "requester@example.com", models.RoleRequester},
		{"Agent User", "agent@example.com", models.RoleAgent},
		{"Admin User", "admin@example.com", models.RoleAdmin},
	}

	for _, u := range seedUsers {
		// Skip if a user with this email already exists.
		_, err := repo.GetByEmail(ctx, u.email)
		if err == nil {
			slog.Info("user already exists, skipping", "email", u.email, "role", u.role)
			continue
		}
		if !errors.Is(err, repositories.ErrNotFound) {
			slog.Error("lookup user failed", "email", u.email, "error", err)
			os.Exit(1)
		}

		created, err := repo.Create(ctx, u.name, u.email, string(hash), u.role)
		if err != nil {
			slog.Error("create user failed", "email", u.email, "error", err)
			os.Exit(1)
		}
		slog.Info("seeded user", "id", created.ID, "email", created.Email, "role", created.Role)
	}

	fmt.Println("seed complete")
}
