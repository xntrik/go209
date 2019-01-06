package go209

// BotConfig defines the configuration that is used by both the slack bot app
// and web server
type BotConfig struct {
	SlackToken         string
	SlackSigningSecret string
	Debug              bool
	RulesFileLocation  string
	RedisAddr          string
	RedisPwd           string
	RedisDB            int
	WebListen          string
	DynamicModules     string
}
