package cmd

import (
	"log"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/spf13/viper"

	"github.com/fsnotify/fsnotify"
)

var appConfig struct {
	configFile     string
	sourceDatabase string
	listenAddr     string
	includeRegex   *regexp.Regexp
	excludeRegex   *regexp.Regexp
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "derpyhead",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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
	rootCmd.PersistentFlags().StringVar(&appConfig.sourceDatabase, "source-db", "", "sqlite3 database to serve information from")

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

	viper.OnConfigChange(func(e fsnotify.Event) {
		if e.Op&fsnotify.Write != 0 {
			log.Println("config reload detected", viper.ConfigFileUsed())
			readConfig()
		}
	})

	if err := viper.ReadInConfig(); err == nil {
		log.Println("using config file:", viper.ConfigFileUsed())
		readConfig()
	}

	go viper.WatchConfig()
}

func readConfig() {
	filterConfig := viper.GetStringMapString("filter")
	if val, ok := filterConfig["include"]; ok {
		pat, err := regexp.Compile(val)
		if err == nil {
			appConfig.includeRegex = pat
		} else {
			log.Println(err)
		}
	}
	if val, ok := filterConfig["exclude"]; ok {
		pat, err := regexp.Compile(val)
		if err == nil {
			appConfig.excludeRegex = pat
		} else {
			log.Println(err)
		}
	}
	if val := viper.GetString("source-db"); len(val) > 0 {
		appConfig.sourceDatabase = val
	}
}
