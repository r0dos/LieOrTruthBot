package storage

import (
	"LieOrTruthBot/internal/models/dto"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

const (
	sqlChatUserUPSERT = `INSERT INTO chat_user (chat_id, user_id) VALUES (?, ?)
    ON CONFLICT(chat_id, user_id) DO UPDATE 
	SET value = value + 1, updated_at = current_timestamp
	;`
	sqlChatUserSELECTValue = `SELECT value
	FROM chat_user
	WHERE chat_id = ? and user_id = ?
	;`
	sqlChatUserSELECTTop = `SELECT user_id, value 
	FROM chat_user
	WHERE chat_id = ?
	ORDER BY value DESC
	LIMIT ?
	;`

	sqlAdminsINSERT = `INSERT INTO admins (user_id) VALUES (?) 
                             ON CONFLICT DO NOTHING;`
	sqlAdminsEXISTS    = `SELECT EXISTS(SELECT 1 FROM admins WHERE user_id=?);`
	sqlQuestionsINSERT = `INSERT INTO questions (question, answer, user_id) VALUES (?, ?, ?)
                            ON CONFLICT DO NOTHING;`
	sqlQuestionSELECTRandom = `SELECT question, answer FROM questions
                        		ORDER BY random()
								LIMIT 1;`
)

const migrationDir = "migrations/sqlite"

//go:embed migrations/sqlite/*.sql
var embedMigrations embed.FS

func up(db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set dialect: %v", err)
	}

	if err := goose.Up(db, migrationDir); err != nil {
		return fmt.Errorf("up: %v", err)
	}

	return nil
}

func (s *anyStorage) AddAdmin(userID int64) error {
	_, err := s.pool.Exec(sqlAdminsINSERT, userID)
	if err != nil {
		return fmt.Errorf("insert: %v", err)
	}

	return nil
}

func (s *anyStorage) CheckAdmin(userID int64) (bool, error) {
	var value int64

	if err := s.pool.QueryRow(sqlAdminsEXISTS, userID).Scan(&value); err != nil {
		return false, fmt.Errorf("select: %v", err)
	}

	if value == 1 {
		return true, nil
	}

	return false, nil
}

func (s *anyStorage) AddQuestion(question string, answer bool, userID string) error {
	var ans int

	if answer {
		ans = 1
	}

	_, err := s.pool.Exec(sqlQuestionsINSERT, question, ans, userID)
	if err != nil {
		return fmt.Errorf("insert: %v", err)
	}

	return nil
}

func (s *anyStorage) GetTop(chatID, limit int64) ([]dto.ChartItem, error) {
	rows, err := s.pool.Query(sqlChatUserSELECTTop, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("select top: %v", err)
	}
	defer rows.Close()

	var chart []dto.ChartItem

	for rows.Next() {
		var item dto.ChartItem

		if err := rows.Scan(&item.UserID, &item.Value); err != nil {
			return nil, fmt.Errorf("row scan: %v", err)
		}

		chart = append(chart, item)
	}

	return chart, nil
}

func (s *anyStorage) GetQuestion() (string, bool, error) {
	var (
		question string
		answer   int
	)

	if err := s.pool.QueryRow(sqlQuestionSELECTRandom).Scan(&question, &answer); err != nil {
		return "", false, fmt.Errorf("select: %v", err)
	}

	if answer == 1 {
		return question, true, nil
	}

	return question, false, nil
}

func (s *anyStorage) IncValue(chatID, userID int64) error {
	_, err := s.pool.Exec(sqlChatUserUPSERT, chatID, userID)
	if err != nil {
		return fmt.Errorf("insert: %v", err)
	}

	return nil
}
