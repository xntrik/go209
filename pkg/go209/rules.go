package go209

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/davecgh/go-spew/spew"
	"github.com/nlopes/slack"
)

// RuleSet is the parent struct that defines the rules.json file
type RuleSet struct {
	Rules                        []Rule `json:"rules"`
	DefaultResponse              string `json:"default"`
	InteractionCancelledResponse string `json:"interaction_cancelled_response,omitempty"`
	InteractionCompleteResponse  string `json:"interaction_complete_response,omitempty"`
}

// Rule defines the mapping of search terms (i.e. words a user may say to the
// bot), to a simple response, OR, the initiation of a more complex
// interaction
type Rule struct {
	SearchTerms        []string         `json:"terms"`
	Response           string           `json:"response,omitempty"`
	Attachment         slack.Attachment `json:"attachment,omitempty"`
	Interactions       []Interaction    `json:"interactions,omitempty"`
	InteractionStart   string           `json:"interaction_start,omitempty"`
	InteractionEndMods []string         `json:"interaction_end_mods,omitempty"`
	SubTerms           []SubTerm        `json:"subterms,omitempty"`
}

// SubTerm defines the mapping of a sub-search term
// The idea is that if a Rule is kicked off, we keep state, and allow
// secondary or more, search terms, offering simple responses. These are
// different from Interactions in that we aren't storing state to report
// anything back.
type SubTerm struct {
	SearchTerms []string `json:"terms"`
	Response    string   `json:"response,omitempty"`
}

// Interaction defines our interactions we want to present (and handle) from
// a user. These could be a simple question (storing the next message from
// the user as a response), or a Slack Attachment (i.e. menu drop down, button)
type Interaction struct {
	InteractionID          string           `json:"interaction_id"`
	StopWord               string           `json:"stop_word"`
	Type                   string           `json:"type"`
	Question               string           `json:"question,omitempty"`
	NextInteraction        string           `json:"next_interaction"`
	Attachment             slack.Attachment `json:"attachment,omitempty"`
	NextInteractionDynamic []DynamicNext    `json:"next_interaction_dynamic,omitempty"`
}

// DynamicNext defines dynamic branching.
// This occurs after an interaction, with an attachment, is responded to by
// a user, which subsequently sends a web hook. We use these to determine the
// next interaction to present to the user
type DynamicNext struct {
	Response        string `json:"response"`
	NextInteraction string `json:"next_interaction"`
}

// findInteractionByID looks for a particular interaction within a rule
func (r *Rule) findInteractionByID(id string) (*Interaction, error) {
	for _, interaction := range r.Interactions {
		if interaction.InteractionID == id {
			return &interaction, nil
		}
	}
	return nil, fmt.Errorf("No Interaction found with ID: '%s'", id)
}

// findInteractionByID looks for a particular interaction within ALL the rules
func (r *RuleSet) findInteractionByID(id string) (*Interaction, error) {
	for _, rule := range r.Rules {
		for _, interaction := range rule.Interactions {
			if interaction.InteractionID == id {
				return &interaction, nil
			}
		}
	}
	return nil, fmt.Errorf("No Interaction found with ID: '%s'", id)
}

// findRuleByID looks for the rule which contains this particular interaction
func (r *RuleSet) findRuleByID(id string) (*Rule, error) {
	for _, rule := range r.Rules {
		for _, interaction := range rule.Interactions {
			if interaction.InteractionID == id {
				return &rule, nil
			}
		}
	}
	return nil, fmt.Errorf("No rule found containing this interaction: '%s'", id)
}

// parseRuleFile attempts to load and parse the rules.json file
func parseRuleFile(fileLoc string) (*RuleSet, error) {
	rawFile, err := ioutil.ReadFile(fileLoc)

	if err != nil {
		return nil, fmt.Errorf("Error opening json file: %s", err)
	}

	var rules RuleSet

	err = json.Unmarshal([]byte(rawFile), &rules)

	if err != nil {
		return nil, fmt.Errorf("Error decoding json: %s", err)
	}

	// checking for unique interaction IDs
	interactionids := make(map[string]bool)
	for _, rule := range rules.Rules {
		for _, interaction := range rule.Interactions {
			if _, ok := interactionids[interaction.InteractionID]; ok == true {
				return nil, fmt.Errorf("Duplicate interaction ID found: %s", interaction.InteractionID)
			}
			interactionids[interaction.InteractionID] = true
		}
	}

	// checking that the InteractionStart is set to a valid interaction
	for _, rule := range rules.Rules {
		subinteractionids := make(map[string]bool)
		for _, interaction := range rule.Interactions {
			subinteractionids[interaction.InteractionID] = true
		}
		if len(subinteractionids) > 0 {
			if _, ok := subinteractionids[rule.InteractionStart]; ok != true {
				return nil, fmt.Errorf("We couldn't find an interaction for '%s'", rule.InteractionStart)
			}
		}
	}

	// check to ensure that if there is a slack.Attachment, that the callback_id
	// matches the interaction_id
	for _, rule := range rules.Rules {
		for _, interaction := range rule.Interactions {
			if len(interaction.Attachment.Fallback) > 0 {
				if interaction.InteractionID != interaction.Attachment.CallbackID {
					return nil, fmt.Errorf("Attachment's callback_id doesn't match the interaction_id: %s", interaction.InteractionID)
				}
			}
		}
	}

	// check to ensure a rule doesn't have both Interactions AND SubTerms
	for _, rule := range rules.Rules {
		if len(rule.Interactions) > 0 && len(rule.SubTerms) > 0 {
			return nil, fmt.Errorf("A rule has both Interactions and SubTerms, it can only have one or the other: %s", rule.SearchTerms)
		}
	}

	return &rules, nil
}

// DumpRules takes the rules.json and dumps it out.
func DumpRules(cfg *BotConfig) error {
	rules, err := parseRuleFile(cfg.RulesFileLocation)

	if err != nil {
		return err
	}

	spew.Dump(rules)
	return nil
}
