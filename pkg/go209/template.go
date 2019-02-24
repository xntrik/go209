package go209

import (
	"bytes"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// TemplatePreParserRegex is the regular expression to parse our template pre-parser
const TemplatePreParserRegex = `\[\[[\w+\|\|]+\w+\]\]`

// SlackUser is only used for template parsing
type SlackUser struct {
	Username string
	UserID   string
}

// preParseTemplate parses strings looking for:
// [[word||word||word]] and will randomly select one of the words
// This function will do this for all instances of [[ ]] in a template
// This happens before the template parsing performed by parseTemplate
func preParseTemplate(templatetext string, re *regexp.Regexp) string {
	result := templatetext
	match := re.FindAllStringSubmatchIndex(result, -1)
	for i := len(match) - 1; i >= 0; i-- {
		extract := result[match[i][0]+2 : match[i][1]-2]
		splitExtract := strings.Split(extract, "||")
		if len(splitExtract) > 1 {
			var b strings.Builder
			b.Grow(len(templatetext))

			s1 := rand.NewSource(time.Now().UnixNano())
			r1 := rand.New(s1)
			rando := r1.Intn(len(splitExtract))
			randoword := splitExtract[rando]

			fmt.Fprintf(&b, result[:match[i][0]])
			fmt.Fprintf(&b, randoword)
			fmt.Fprintf(&b, result[match[i][1]:])

			result = b.String()
		}
	}
	return result
}

// parseTemplate will use text/template to parse the provided string
// The only attributes we're running through the template are the slack user's:
// * username
// * userid
//
// Therefore the only template items you should include in your rules are:
// {{.Username}} or {{.UserID}}
func parseTemplate(templatetext, username, userid string) (string, error) {
	u := SlackUser{username, userid}

	templ := template.New("dmtemplate")
	templ, err := templ.Parse(templatetext)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	err = templ.Execute(buf, u)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
