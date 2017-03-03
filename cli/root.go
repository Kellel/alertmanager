package cli

import (
	"fmt"
	"os"

	"github.com/prometheus/alertmanager/cli/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "amtool",
	Short: "Alertmanager CLI",
	Long: `View and modify the current Alertmanager state.

[Config File]

The alertmanger tool will read a config file from $HOME/.amtool.yml or /etc/amtool.yml the options are as follows

	alertmanager.url
		Set a default alertmanager url for each request

	author
		Set a default author value for new silences. If this argument is not specified then the username will be used

	comment_required
		Require a comment on silence creation

	output
		Set a default output type. Options are (simple, extended, json)
	`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().String("config", "", "config file (default is $HOME/.amtool.yml)")
	viper.BindPFlag("config", RootCmd.PersistentFlags().Lookup("config"))
	RootCmd.PersistentFlags().String("alertmanager.url", "", "Alertmanager to talk to")
	viper.BindPFlag("alertmanager.url", RootCmd.PersistentFlags().Lookup("alertmanager.url"))
	RootCmd.PersistentFlags().StringP("output", "o", "simple", "Output formatter (simple, extended, json)")
	viper.BindPFlag("output", RootCmd.PersistentFlags().Lookup("output"))
	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose running information")
	viper.BindPFlag("verbose", RootCmd.PersistentFlags().Lookup("verbose"))
	viper.SetDefault("date.format", format.DefaultDateFormat)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigName(".amtool") // name of config file (without extension)
	viper.AddConfigPath("/etc")
	viper.AddConfigPath("$HOME")
	viper.SetEnvPrefix("AMTOOL")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	cfgFile := viper.GetString("config")
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}
	err := viper.ReadInConfig()
	if err == nil {
		if viper.GetBool("verbose") {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}
