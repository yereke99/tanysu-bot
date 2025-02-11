package config

// Config содержит параметры конфигурации приложения.
type Config struct {
	Token         string `json:"token"`          // Токен для Telegram бота
	RedisAddr     string `json:"redis_addr"`     // Адрес Redis
	RedisPassword string `json:"redis_password"` // Пароль для Redis
	RedisDB       int    `json:"redis_db"`       // Номер базы данных Redis
	ChannelID     int64  `json:"channelID"`      // Идентификатор канала
	ChannelName   string `json:"channelName"`

	// Параметры для SQLite (используем DBName как путь к файлу базы)
	DBName string `json:"db_name"`
}

// NewConfig создаёт и возвращает новый экземпляр конфигурации.
func NewConfig() (*Config, error) {
	cfg := &Config{
		Token:         "1325617758:AAHD8tkdxsDOE2M5oAP9BW5LF71dg5KdRQo",
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
		RedisDB:       0,
		ChannelID:     2403228914,
		ChannelName:   "@jaiAngmeAitamyz",
		DBName:        "tanysu.db", // Имя файла базы данных SQLite
	}
	return cfg, nil
}
