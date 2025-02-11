package database

import (
	"database/sql"
	"fmt"
	"log"
	"tanysu-bot/config"

	_ "github.com/mattn/go-sqlite3"
)

// DatabaseConnection устанавливает подключение к SQLite с использованием данных из конфигурации.
// В конфигурации параметр DBName используется как путь к файлу базы (например, "tanysu.db").
func DatabaseConnection(cfg *config.Config) *sql.DB {
	// Для SQLite имя базы данных — это путь к файлу базы.
	// Если файла не существует, он будет создан.
	dbPath := cfg.DBName

	// Формируем строку подключения для SQLite.
	// Опция _foreign_keys=on включает поддержку внешних ключей.
	connStr := fmt.Sprintf("%s?_foreign_keys=on", dbPath)

	// Открываем подключение к базе данных SQLite.
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		log.Fatalf("Ошибка при открытии подключения к SQLite: %v", err)
	}

	// Проверяем соединение.
	if err := db.Ping(); err != nil {
		log.Fatalf("Ошибка при подключении к SQLite: %v", err)
	}

	log.Println("Успешное подключение к SQLite!")

	// Скрипт для создания таблицы, если она не существует.
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		ava TEXT,
		ava_file_id TEXT,
		user_nickname TEXT,
		user_name TEXT,
		user_age INTEGER,
		user_sex TEXT,
		user_geo TEXT,
		first_name TEXT,
		last_name TEXT,
		contact TEXT
	);
	`

	// Выполняем запрос на создание таблицы.
	if _, err := db.Exec(createTableQuery); err != nil {
		log.Fatalf("Ошибка при создании таблицы users: %v", err)
	}
	log.Println("Таблица users успешно создана (если не существовала).")

	return db
}
