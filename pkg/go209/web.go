package go209

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/go-redis/redis"
	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
)

// myChannel - this is the overridden element, so we can get the Channel ID
type myChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// myCallbackType - We provide our own version of the slack.InteractionCallback
// this is so we can get the channel ID
type myCallbackType struct {
	Type            slack.InteractionType `json:"type"`
	Token           string                `json:"token"`
	CallbackID      string                `json:"callback_id"`
	ResponseURL     string                `json:"response_url"`
	TriggerID       string                `json:"trigger_id"`
	ActionTs        string                `json:"action_ts"`
	Team            slack.Team            `json:"team"`
	Channel         myChannel             `json:"channel"` // only change is here
	User            slack.User            `json:"user"`
	OriginalMessage slack.Message         `json:"original_message"`
	Message         slack.Message         `json:"message"`
	Name            string                `json:"name"`
	Value           string                `json:"value"`
	slack.ActionCallback
	slack.DialogSubmissionCallback
}

// slackRespond is a method for the web server to respond to a web callback
// immediately, with a replacement message
func slackRespond(w http.ResponseWriter, replace bool, message string) error {
	responseMsg := slack.Msg{
		Text:            message,
		ReplaceOriginal: replace,
		ResponseType:    "in_channel",
	}
	responseJSON, err := json.Marshal(responseMsg)
	if err != nil {
		return fmt.Errorf("Error marshalling json: %s", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
	return nil
}

// slackRespondWithAttachment is identical to slackRespond except it can handle
// a slack Attachment (button, menu drop down) as well
func slackRespondWithAttachment(w http.ResponseWriter, replace bool, message string, attachment slack.Attachment) error {
	responseMsg := slack.Msg{
		Text:            message,
		ReplaceOriginal: replace,
		ResponseType:    "in_channel",
	}
	responseMsg.Attachments = append(responseMsg.Attachments, attachment)
	responseJSON, err := json.Marshal(responseMsg)
	if err != nil {
		return fmt.Errorf("Error marshalling json: %s", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
	return nil
}

// messageHandler handles all the incoming Slack web hooks
func messageHandler(cfg *BotConfig, db *redis.Client, rules *RuleSet) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// We split the body in half because we need it for signature validation
		// then later to read it for JSON parsing
		rawBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Warn(fmt.Sprintf("Error reading Body: %s", err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Clone the body
		rdr1 := ioutil.NopCloser(bytes.NewBuffer(rawBody))
		rdr2 := ioutil.NopCloser(bytes.NewBuffer(rawBody))
		// reset r.Body to the first clone
		r.Body = rdr1
		// now set the bodyData for sig validation from the second clone
		bodyData, err := ioutil.ReadAll(rdr2)

		//Validating sig
		sv, err := slack.NewSecretsVerifier(r.Header, cfg.SlackSigningSecret)
		if err != nil {
			log.Warn(fmt.Sprintf("Error generating new secrets verifier: %s", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = sv.Write(bodyData)
		if err != nil {
			log.Warn(fmt.Sprintf("Error writing body to hmac: %s", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = sv.Ensure()
		if err != nil {
			log.Warn(fmt.Sprintf("Error validating HMAC!"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Now we parse the body for conversion into a slack struct
		r.ParseForm()

		var interactioncb myCallbackType

		err = json.Unmarshal([]byte(r.Form.Get("payload")), &interactioncb)

		if err != nil {
			log.Warn(fmt.Sprintf("Error parsing JSON from slack interaction callback: %s", err))
		}

		redKey := fmt.Sprintf("%s:%s", interactioncb.Team.ID, interactioncb.Channel.ID)
		cbID := interactioncb.CallbackID
		selected := ""
		if interactioncb.ActionCallback.Actions[0].Type == "select" {
			// The user has submitted a select menu item
			if len(interactioncb.ActionCallback.Actions[0].SelectedOptions) == 1 {
				selected = interactioncb.ActionCallback.Actions[0].SelectedOptions[0].Value
			}
		} else {
			// The user has simply clicked a button
			selected = interactioncb.ActionCallback.Actions[0].Value
		}

		val, err := db.HGetAll(redKey).Result()
		if err != nil {
			log.Warn(fmt.Sprintf("Redis error: %s", err))
		}

		if len(val) == 0 {
			// no previous state found, do nothing
			err = slackRespond(w, false, "Looks like this Interaction timed out or no longer exists")
			if err != nil {
				log.Info(fmt.Sprintf("*** MessageEvent Error trying to respond to slack message: %s", err))
			}
		} else {
			// Found a previous state, therefore we're going to carry on
			// We are in an active interaction now!
			// spew.Dump(val)
			log.Info(fmt.Sprintf("User %s (%s) has responded to interaction %s", val["username"], val["userid"], cbID))

			err = db.HSet(redKey, fmt.Sprintf("response:%s", cbID), selected).Err()
			if err != nil {
				log.Fatal(fmt.Sprintf("Error saving response into hash: %s", err))
			}

			// Handle dynamic next interaction
			// Get current rule
			nextInteraction := val["next_interaction"]

			currinteraction, err := rules.findInteractionByID(cbID)
			if err != nil {
				log.Fatal(fmt.Sprintf("Error current the current interaction: %s", err))
			}

			// dynamic determine the next step, based on the dynamic sellection
			if len(currinteraction.NextInteractionDynamic) > 0 {
				for _, dynamicNext := range currinteraction.NextInteractionDynamic {
					if dynamicNext.Response == selected {
						nextInteraction = dynamicNext.NextInteraction
					}
				}
			}

			if nextInteraction != "end" {
				// Get the next interaction
				nextinteraction, err := rules.findInteractionByID(nextInteraction)
				if err != nil {
					log.Fatal(fmt.Sprintf("Error getting the next interaction: %s", err))
				}
				err = updateState(db, redKey, nextinteraction)
				if err != nil {
					log.Fatal(fmt.Sprintf("Error updating the state: %s", err))
				}

				log.Info(fmt.Sprintf("Sending interaction %s to user %s (%s)", nextinteraction.InteractionID, val["username"], val["userid"]))

				// time to ask the next question
				switch nextinteraction.Type {
				case "text":
					err = slackRespond(w, true, fmt.Sprintf("You selected: %s\n%s", selected, nextinteraction.Question))
					if err != nil {
						log.Warn(fmt.Sprintf("Error responding to slack message: %s", err))
					}
				case "attachment":
					err = slackRespondWithAttachment(w, true, fmt.Sprintf("You selected: %s\n%s", selected, nextinteraction.Question), nextinteraction.Attachment)
					if err != nil {
						log.Warn(fmt.Sprintf("Error responding to slack message: %s", err))
					}
				}
			} else {
				// This is the last interaction
				finalval, err := db.HGetAll(redKey).Result()
				log.Info(fmt.Sprintf("User %s (%s) has completed all interactions, final step %s", val["username"], val["userid"], cbID))
				log.Info(fmt.Sprintf("Interaction RESULT:\n%v", finalval))
				err = db.Del(redKey).Err()
				if err != nil {
					log.Warn(fmt.Sprintf("Error deleting hash: %s", err))
				}
				err = slackRespond(w, true, fmt.Sprintf("You selected: %s\nThanks! We'll get back to you soon", selected))
				if err != nil {
					log.Warn(fmt.Sprintf("Error responding to slack message: %s", err))
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

									// Running the module
									err = mod.Run(finalval, evSet)
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
		}
	})
}

// StartWeb starts the web server
func StartWeb(cfg *BotConfig) error {
	db := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPwd,
		DB:       cfg.RedisDB,
	})
	_, err := db.Ping().Result()
	if err != nil {
		return fmt.Errorf("Redis error: %s", err)
	}

	rules, err := parseRuleFile(cfg.RulesFileLocation)
	if err != nil {
		return err
	}

	log.SetOutput(os.Stdout)
	if cfg.Debug {
		log.SetLevel(log.DebugLevel)
	}

	http.Handle("/slack/message_handler", messageHandler(cfg, db, rules))

	log.Info(fmt.Sprintf("Starting web server on '%s'....", cfg.WebListen))
	log.Fatal(http.ListenAndServe(cfg.WebListen, nil))

	return nil
}
