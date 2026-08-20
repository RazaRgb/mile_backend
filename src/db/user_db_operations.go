package db

import (
	"backend/src/models"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func UserExists(email string, tx ...pgx.Tx) (bool, error) {
	userQuery := `
	SELECT EXISTS (
		SELECT 1
		FROM users
		WHERE email = $1
	)
	`
	var row pgx.Row
	if len(tx) == 0 {
		row = DB.QueryRow(context.Background(), userQuery, email)
	} else {
		row = tx[0].QueryRow(context.Background(), userQuery, email)
	}

	var exists bool
	err := row.Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("userExists query failed %w", err)
	}

	return exists, nil
}

// UserExistsByID reports whether a user with the given id exists.
func UserExistsByID(userID uuid.UUID, tx ...pgx.Tx) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`
	var row pgx.Row
	if len(tx) == 0 {
		row = DB.QueryRow(context.Background(), query, userID)
	} else {
		row = tx[0].QueryRow(context.Background(), query, userID)
	}

	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("userExistsByID query failed: %w", err)
	}
	return exists, nil
}

func InsertUser(newUser models.User, tx ...pgx.Tx) error {
	insertUserQuery := `
	INSERT INTO users (email, username, pass_hash, id)
	VALUES ($1,$2,$3,$4)
	`
	var err error
	if len(tx) == 0 {
		_, err = DB.Exec(
			context.Background(),
			insertUserQuery,
			newUser.Email,
			newUser.Username,
			newUser.HashedPassword,
			newUser.ID,
		)
	} else {
		_, err = tx[0].Exec(
			context.Background(),
			insertUserQuery,
			newUser.Email,
			newUser.Username,
			newUser.HashedPassword,
			newUser.ID,
		)
	}

	if err != nil {
		fmt.Printf("failed to create user %s\n", newUser.Username)
		return fmt.Errorf("failed to create user %w\n", err)
	}
	return nil
}

func GetUser(email string, tx ...pgx.Tx) (models.User, error) {
	selectQuery := `
	SELECT email, username, pass_hash, created_at, id
	FROM users
	WHERE email = $1
	`
	var user models.User
	var row pgx.Row

	if len(tx) == 0 {
		row = DB.QueryRow(context.Background(), selectQuery, email)
	} else {
		row = tx[0].QueryRow(context.Background(), selectQuery, email)
	}
	err := row.Scan(
		&user.Email,
		&user.Username,
		&user.HashedPassword,
		&user.CreatedAt,
		&user.ID,
	)
	if err != nil {
		return user, fmt.Errorf("unable to get user %w", err)
	}

	return user, nil
}
