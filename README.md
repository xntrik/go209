# go209

[![Travis CI](https://img.shields.io/travis/xntrik/go209.svg?style=for-the-badge)](https://travis-ci.org/xntrik/go209)
[![GoDoc](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=for-the-badge)](https://godoc.org/github.com/xntrik/go209/pkg/go209)

Hi, I'm a slack bot written in Go. My name is inspired from one of the dumbest-smart robots out there, good ol' [ED-209](https://www.youtube.com/watch?v=A9l9wxGFl4k). 

Instead of being triggered with a `/slack` command, you interact with go209 through DMs. By definining a relatively simple JSON files, you can make go209 respond to various terms. Not only that, but you can also define linear or branching Q/A interactions with go209 too! Responses to these interactions can then be processed by arbitrary modules, for instance, emailing you the results.

![go209 in action](https://media.giphy.com/media/MX5HTUIiEL4jaYZhdg/giphy.gif)

![go209 in action](https://media.giphy.com/media/1gTnQtkPgT6uvfuU0Q/giphy.gif)

## Installation

### Dependencies

- `redis` - this is required to track state between conversations, and also to keep the separate slack app and web hook server synchronized. (There's also a docker-compose setup too, which containerizes the whole thing if that's easier)

### Via Go

```console
$ go get github.com/xntrik/go209
$ cd <into go209 folder - often ~/go/src/github.com/xntrik/go209>
$ make buildplugins
```

## Usage

```console
$ go209 -h
NAME:
   go209 - The dumbest-smart slack bot app (in go)

USAGE:
   go209 [global options] command [command options] [arguments...]

COMMANDS:
     start, s  Start the slack bot.
     modules   Display the loaded modules
     dump      Dump the rules json file, makes sure it parses too
     web, w    Start the web app.
     help, h   Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug, -d    enable debug output
   --help, -h     show help
   --version, -v  print the version

ENV VARIABLES:
  SLACK_TOKEN          Slack Bot User OAuth Access Token (required)
  SLACK_SIGNING_SECRET Slack Bot Signing Secret (required)
  REDIS_ADDR           REDIS address (required)
  REDIS_PWD            REDIS password (default: "")
  REDIS_DB             REDIS DB (default: 0)
  JSON_RULES           The rule file (default: "rules.json")
  WEB_ADDR             The web listener address (default: "localhost:8000")
  DYNAMIC_MODULES      Optional .so plugins you want to load (separate with ":")

  EmailModule Module ENV VARIABLES:
  EMAILMODULE_FROM
  EMAILMODULE_TO
  EMAILMODULE_SMTPSERVER
  EMAILMODULE_USERNAME
  EMAILMODULE_PASSWORD
  EMAILMODULE_SKIPTLS
```

To run properly you need to be running:
- A redis instance
- `./go209 start` for the interactive slack app
- `./go209 web` to handle web hooks from slack

To simplify this:

```console
$ make docker-compose-up
```

This will start a redis container, and the two go209 containers. You can read more about the docker-compose setup below. You can read more about the docker-compose setup below

### Variables

go209 requires a few different ENV VARs setup to run, but don't worry, you can just plonk them in your `.env` file.

- `SLACK_TOKEN` **This is the Slack Bot User OAuth Access Token (required)** See below under Slack Setup
- `SLACK_SIGNING_TOKEN` **This is the Slack Bot Signing Secret (required)** See below under Slack Setup
- `REDIS_ADDR` **Points to your redis instance. (required)** If using docker-compose, set this to `redis:6379`
- `REDIS_PWD` **If your redis requires authentication**
- `REDIS_DB` **If you want to use a redis DB other than 0**
- `JSON_RULES` **go209 comes with a sample rules.json, if you want to point to the location of a different file, set it here**
- `WEB_ADDR` **This sets the go209 web server listening interface**
- `DYNAMIC_MODULES` **If you want to load further modules, after you've compiled them, set their names here** See below under Modules

Any modules that require env vars will also be displayed, for instance, if you want to send emails.

#### Slack Setup

1. You need to setup your go209 app bot in slack by visiting https://api.slack.com/ and then press the `Start Building` button.
2. Give your bot a name
3. Select your slack workspace
4. Click `Create App`
5. Under `Add features and functionality` click `Bots`, then click `Add a Bot User`, give it a name, and finalize by clicking `Add Bot User`
6. To generate the `SLACK_TOKEN` head to `Install App` on the left, then `Install App to Workspace`
7. From here, you'll see the `Bot User OAuth Access Token`, set this to your `SLACK_TOKEN`
8. The `SLACK_SIGNING_SECRET` is available under the `Signing Secret` portion under `App Credentials` on the `Basic Information` page.

![Create a Slack App](https://i.imgur.com/VSZcW6f.png)

![Bot User](https://i.imgur.com/yO0Rjq5.png)

![OAuth Tokens](https://i.imgur.com/O6retDp.png)

To handle interactive attachments, such as menu drop downs or button selection, you need to ensure that the web hook callback is configured to hit your instance of go209. You must have TLS configured, or else slack won't connect properly. go209 doesn't terminate TLS, so I would recommend spinning up a simple nginx in front of the web portion. (or ELB/ALB).

1. Visit `Interactive Components` in the slack app's API page
2. Click `Interactivity` to on
3. Enter the URL into the `Request URL` that can hit your running instance of `go209 web`, for instance `https://yourdomain.com/slack/message_handler`

![Interactive Components](https://i.imgur.com/cgmVfvr.png)

### Rules JSON

#### Simple responses

The rules for go209 are defined in, by default, rules.json. In their simplest form, a rule could simply respond to a message:

```
{
  "terms": ["hi"],
  "response": "Hello"
}
```

If you want a single response to apply to multiple terms:

```
{
  "terms": ["hi", "hello"],
  "response": "Ohai"
}
```

You can also spice up your responses by using limited randomness and references.

```
{
  "terms": ["hi", "hello"],
  "response" "[[Hi||Hey]] {{.Username}}, How's [[things||stuff||life]]?"
}
```

#### Text base question / answers

If you want go209 to ask questions, and store the results:

```
{
  "terms": ["questionnaire"],
  "response": "Hey {{.Username}}, I'm going to ask you some questions. If you want to finish early, just send me the word 'stop'.",
  "interactions": [
    {
      "interaction_id": "q1",
      "stop_word": "stop",
      "type": "text",
      "question": "How was your day yesterday?",
      "next_interaction": "q2"
    },
    {
      "interaction_id": "q2",
      "stop_word": "stop",
      "type": "text",
      "question": "Do you think you'll have a good day tomorrow?",
      "next_interaction": "end"
    }
  ],
  "interaction_start": "q1"
}
```

There's a bit to unravel. If in your rule, you add an `interactions` array, you can define multiple interactions. All interactions in your rules file must have a unique `interaction_id`. You also need to specify the `interaction_start` in your rule, this specifies the first interaction to kick off when the user sends the message `questionnaire`.

You have to ensure at least one interaction has the `next_interaction` set to `end`, otherwise it'll never finish, and that would be dreadful.

#### Kicking off a Dynamic Module at the end of a set of interactions

Responses to these will just be echoed at the terminal, which isn't that useful. This is where modules can come into play. You can read more about modules below, but a default module includes the `EmailModule`. If you start go209 with correct `EMAILMODULE` ENV VARs, you can then adjust your rules to run a dynamic module at the end of the question/answers.

```
{
  "terms": ["questionnaire"],
  "response": "A quick questionnaire",
  "interactions": [
    {
      "interaction_id": "qq1",
      "stop_word": "stop",
      "type": "text",
      "question": "Do you like golang?"
      "next_interaction": "end"
    }
  ],
  "interaction_start": "qq1",
  "interaction_end_mods": ["EmailModule"]
}
```

#### Buttons and other slack attachments

Sure, text-based q&a is fun, but what if you want to present and handle buttons or menus.

Attachments follow the exact same specification from [nlopes/slack](https://github.com/nlopes/slack/blob/master/attachments.go#L64)

```
{
  "terms": ["questionnaire"],
  "response": "A quick q",
  "interactions": [
    {
      "interaction_id": "a1",
      "stop_word": "stop",
      "type": "attachment",
      "question": "Do you like pineapple on pizza?",
      "attachment": {
        "fallback": "Do you like pineapple on pizza?",
        "callback_id": "a1",
        "actions": [
          {
            "name": "pineapple",
            "text": "Yes"
            "style": "danger",
            "type": "button",
            "value": "yes",
            "confirm: {
              "title": "Are you sure?",
              "text": "You actually like pineapple on pizza?",
              "ok_text": "Yes",
              "dismiss_text": "No"
            }
          },
          {
            "name": "pineapple",
            "text": "No",
            "type": "button",
            "value": "no"
          }
        ]
      },
      "next_interaction": "end"
    }
  ],
  "interaction_start": "a1"
}
```

#### Branching interactions

You can also branch to different interactions depending on the responses to buttons.

```
{
  "terms": ["questionnaire"],
  "response": "A quick q",
  "interactions": [
    {
      "interaction_id": "a1",
      "stop_word": "stop",
      "type": "attachment",
      "question": "Do you like pineapple on pizza?",
      "attachment": {
        "fallback": "Do you like pineapple on pizza?",
        "callback_id": "a1",
        "actions": [
          {
            "name": "pineapple",
            "text": "Yes",
            "style": "danger",
            "type": "button",
            "value": "yes",
            "confirm": {
              "title": "Are you sure?",
              "text": "You actually like pineapple on pizza?",
              "ok_text": "Yes",
              "dismiss_text": "No"
            }
          },
          {
            "name": "pineapple",
            "text": "No",
            "type": "button",
            "value": "no"
          }
        ]
      },
      "next_interaction_dynamic": [
        {
          "response": "no",
          "next_interaction": "end"
        },
        {
          "response": "yes",
          "next_interaction": "a2"
        }
      ],
      "next_interaction": "end"
    },
    {
      "interaction_id": "a2",
      "stop_word": "stop",
      "type": "text",
      "question": "Why do you like pineapple on pizza?",
      "next_interaction": "end"
    }
  ],
  "interaction_start": "a1"
}
```

You can see we've defined the `next_interaction_dynamic` array, which will branch off to a different interaction depending on the response. You'll still want a fallback `next_interaction`, just in case.

#### Default responses

In the root of the rules file you can also specify:

- `default` - This is what go209 will say if it doesn't understand something.
- `interaction_cancelled_response` - If a user uses the stop word mid-interaction.
- `interaction_complete_response` - Once a user completes a set of interactions.

#### go209 Modules

Currently go209 supports output modules. These are executed, if configured in your rules.json, to perform arbitrary actions at the completion of a interaction/Q&A. go209 comes with an email module (loaded), and a test module (not loaded).

To write your own, great a .go file in the `pkg/go209/modules/` folder:

```
package main

import (
	"fmt"
)

type testModule string

func (tm testModule) Name() string {
	return "TestModule"
}

func (tm testModule) EnvVars() []string {
	return []string{"One", "Two"}
}

func (tm testModule) Run(in interface{}, ev map[string]string) error {
	fmt.Println("******* MODULE RUNNING!")

	return nil
}

// Module is exported to be picked up by the plugin system
var Module testModule
```

Then build it:

```console
$ make buildplugins
```

That should generate .so in the root go209 folder.

To ensure that these are loaded, with the `DYNAMIC_MODULES` ENV VAR. For instance, say you compiled my-mod.go into my-mod.so:

```console
$ DYNAMIC_MODULES=my-mod ./go209 modules
```

Will print out the dynamic modules. If you want to add more modules:

```console
$ DYNAMIC_MODULES=my-mod:my-other-mod ./go209
```

## Building etc

```console
$ make
all                   Clean, fmt, lint, vet and build!
build                 Builds the binary and plugins
static                Build a static executable - don't forget to build the plugins statically as well
buildplugins          Build .so files from contents of pkg/go209/modules/*.go
fmt                   Verifies all files have been `gofmt`ed.
lint                  Verifies `golint` passes.
vet                   Verifies `go vet` passes.
image                 Create docker image from the Dockerfile
docker-compose-build  Build the docker compose
docker-compose-up     Start the docker compose
docker-compose-upd    Start the docker compose in background mode
clean                 Cleanup any build binaries or packages
```

### Docker-compose

```console
$ make docker-compose-build
$ make docker-compose-up
```

### Example nginx server config

```
server {
    listen       443 ssl http2 default_server;
    listen       [::]:443 ssl http2 default_server;
    server_name  _;
    root         /usr/share/nginx/html;

    ssl_certificate "/etc/letsencrypt/live/YOURDOMAIN/fullchain.pem";
    ssl_certificate_key "/etc/letsencrypt/live/YOURDOMAIN/privkey.pem";
    # It is *strongly* recommended to generate unique DH parameters
    # Generate them with: openssl dhparam -out /etc/pki/nginx/dhparams.pem 2048
    ssl_dhparam "/etc/pki/nginx/dhparams.pem";
    ssl_session_cache shared:SSL:1m;
    ssl_session_timeout  10m;
    ssl_protocols TLSv1 TLSv1.1 TLSv1.2;
    ssl_ciphers HIGH:SEED:!aNULL:!eNULL:!EXPORT:!DES:!RC4:!MD5:!PSK:!RSAPSK:!aDH:!aECDH:!EDH-DSS-DES-CBC3-SHA:!KRB5-DES-CBC3-SHA:!SRP;
    ssl_prefer_server_ciphers on;

    # Load configuration files for the default server block.
    include /etc/nginx/default.d/*.conf;

    location / {
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_pass http://localhost:8000;
    }

    error_page 404 /404.html;
        location = /40x.html {
    }

    error_page 500 502 503 504 /50x.html;
        location = /50x.html {
    }
}
```
