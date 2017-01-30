package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path"
	"time"

	"github.com/prometheus/alertmanager/types"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type addResponse struct {
	Status string `json:"status"`
	Data   struct {
		SilenceID string `json:"silenceId"`
	} `json:"data,omitempty"`
	ErrorType string `json:"errorType,omitempty"`
	Error     string `json:"error,omitempty"`
}

var addFlags *flag.FlagSet
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add silence",
	Long: `Add a new alertmanager silence

  Amtool uses a simplified prometheus syntax to represent silences. The
  non-option section of arguments constructs a list of "Matcher Groups"
  that will be used to create a number of silences. The following examples
  will attempt to show this behaviour in action:

  amtool silence add alertname=foo node=bar

	This statement will add a silence that matches alerts with the
	alertname=foo and node=bar label value pairs set.

  amtool silence add foo node=bar

	If alertname is ommited and the first argument does not contain a '=' or a
	'=~' then it will be assumed to be the value of the alertname pair.

  amtool silence add 'alertname=~foo.*'

	As well as direct equality, regex matching is also supported. The '=~' syntax
	(similar to prometheus) is used to represent a regex match. Regex matching
	can be used in combination with a direct match.

  amtool silence add alertname=foo node={bar,baz}

	This statement will create multiple silences. It will create a silence that
	matches the alertname=foo label value pair and node=bar pair as well as a
	silence that matches alertname=foo and node=baz.

  amtool silence add alertname=foo{a,b} node={bar,baz}

	Similar to the previous example this statement will create 4 silences to match
	any combinartion of alertname=fooa or alertname=foob and node=bar or node=baz.
	`,
	RunE: add,
}

func init() {
	user, _ := user.Current()
	addCmd.Flags().StringP("author", "a", user.Username, "Username for CreatedBy field")
	addCmd.Flags().StringP("expires", "e", "1h", "Duration of silence (100h)")
	addCmd.Flags().String("expire-on", "", "Expire at a certain time (Overwrites expires) RFC3339 format 2006-01-02T15:04:05Z07:00")
	addCmd.Flags().StringP("comment", "c", "", "A comment to help describe the silence")
	viper.BindPFlag("author", addCmd.Flags().Lookup("author"))
	viper.BindPFlag("expires", addCmd.Flags().Lookup("expires"))
	viper.BindPFlag("comment", addCmd.Flags().Lookup("comment"))
	viper.SetDefault("comment_required", false)
	addFlags = addCmd.Flags()
}

func add(cmd *cobra.Command, args []string) error {
	var err error

	matchers, err := parseMatchers(args)
	if err != nil {
		return err
	}

	groups := parseMatcherGroups(matchers)

	if len(groups) < 1 {
		return errors.New("No matchers specified")
	}

	silences, err := fetchSilences()
	for groupIndex, group := range groups {
		for _, silence := range silences {
			// Don't allow multiple active silences with the same matchers
			if silence.EndsAt.Before(time.Now()) && silence.Matchers.Equal(group) {
				// Slice delete groupIndex
				groups = append(groups[:groupIndex], groups[groupIndex+1:]...)
				break
				if viper.GetBool("verbose") {
					fmt.Fprintf(os.Stderr, "Skipped adding duplicate silence, %s matches\n", silence.ID)
				}
			}
		}
	}

	if len(groups) < 1 {
		return errors.New("No new silences specified (Enable verbose mode for more information)")
	}

	expire_on, err := addFlags.GetString("expire-on")
	if err != nil {
		return err
	}

	expires := viper.GetString("expires")
	var endsAt time.Time

	if expire_on != "" {
		endsAt, err = time.Parse(time.RFC3339, expire_on)
		if err != nil {
			return err
		}
	} else {
		duration, err := time.ParseDuration(expires)
		if err != nil {
			return err
		}
		endsAt = time.Now().UTC().Add(duration)
	}

	author := viper.GetString("author")
	comment := viper.GetString("comment")
	comment_required := viper.GetBool("comment_required")

	if comment_required && comment == "" {
		return errors.New("Comment required by config")
	}

	for _, matchers := range groups {
		silence := types.Silence{
			Matchers:  matchers,
			StartsAt:  time.Now().UTC(),
			EndsAt:    endsAt,
			CreatedBy: author,
			Comment:   comment,
		}

		u, err := GetAlertmanagerURL()
		if err != nil {
			return err
		}
		u.Path = path.Join(u.Path, "/api/v1/silences")

		buf := bytes.NewBuffer([]byte{})
		enc := json.NewEncoder(buf)
		err = enc.Encode(silence)
		if err != nil {
			return err
		}

		res, err := http.Post(u.String(), "application/json", buf)
		if err != nil {
			return err
		}

		defer res.Body.Close()
		decoder := json.NewDecoder(res.Body)

		response := addResponse{}
		err = decoder.Decode(&response)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to parse silence json response from %s", u.String()))
		}

		if response.Status == "error" {
			fmt.Printf("[%s] %s\n", response.ErrorType, response.Error)
		} else {
			fmt.Println(response.Data.SilenceID)
		}
	}
	return nil
}
