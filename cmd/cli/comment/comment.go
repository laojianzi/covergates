package comment

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/urfave/cli/v2"

	"github.com/covergates/covergates/cmd/cli/modules"
)

// Command for comment on pull request
var Command = &cli.Command{
	Name:  "comment",
	Usage: "leave a report summary comment",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "report",
			Usage:    "report id",
			EnvVars:  []string{"REPORT_ID"},
			Value:    "",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "number",
			Usage:    "pull request number",
			EnvVars:  []string{"DRONE_PULL_REQUEST", "PULL_REQUEST"},
			Value:    "",
			Required: true,
		},
	},
	Action: comment,
}

func comment(c *cli.Context) error {
	url := fmt.Sprintf(
		"%s/reports/%s/comment/%s",
		c.String("url"),
		c.String("report"),
		c.String("number"),
	)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	client := modules.GetHTTPClient(c)
	respond, err := client.Do(req)
	if err != nil {
		return err
	}

	var text []byte
	defer func() {
		_ = respond.Body.Close()
		if respond.StatusCode >= 400 {
			log.Fatal(string(text))
		} else {
			log.Println(string(text))
		}
	}()

	text, err = ioutil.ReadAll(respond.Body)
	if err != nil {
		return err
	}

	return nil
}
