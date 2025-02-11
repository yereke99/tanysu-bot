package repository

import (
	"database/sql"
	"fmt"
)

// User пайдаланушының деректерін сақтайды.
type User struct {
	UserID       int64  // Telegram-дағы ID
	Ava          string // Аватар жолы (мысалы, "./ava/...")
	AvaFileID    string // Telegram-дан алынған файл ID
	UserNickname string // Қолданушы енгізген никнейм
	UserName     string // Telegram-дағы қолданушы аты
	UserAge      int    // Жас (кейін толтырылады)
	UserSex      string // Жыныс (кейін толтырылады)
	UserGeo      string // Геолокация (кейін толтырылады)
	FirstName    string // Telegram-дағы аты
	LastName     string // Telegram-дағы тегі
	Contact      string // Байланыс (бар болса)
}

// UserRepository пайдаланушы деректерін БД-мен жұмыс істейді.
type UserRepository struct {
	db *sql.DB
}

// NewRepository жаңа UserRepository-ді құрайды.
func NewRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// UserExists берілген userID-мен қолданушының бар-жоғын тексереді.
func (r *UserRepository) UserExists(userID int64) (bool, error) {
	query := `SELECT COUNT(*) FROM users WHERE user_id = ?`
	var count int
	if err := r.db.QueryRow(query, userID).Scan(&count); err != nil {
		return false, fmt.Errorf("UserExists қатесі: %w", err)
	}
	return count > 0, nil
}

// InsertUser алғашқы қолданушы деректерін енгізеді.
func (r *UserRepository) InsertUser(user *User) error {
	query := `
		INSERT OR IGNORE INTO users (
			user_id,
			ava,
			ava_file_id,
			user_nickname,
			user_name,
			user_age,
			user_sex,
			user_geo,
			first_name,
			last_name,
			contact
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query,
		user.UserID,
		user.Ava,
		user.AvaFileID,
		user.UserNickname,
		user.UserName,
		user.UserAge,
		user.UserSex,
		user.UserGeo,
		user.FirstName,
		user.LastName,
		user.Contact,
	)
	if err != nil {
		return fmt.Errorf("InsertUser қатесі: %w", err)
	}
	return nil
}

// GetUser userID бойынша қолданушыны қайтарады.
func (r *UserRepository) GetUser(userID int64) (*User, error) {
	query := `
		SELECT user_id, ava, ava_file_id, user_nickname, user_name, user_age, user_sex, user_geo, first_name, last_name, contact
		FROM users WHERE user_id = ?
	`
	var user User
	err := r.db.QueryRow(query, userID).Scan(
		&user.UserID,
		&user.Ava,
		&user.AvaFileID,
		&user.UserNickname,
		&user.UserName,
		&user.UserAge,
		&user.UserSex,
		&user.UserGeo,
		&user.FirstName,
		&user.LastName,
		&user.Contact,
	)
	if err != nil {
		return nil, fmt.Errorf("GetUser қатесі: %w", err)
	}
	return &user, nil
}

// UpdateNickname қолданушының никнеймін жаңартады.
func (r *UserRepository) UpdateNickname(userID int64, nickname string) error {
	query := `UPDATE users SET user_nickname = ? WHERE user_id = ?`
	_, err := r.db.Exec(query, nickname, userID)
	if err != nil {
		return fmt.Errorf("UpdateNickname қатесі: %w", err)
	}
	return nil
}

// UpdateAvatar қолданушының аватар деректерін жаңартады.
func (r *UserRepository) UpdateAvatar(userID int64, avaPath, avaFileID string) error {
	query := `UPDATE users SET ava = ?, ava_file_id = ? WHERE user_id = ?`
	_, err := r.db.Exec(query, avaPath, avaFileID, userID)
	if err != nil {
		return fmt.Errorf("UpdateAvatar қатесі: %w", err)
	}
	return nil
}

// UpdateUserSex қолданушының жынысын жаңартады.
func (r *UserRepository) UpdateUserSex(userID int64, sex string) error {
	query := `UPDATE users SET user_sex = ? WHERE user_id = ?`
	_, err := r.db.Exec(query, sex, userID)
	if err != nil {
		return fmt.Errorf("UpdateUserSex қатесі: %w", err)
	}
	return nil
}

// UpdateUserAge обновляет возраст пользователя в базе данных.
func (r *UserRepository) UpdateUserAge(userID int64, age int) error {
	query := `UPDATE users SET user_age = ? WHERE user_id = ?`
	_, err := r.db.Exec(query, age, userID)
	if err != nil {
		return fmt.Errorf("UpdateUserAge: %w", err)
	}
	return nil
}

// UpdateUserGeo қолданушының геолокациясын жаңартады.
func (r *UserRepository) UpdateUserGeo(userID int64, geo string) error {
	query := `UPDATE users SET user_geo = ? WHERE user_id = ?`
	_, err := r.db.Exec(query, geo, userID)
	if err != nil {
		return fmt.Errorf("UpdateUserGeo қатесі: %w", err)
	}
	return nil
}
