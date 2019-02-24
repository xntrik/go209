// Package go209 is the core package used to build go209 slack RTM bot (and associated web app)
package go209

import (
	"errors"
	"fmt"
	stdlog "log"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/go-redis/redis"
	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
)

// respondToDM determines whether the bot should respond to a MessageEvent
// This function will return true if the bot should respond.
func respondToDM(ev *slack.MessageEvent) bool {

	// We don't talk to bots - it could be ourselves?
	if len(ev.Msg.User) == 0 && len(ev.Msg.BotID) > 0 {
		log.Debug("*** MessageEvent We don't talk to bots")
		return false
	}

	// The message isn't from a user OR bot?
	if len(ev.Msg.User) == 0 && len(ev.Msg.BotID) == 0 {
		log.Debug("*** MessageEvent No user OR BotID?")
		return false
	}

	// Don't talk to slackbot plz
	if ev.Msg.User == "USLACKBOT" {
		log.Debug("*** MessageEvent We don't talk to slackbot")
		return false
	}

	// We only respond to DMs
	if !strings.HasPrefix(ev.Msg.Channel, "D") {
		log.Debug("*** MessageEvent We only respond to DMs")
		return false
	}

	return true
}

func finalizeInteraction(redKey, channel, username, user string, db *redis.Client, rules *RuleSet, re *regexp.Regexp, rtm *slack.RTM) {
	finalval, err := db.HGetAll(redKey).Result()
	log.Info(fmt.Sprintf("User %s (%s) has completed all interactions, final step %s", username, user, finalval["interaction"]))
	log.Info(fmt.Sprintf("Interaction RESULT:\n%v", finalval))
	err = db.Del(redKey).Err()
	if err != nil {
		log.Warn(fmt.Sprintf("Error deleting hash: %s", err))
	}

	if len(rules.InteractionCompleteResponse) > 0 {
		// We have a JSON rule to parse and respond with
		resp := preParseTemplate(rules.InteractionCompleteResponse, re)

		resp, err = parseTemplate(resp, username, user)
		if err != nil {
			log.Warn(fmt.Sprintf("Error parsing template: %s", err))
		}
		rtm.PostMessage(channel, slack.MsgOptionText(resp, false))

	} else {
		rtm.PostMessage(channel, slack.MsgOptionText("Thanks! We'll get back to you soon", false))
	}

	// now we check for any modules we need to parse for this rule
	thisRule, err := rules.findRuleByID(finalval["interaction"])
	if err != nil {
		log.Warn(fmt.Sprintf("Couldn't find rule: %s", err))
	} else {
		// we have the rule, and therefore can check for end mods
		if len(thisRule.InteractionEndMods) > 0 {
			log.Debug(fmt.Sprintf("We found %d modules to run", len(thisRule.InteractionEndMods)))

			for _, endModName := range thisRule.InteractionEndMods {
				foundMod := false
				for _, mod := range modules.Modules {
					if endModName == mod.Name() {
						foundMod = true
						log.Debug(fmt.Sprintf("We found %s module to run", mod.Name()))

						// Fetch the modules ENV VARs
						evSet := make(map[string]string)
						for _, ev := range mod.EnvVars() {
							adjusted := strings.ToUpper(fmt.Sprintf("%s_%s", mod.Name(), ev))
							evSet[adjusted] = os.Getenv(adjusted)
						}

						// Set the interactions
						interactions := make(map[string]string)
						for _, i := range thisRule.Interactions {
							interactions[i.InteractionID] = i.Question
						}

						// Running the module
						err = mod.Run(finalval, evSet, interactions)
						if err != nil {
							log.Warn(fmt.Sprintf("Error running module: %s", err))
						}
					}
				}

				if foundMod == false {
					log.Warn(fmt.Sprintf("Referenced module not found: %s", endModName))
				}
			}
		}
	}
}

// handleDM handled all the slack.MessageEvents that the bot receives
// Messages presented here have already been validated by respondToDM to ensure
// the bot only responds to what it should
func handleDM(rtm *slack.RTM, rules *RuleSet, msg, team, channel, user, username string, re *regexp.Regexp, db *redis.Client) {
	// redKey is the key used in our redis state
	redKey := fmt.Sprintf("%s:%s", team, channel)

	val, err := db.HGetAll(redKey).Result()
	if err != nil {
		// If we get to this branch, it means there was a redis error?
		log.Fatal(fmt.Sprintf("Redis error: %s", err))
	}

	// No existing state is found, this is a fresh/stateless message
	if len(val) == 0 {

		//go through the rules first
		for _, rule := range rules.Rules {
			for _, term := range rule.SearchTerms {
				// lowercase the string
				msg = strings.ToLower(msg)

				if strings.Contains(msg, term) {
					// We found an instance of a 'searchTerm' in the message

					// If there's a response in the rule, send it now.
					if len(rule.Response) > 0 {
						resp := preParseTemplate(rule.Response, re)
						resp, err := parseTemplate(resp, username, user)
						if err != nil {
							log.Warn(fmt.Sprintf("Error parsing template: %s", err))
						}

						log.Info(fmt.Sprintf("Sending standard response to search term '%s' to %s (%s)", term, username, user))
						rtm.PostMessage(channel, slack.MsgOptionText(resp, false))
					}

					// If there's an attachment in the rule, send it now
					if len(rule.Attachment.Text) > 0 {
						log.Info(fmt.Sprintf("Sending standard attachment to search term '%s' to %s (%s)", term, username, user))
						rtm.PostMessage(channel, slack.MsgOptionAttachments(rule.Attachment))
					}

					// If there's interactions in the rule, kick it off
					if len(rule.Interactions) > 0 && len(rule.InteractionStart) > 0 {
						interaction, err := rule.findInteractionByID(rule.InteractionStart)
						if err != nil {
							log.Fatal(fmt.Sprintf("Error finding starting interaction: %s", err))
						}

						err = newState(db, redKey, user, username, interaction)
						if err != nil {
							log.Fatal(fmt.Sprintf("Error saving initial state for interaction: %s", err))
						}

						log.Info(fmt.Sprintf("Initiating interaction to term '%s' to %s (%s)", term, username, user))

						// time to ask the first question
						switch interaction.Type {
						case "text":
							rtm.PostMessage(channel, slack.MsgOptionText(interaction.Question, false))
						case "attachment":
							if len(interaction.Question) > 0 {
								rtm.PostMessage(channel, slack.MsgOptionText(interaction.Question, false))
							}
							rtm.PostMessage(channel, slack.MsgOptionAttachments(interaction.Attachment))
						case "finaltext":
							rtm.PostMessage(channel, slack.MsgOptionText(interaction.Response, false))
							finalizeInteraction(redKey, channel, username, user, db, rules, re, rtm)
						}
					}

					// If there's subterms in the rule, let's set the state to handle it
					if len(rule.SubTerms) > 0 {
						err := newSubTermState(db, redKey, msg)
						if err != nil {
							log.Fatal(fmt.Sprintf("Error saving state: %s", err))
						}

						log.Info(fmt.Sprintf("Set state to handle sub search terms from '%s' to %s (%s)", term, username, user))
					}

					// if we find a matching rule, we process it and return
					// this also means that we don't handle duplicate rules.
					return
				}
			}
		}

		// if we get to here - just throw the default
		resp := preParseTemplate(rules.DefaultResponse, re)

		resp, err = parseTemplate(resp, username, user)
		if err != nil {
			log.Warn(fmt.Sprintf("Error parsing template: %s", err))
		}

		log.Info(fmt.Sprintf("Default response sent to %s (%s)", username, user))
		rtm.PostMessage(channel, slack.MsgOptionText(resp, false))

	} else {
		// Because we found a valid state in redis,  we are within an interaction now!

		if _, ok := val["searchTerm"]; ok == true {
			// This is a sub-term state

			// Let's find the rule from the stored state
			for _, rule := range rules.Rules {
				for _, term := range rule.SearchTerms {
					if strings.Contains(val["searchTerm"], term) {
						// Found the matching rule, now let's check for subterms
						if len(rule.SubTerms) > 0 {
							msg = strings.ToLower(msg)
							foundSubTerm := false

							for _, subTerm := range rule.SubTerms {
								for _, subTermSearch := range subTerm.SearchTerms {
									if strings.Contains(msg, subTermSearch) {
										// We found a hit
										foundSubTerm = true
										// If there's a response in the rule, send it now.
										if len(subTerm.Response) > 0 {
											resp := preParseTemplate(subTerm.Response, re)
											resp, err := parseTemplate(resp, username, user)
											if err != nil {
												log.Warn(fmt.Sprintf("Error parsing template: %s", err))
											}

											log.Info(fmt.Sprintf("Sending sub-term response to search term '%s'/'%s' to %s (%s)", val["searchTerm"], subTermSearch, username, user))
											rtm.PostMessage(channel, slack.MsgOptionText(resp, false))
										}
									}
								}
							}

							if foundSubTerm == false {
								// no sub-term found, send a default response
								log.Info(fmt.Sprintf("No sub-term found to search term '%s'/'%s' to %s (%s)", val["searchTerm"], msg, username, user))
								rtm.PostMessage(channel, slack.MsgOptionText("Sorry, couldn't help you", false))
							}
						}
					}
				}
			}
			// We always delete the state now
			err := db.Del(redKey).Err()
			if err != nil {
				log.Warn(fmt.Sprintf("Error deleting hash: %s", err))
			}
		} else {
			// We are assuming we're now in an interaction state

			// If the message is the stop-word, kill the session and send the interaction
			// cancelled message
			if msg == val["stop_word"] {
				err = db.Del(redKey).Err()
				if err != nil {
					log.Warn(fmt.Sprintf("Error deleting hash: %s", err))
				}
				log.Info(fmt.Sprintf("User %s (%s) has cancelled interaction %s", username, user, val["interaction"]))
				if len(rules.InteractionCancelledResponse) > 0 {
					// We have a JSON rule to parse and respond with
					resp := preParseTemplate(rules.InteractionCancelledResponse, re)

					resp, err = parseTemplate(resp, username, user)
					if err != nil {
						log.Warn(fmt.Sprintf("Error parsing template: %s", err))
					}
					rtm.PostMessage(channel, slack.MsgOptionText(resp, false))

				} else {
					rtm.PostMessage(channel, slack.MsgOptionText("Interaction cancelled", false))
				}
			} else {
				// The message wasn't the stop-word, we're going to save the response into redis
				err = db.HSet(redKey, fmt.Sprintf("response:%s", val["interaction"]), msg).Err()
				if err != nil {
					log.Fatal(fmt.Sprintf("Error saving response into hash: %s", err))
				}

				log.Info(fmt.Sprintf("User %s (%s) has responded to an interaction %s", username, user, val["interaction"]))

				// Determine if this was the last interaction in the rule
				if val["next_interaction"] != "end" {
					// This was not the last interaction (because the next isn't 'end')
					// Because there is another, we have to load it up, and then update the state
					// and then send the interaction to the user

					nextinteraction, err := rules.findInteractionByID(val["next_interaction"])
					if err != nil {
						log.Fatal(fmt.Sprintf("Error getting the next interaction: %s", err))
					}
					// We should probably throw a hard error above
					err = updateState(db, redKey, nextinteraction)
					if err != nil {
						log.Fatal(fmt.Sprintf("Error updating the state: %s", err))
					}

					// time to ask the next question
					switch nextinteraction.Type {
					case "text":
						rtm.PostMessage(channel, slack.MsgOptionText(nextinteraction.Question, false))
					case "attachment":
						if len(nextinteraction.Question) > 0 {
							rtm.PostMessage(channel, slack.MsgOptionText(nextinteraction.Question, false))
						}
						rtm.PostMessage(channel, slack.MsgOptionAttachments(nextinteraction.Attachment))
					case "finaltext":
						rtm.PostMessage(channel, slack.MsgOptionText(nextinteraction.Response, false))
						finalizeInteraction(redKey, channel, username, user, db, rules, re, rtm)
					}
				} else {
					// This is now after receiving text after the *final* interaction
					// We will store the result, then clear the state and handle the response
					finalizeInteraction(redKey, channel, username, user, db, rules, re, rtm)
				}
			}
		}
	}
}

// StartBot starts the slack bot
func StartBot(cfg *BotConfig) error {
	botUsername := ""
	botID := ""

	// load the rules json file
	rules, err := parseRuleFile(cfg.RulesFileLocation)
	if err != nil {
		return err
	}

	// compile the regular expression
	re := regexp.MustCompile(TemplatePreParserRegex)

	// setup redis
	db := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPwd,
		DB:       cfg.RedisDB,
	})
	_, err = db.Ping().Result()
	if err != nil {
		return fmt.Errorf("Redis error: %s", err)
	}

	// configure the logger
	log.SetOutput(os.Stdout)
	if cfg.Debug {
		log.SetLevel(log.DebugLevel)
	}

	// connect to the slack API
	api := slack.New(cfg.SlackToken,
		slack.OptionDebug(cfg.Debug),
		slack.OptionLog(stdlog.New(os.Stdout, "Debug-slackAPI: ", stdlog.Lshortfile|stdlog.LstdFlags)),
	)

	// turn on the batch_presence_aware option
	rtm := api.NewRTM(slack.RTMOptionConnParams(url.Values{
		"batch_presence_aware": {"1"},
	}))

	// start a new goroutine with the slack RTM API
	go rtm.ManageConnection()

	// handle incoming RTM messages
	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {

		case *slack.MessageEvent:
			if respondToDM(ev) {
				u, err := rtm.GetUserInfo(ev.Msg.User)
				if err != nil {
					log.Error(fmt.Sprintf("*** MessageEvent - GetUserInfo error: %s", err))
				} else {
					handleDM(rtm, rules, ev.Msg.Text, ev.Msg.Team, ev.Msg.Channel, ev.Msg.User, u.RealName, re, db)
				}
			}

		case *slack.HelloEvent:
			log.Debug("*** HelloEvent: Hello! We have connected")

		case *slack.MemberJoinedChannelEvent:
			log.Info("*** MemberJoinedChannelEvent")
			log.Info("*** This slack bot probably shouldn't be in a channel")
			rtm.PostMessage(ev.Channel, slack.MsgOptionText("I don't really like being in channels, so feel free to kick me out", false))
			// bots API access can't LeaveChannel - it's a slack limitation :/
			// b, err := rtm.LeaveChannel(ev.Channel)

		case *slack.ConnectedEvent:
			log.Debug(fmt.Sprintf("*** ConnectedEvent: Infos: %v", ev.Info))
			botUsername = ev.Info.User.Name
			botID = ev.Info.User.ID
			log.Info(fmt.Sprintf("*** Connected to slack. I am '%s' and my userid is %s", botUsername, botID))
			log.Debug(fmt.Sprintf("*** ConnectedEvent: Connection counter: %d", ev.ConnectionCount))

		case *slack.PresenceChangeEvent:
			log.Info(fmt.Sprintf("*** PresenceChangeEvent: %v", ev))

		case *slack.LatencyReport:
			log.Debug(fmt.Sprintf("*** LatencyReport: Current latency: %v", ev.Value))

		case *slack.RTMError:
			log.Warn(fmt.Sprintf("*** RTMError: %s", ev.Error()))

		case *slack.InvalidAuthEvent:
			log.Warn(fmt.Sprintf("*** InvalidAuthEvent"))
			return errors.New("*** InvalidAuthEvent")

		default:
			// fmt.Printf("*** defaultEvent [%v]\n", msg.Data)
		}
	}
	return nil
}
