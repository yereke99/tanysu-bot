package repository

import (
	"database/sql"
	"fmt"
)

// User представляет данные пользователя.
type User struct {
	UserID       int64  // ID пользователя (Telegram)
	Ava          string // Путь к файлу аватарки (например, "./ava/...")
	AvaFileID    string // FileID аватарки (полученный от Telegram)
	UserNickname string // Никнейм, который задаёт пользователь
	UserName     string // Имя пользователя (из Telegram)
	UserAge      int    // Возраст (может быть добавлен позже)
	UserSex      string // Пол (может быть добавлен позже)
	UserGeo      string // Геолокация (может быть добавлен позже)
	FirstName    string // Имя (из Telegram)
	LastName     string // Фамилия (из Telegram)
	Contact      string // Контакт (если есть)
}

// UserRepository работает с данными пользователей в БД PostgreSQL.
type UserRepository struct {
	db *sql.DB
}

// NewRepository создаёт новый экземпляр UserRepository.
func NewRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// UserExists проверяет, существует ли пользователь с данным userID.
func (r *UserRepository) UserExists(userID int64) (bool, error) {
	query := `SELECT COUNT(*) FROM users WHERE user_id = $1`
	var count int
	if err := r.db.QueryRow(query, userID).Scan(&count); err != nil {
		return false, fmt.Errorf("UserExists: %w", err)
	}
	return count > 0, nil
}

// InsertUser вставляет базовые данные пользователя в БД.
// При первом обращении (например, когда юзер пишет /start или любое сообщение)
// сохраняются: user_id, user_name, first_name, last_name и contact (если имеется).
// Остальные поля (UserNickname, Ava, AvaFileID, UserAge, UserSex, UserGeo)
// можно заполнить позже через UPDATE.
func (r *UserRepository) InsertUser(user *User) error {
	query := `
		INSERT INTO users (
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
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id) DO NOTHING
	`
	_, err := r.db.Exec(query,
		user.UserID,
		user.Ava,          // на начальном этапе может быть пустой строкой
		user.AvaFileID,    // на начальном этапе может быть пустой строкой
		user.UserNickname, // изначально может быть пустым, если юзер еще не задал nickname
		user.UserName,
		user.UserAge, // можно задать 0, если неизвестно
		user.UserSex, // можно задать пустую строку
		user.UserGeo, // можно задать пустую строку
		user.FirstName,
		user.LastName,
		user.Contact,
	)
	if err != nil {
		return fmt.Errorf("InsertUser: %w", err)
	}
	return nil
}

// UpdateNickname обновляет никнейм пользователя.
func (r *UserRepository) UpdateNickname(userID int64, nickname string) error {
	query := `UPDATE users SET user_nickname = $1 WHERE user_id = $2`
	_, err := r.db.Exec(query, nickname, userID)
	if err != nil {
		return fmt.Errorf("UpdateNickname: %w", err)
	}
	return nil
}

// UpdateAvatar обновляет данные аватарки пользователя: путь к файлу и FileID.
func (r *UserRepository) UpdateAvatar(userID int64, avaPath, avaFileID string) error {
	query := `UPDATE users SET ava = $1, ava_file_id = $2 WHERE user_id = $3`
	_, err := r.db.Exec(query, avaPath, avaFileID, userID)
	if err != nil {
		return fmt.Errorf("UpdateAvatar: %w", err)
	}
	return nil
}
