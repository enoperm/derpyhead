package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/spf13/viper"

	"github.com/google/shlex"
)

var keysCommandWhole string

var appConfig struct {
	updateInterval  string
	configFile      string
	listenAddr      string
	keysCommand     string
	keysCommandArgs []string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "derpyhead",
	Short: "tiny nodekey provider for derper",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.AddCommand(serveCmd)
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&appConfig.configFile, "config", "", "config file (default is $PWD/derpyhead.yaml)")
	rootCmd.PersistentFlags().StringVar(&appConfig.listenAddr, "listen-path", "derpyhead.sock", "path of unix socket to serve peer IDs on")
	rootCmd.PersistentFlags().StringVar(&keysCommandWhole, "keys-command", "", "command to run when querying keys (must return one nodekey per line)")
	rootCmd.PersistentFlags().StringVar(&appConfig.updateInterval, "update-interval", "10s", "interval between executions of keys-command")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if appConfig.configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(appConfig.configFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("derpyhead")
	}

	viper.AutomaticEnv() // read in environment variables that match

	if err := viper.ReadInConfig(); err == nil {
		log.Println("using config file:", viper.ConfigFileUsed())
		readConfig()
	}

	if len(keysCommandWhole) > 0 {
		setKeysCommand(keysCommandWhole)
	}

	if len(appConfig.keysCommand) < 1 {
		log.Fatal("must specify a command to fetch node keys with")
	}
}

func readConfig() {
	if val := viper.GetString("update-interval"); len(val) > 0 {
		appConfig.updateInterval = val
	}

	if val := viper.GetString("keys-command"); len(val) > 0 {
		setKeysCommand(val)
	}
}

func setKeysCommand(whole string) {
	argv, err := shlex.Split(whole)
	if err != nil {
		log.Fatal(err)
	}

	appConfig.keysCommand = argv[0]
	if len(argv) > 1 {
		appConfig.keysCommandArgs = argv[1:]
	} else {
		appConfig.keysCommandArgs = appConfig.keysCommandArgs[:0]
	}
}
