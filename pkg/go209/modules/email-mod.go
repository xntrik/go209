package main

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/gomail.v2"
)

type emailModule string

func (tm emailModule) Name() string {
	return "EmailModule"
}

func (tm emailModule) EnvVars() []string {
	return []string{"FROM", "TO", "SMTPSERVER", "USERNAME", "PASSWORD", "SKIPTLS"}
}

func (tm emailModule) Run(in interface{}, ev map[string]string, interactions map[string]string) error {
	if len(ev["EMAILMODULE_SKIPTLS"]) > 0 {
		// Sending plaintext email
		Sender := ev["EMAILMODULE_FROM"]
		SenderName := "go209"

		Recipient := ev["EMAILMODULE_TO"]

		smtpServer := strings.Split(ev["EMAILMODULE_SMTPSERVER"], ":")
		Host := smtpServer[0]
		Port, err := strconv.Atoi(smtpServer[1])
		if err != nil {
			return err
		}

		Subject := "Email from the go209 slackbot"
		// Build email
		emailBody := "go209 slack bot received a complete response from someone.\nHere is the data\n"

		// Extract the user
		switch i := in.(type) {
		case map[string]string:
			for k, v := range i {
				if k == "username" {
					emailBody = fmt.Sprintf("%s %s: %s\n\n", emailBody, k, v)
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
							emailBody = fmt.Sprintf("%s %s: %s\n\n", emailBody, interactionval, inval)
						}
					}
				}
			}
		}

		m := gomail.NewMessage()
		m.SetHeader("From", m.FormatAddress(Sender, SenderName))
		m.SetHeader("To", Recipient)
		m.SetHeader("Subject", Subject)
		m.SetBody("text/plain", emailBody)

		d := gomail.Dialer{
			Host: Host,
			Port: Port,
		}

		err = d.DialAndSend(m)
		if err != nil {
			return err
		}

	} else {
		// Sending encrypted email
		Sender := ev["EMAILMODULE_FROM"]
		SenderName := "go209"

		Recipient := ev["EMAILMODULE_TO"]

		SMTPUser := ev["EMAILMODULE_USERNAME"]
		SMTPPass := ev["EMAILMODULE_PASSWORD"]

		smtpServer := strings.Split(ev["EMAILMODULE_SMTPSERVER"], ":")
		Host := smtpServer[0]
		Port, err := strconv.Atoi(smtpServer[1])
		if err != nil {
			return err
		}

		Subject := "Email from the go209 slackbot"
		// Build email
		emailBody := "go209 slack bot received a complete response from someone.\nHere is the data\n"

		// Extract the user
		switch i := in.(type) {
		case map[string]string:
			for k, v := range i {
				if k == "username" {
					emailBody = fmt.Sprintf("%s %s: %s\n\n", emailBody, k, v)
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
							emailBody = fmt.Sprintf("%s %s: %s\n\n", emailBody, interactionval, inval)
						}
					}
				}
			}
		}

		m := gomail.NewMessage()
		m.SetBody("text/plain", emailBody)
		m.SetHeaders(map[string][]string{
			"From":    {m.FormatAddress(Sender, SenderName)},
			"To":      {Recipient},
			"Subject": {Subject},
		})

		d := gomail.NewPlainDialer(Host, Port, SMTPUser, SMTPPass)

		err = d.DialAndSend(m)
		if err != nil {
			return err
		}
	}

	return nil
}

// Module is exported for the plugin system
var Module emailModule
