package webhooks

type Config struct {
	Secret       string `koanf:"secret"`
	BotUserEmail string `koanf:"bot_user_email"`
}
