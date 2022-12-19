package config

// Config is the options of r0bot.
type Config struct {
	BotToken  string `yaml:"bot_token"`
	SuperUser int64  `yaml:"super_user"`
}
