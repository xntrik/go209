package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type myAttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type myAttachment struct {
	Fallback string              `json:"fallback"`
	Text     string              `json:"text,omitempty"`
	Color    string              `json:"color,omitempty"`
	Footer   string              `json:"footer,omitempty"`
	Ts       int64               `json:"ts,omitempty"`
	Fields   []myAttachmentField `json:"fields,omitempty"`
}

type myWebhookBody struct {
	Text        string         `json:"text"`
	Attachments []myAttachment `json:"attachments,omitempty"`
}

type slackWebhookModule string

func (sm slackWebhookModule) Name() string {
	return "SlackWebhookModule"
}

func (sm slackWebhookModule) EnvVars() []string {
	return []string{"URL"}
}

func (sm slackWebhookModule) Run(in interface{}, ev map[string]string, interactions map[string]string) error {
	if len(ev["SLACKWEBHOOKMODULE_URL"]) == 0 {
		return errors.New("Missing SlackWebhookModule URL param")
	}
	uri := ev["SLACKWEBHOOKMODULE_URL"]

	// Construct the attachment for the message
	now := time.Now()
	secs := now.Unix()
	att1 := myAttachment{
		Fallback: "go209 received a complete response from someone",
		Color:    "#36a64f",
		Footer:   "go209",
		Ts:       secs,
	}

	// Extract the userid from the input
	switch i := in.(type) {
	case map[string]string:
		for k, v := range i {
			if k == "userid" {
				att1.Text = fmt.Sprintf("Response from <@%s>", v)
			}
		}
	}

	// Iterate over interactions, and extract results from the input
	switch i := in.(type) {
	case map[string]string:
		for interactionkey, interactionval := range interactions {
			for inkey, inval := range i {
				if strings.HasPrefix(inkey, "response:") {
					inkey = strings.TrimPrefix(inkey, "response:")
					if interactionkey == inkey {
						f := myAttachmentField{
							Title: interactionval,
							Value: inval,
							Short: false,
						}
						att1.Fields = append(att1.Fields, f)
					}
				}
			}
		}
	}

	// Raw body to send Slack webhook
	msg := &myWebhookBody{
		Text: "go209 received a complete response from someone",
	}
	msg.Attachments = append(msg.Attachments, att1)

	jsonMarshal, _ := json.Marshal(msg)
	jsonStr := []byte(string(jsonMarshal))
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return err
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*30)
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Module is exported for the plugin system
var Module slackWebhookModule
