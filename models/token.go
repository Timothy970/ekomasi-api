package models

import (
	"database/sql"
	"errors"
	"time"

	"github.com/teris-io/shortid"
)

type ResetToken struct {
	ID        int
	Email     string
	Token     string
	Used      bool
	CreatedAt time.Time
	ExpiresAt time.Time
}

func StoreToken(userID string, token string, expiresAt time.Time) error {
	tokenID, _ := shortid.Generate()
	_, err := DB.Exec(`
		INSERT INTO user_tokens (id, user_id, token, expires_at)
		VALUES (?, ?, ?, ?)`,
		tokenID, userID, token, expiresAt,
	)
	return err
}
func StoreResetPasswordToken(email string, token string, expiresAt time.Time) error {
	_, err := DB.Exec(`
		INSERT INTO reset_tokens (email, token, expires_at)
		VALUES (?, ?, ?)`,
		email, token, expiresAt,
	)
	return err
}

func DeleteToken(token string) error {
	_, err := DB.Exec("DELETE FROM user_tokens WHERE token = ?", token)
	return err
}

func GetToken(token string) (string, error) {
	var tokenRecord string
	err := DB.QueryRow("SELECT token FROM user_tokens WHERE token = ?", token).Scan(&tokenRecord)
	if err != nil {
		return "", err
	}
	return tokenRecord, nil
}

// validate reset token
func GetValidResetToken(token string) (*ResetToken, error) {
	var rt ResetToken
	query := `SELECT id, email, token, used, created_at, expires_at 
              FROM reset_tokens 
              WHERE token = ?`

	row := DB.QueryRow(query, token)
	err := row.Scan(&rt.ID, &rt.Email, &rt.Token, &rt.Used, &rt.CreatedAt, &rt.ExpiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("reset token not found")
		}
		return nil, err
	}

	if rt.Used {
		return nil, errors.New("token already used")
	}

	if time.Now().After(rt.ExpiresAt) {
		return nil, errors.New("token has expired")
	}

	return &rt, nil
}
func MarkResetTokenUsed(token string) error {
	query := `UPDATE reset_tokens SET used = ? WHERE token = ?`
	_, err := DB.Exec(query, true, token)
	return err
}
