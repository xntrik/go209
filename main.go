package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xntrik/go209/pkg/go209"

	"github.com/urfave/cli"
)

func main() {
	loadDotEnv()

	app := NewApp()
	err := app.Run(os.Args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

// NewApp is the cli.App which bootstraps everything
func NewApp() *cli.App {
	app := cli.NewApp()
	app.Name = "go209"
	app.Usage = "The dumbest-smart slack bot app (in go)"
	app.Version = go209.Version

	cli.AppHelpTemplate = fmt.Sprintf(`%s
ENV VARIABLES:
	SLACK_TOKEN          Slack Bot User OAuth Access Token (required)
	SLACK_SIGNING_SECRET Slack Bot Signing Secret (required)
	REDIS_ADDR           REDIS address (required)
	REDIS_PWD            REDIS password (default: "")
	REDIS_DB             REDIS DB (default: 0)
	JSON_RULES           The rule file (default: "rules.json")
	WEB_ADDR             The web listener address (default: "localhost:8000")
	DYNAMIC_MODULES      Optional .so plugins you want to load (separate with ":") `, cli.AppHelpTemplate)

	// Check for additional app help for module ENV VARS
	tmpMods := go209.FetchMods()

	modEnvVarHelp := ""

	for _, mod := range tmpMods.Modules {
		if len(mod.EnvVars()) > 0 {
			// this module has env vars
			modEnvVarHelp = fmt.Sprintf(`%s

%s Module ENV VARIABLES:`, modEnvVarHelp, mod.Name())

			for _, ev := range mod.EnvVars() {
				adjusted := strings.ToUpper(fmt.Sprintf("%s_%s", mod.Name(), ev))
				modEnvVarHelp = fmt.Sprintf(`%s
  %s`, modEnvVarHelp, adjusted)
			}
		}
	}

	// If there are module env vars, lets add them to the help dialog
	if len(modEnvVarHelp) > 0 {
		cli.AppHelpTemplate = fmt.Sprintf(`%s %s

	`, cli.AppHelpTemplate, modEnvVarHelp)
	}

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "enable debug output",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "start",
			Aliases: []string{"s"},
			Usage:   "Start the slack bot.",
			Action: func(c *cli.Context) error {
				// Fetch required env vars
				slackToken, err := getSlackToken()
				if err != nil {
					return err
				}

				slackSigningSecret, err := getSlackSigningSecret()
				if err != nil {
					return err
				}

				redisAddr, err := getRedisAddr()
				if err != nil {
					return err
				}

				cfg := go209.BotConfig{
					SlackToken:         slackToken,
					SlackSigningSecret: slackSigningSecret,
					Debug:              c.GlobalBool("debug"),
					RulesFileLocation:  getRulesFileLocation(),
					RedisAddr:          redisAddr,
					RedisPwd:           getRedisPwd(),
					RedisDB:            getRedisDB(),
				}

				err = go209.StartBot(&cfg)
				return err
			},
		},
		{
			Name:  "modules",
			Usage: "Display the loaded modules",
			Action: func(c *cli.Context) error {
				err := go209.DumpMods()
				return err
			},
		},
		{
			Name:  "dump",
			Usage: "Dump the rules json file, makes sure it parses too",
			Action: func(c *cli.Context) error {
				cfg := go209.BotConfig{
					RulesFileLocation: getRulesFileLocation(),
				}

				err := go209.DumpRules(&cfg)
				return err
			},
		},
		{
			Name:    "web",
			Aliases: []string{"w"},
			Usage:   "Start the web app.",
			Action: func(c *cli.Context) error {
				// Fetch required env vars
				slackSigningSecret, err := getSlackSigningSecret()
				if err != nil {
					return err
				}

				redisAddr, err := getRedisAddr()
				if err != nil {
					return err
				}

				cfg := go209.BotConfig{
					SlackSigningSecret: slackSigningSecret,
					Debug:              c.GlobalBool("debug"),
					RulesFileLocation:  getRulesFileLocation(),
					RedisAddr:          redisAddr,
					RedisPwd:           getRedisPwd(),
					RedisDB:            getRedisDB(),
					WebListen:          getWebListen(),
				}

				err = go209.StartWeb(&cfg)
				return err
			},
		},
	}

	return app
}
