// VulcanizeDB
// Copyright © 2019 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"fmt"
	"plugin"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/evalphobia/logrus_sentry"
	"github.com/getsentry/sentry-go"
	"github.com/makerdao/vulcanizedb/libraries/shared/factories/event"
	"github.com/makerdao/vulcanizedb/libraries/shared/factories/storage"
	"github.com/makerdao/vulcanizedb/libraries/shared/transformer"
	"github.com/makerdao/vulcanizedb/pkg/config"
	"github.com/makerdao/vulcanizedb/pkg/eth"
	"github.com/makerdao/vulcanizedb/pkg/eth/client"
	"github.com/makerdao/vulcanizedb/pkg/eth/converters"
	"github.com/makerdao/vulcanizedb/pkg/eth/node"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	LogWithCommand                       logrus.Entry
	SubCommand                           string
	blockNumberFlagName                  = "block-number"
	cfgFile                              string
	databaseConfig                       config.Database
	newDiffBlockFromHeadOfChain          int64
	unrecognizedDiffBlockFromHeadOfChain int64
	ipc                                  string
	maxUnexpectedErrors                  int
	recheckHeadersArg                    bool
	retryInterval                        time.Duration
	startingBlockNumber                  int64
)

const (
	pollingInterval      = 7 * time.Second
	validationWindowSize = 15
)

var rootCmd = &cobra.Command{
	Use:              "vulcanizedb",
	PersistentPreRun: initFuncs,
}

func Execute() {
	logrus.Info("----- Starting vDB -----")
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func initFuncs(cmd *cobra.Command, args []string) {
	setViperConfigs()
	logLvlErr := logLevel()
	if logLvlErr != nil {
		logrus.Fatalf("could not set log level: %s", logLvlErr.Error())
	}
	sentryErr := setupSentryHook()
	if sentryErr != nil {
		logrus.Fatalf("could not setup Sentry: %s", sentryErr)
	}
}

func setViperConfigs() {
	ipc = viper.GetString("client.ipcpath")
	databaseConfig = config.Database{
		Name:     viper.GetString("database.name"),
		Hostname: viper.GetString("database.hostname"),
		Port:     viper.GetInt("database.port"),
		User:     viper.GetString("database.user"),
		Password: viper.GetString("database.password"),
	}
	viper.Set("database.config", databaseConfig)
}

func logLevel() error {
	lvl, err := logrus.ParseLevel(viper.GetString("log.level"))
	if err != nil {
		return err
	}
	logrus.SetLevel(lvl)
	if lvl > logrus.InfoLevel {
		logrus.SetReportCaller(true)
	}
	logrus.Info("Log level set to ", lvl.String())
	return nil
}

func setupSentryHook() error {
	sentryEnv := viper.GetString("sentry.env")
	sentryDSN := viper.GetString("sentry.dsn")
	if sentryDSN == "" {
		logrus.Info("skipping Sentry setup because missing DSN")
		return nil
	}

	sentryErr := sentry.Init(sentry.ClientOptions{
		Dsn:         sentryDSN,
		Environment: sentryEnv,
	})
	if sentryErr != nil {
		return fmt.Errorf("error initializing Sentry: %w", sentryErr)
	}

	sentryHook, hookErr := logrus_sentry.NewSentryHook(sentryDSN, []logrus.Level{
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	})
	if hookErr != nil {
		return fmt.Errorf("error creating Sentry hook for logrus: %w", hookErr)
	}

	sentryHook.StacktraceConfiguration.Enable = true
	// it's easy to hit the default timeout of 100ms, so increase it to reduce clutter in logs
	sentryHook.Timeout = 2 * time.Second
	logrus.AddHook(sentryHook)
	return nil
}

func init() {
	// When searching for env variables, replace dots in config keys with underscores
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file location")
	rootCmd.PersistentFlags().String("database-name", "vulcanize_public", "database name")
	rootCmd.PersistentFlags().Int("database-port", 5432, "database port")
	rootCmd.PersistentFlags().String("database-hostname", "localhost", "database hostname")
	rootCmd.PersistentFlags().String("database-user", "", "database user")
	rootCmd.PersistentFlags().String("database-password", "", "database password")
	rootCmd.PersistentFlags().String("client-ipcPath", "", "rpc path to client node or geth.ipc file")
	rootCmd.PersistentFlags().String("exporter-name", "exporter", "name of exporter plugin")
	rootCmd.PersistentFlags().String("log-level", logrus.InfoLevel.String(), "Log level (trace, debug, info, warn, error, fatal, panic")
	rootCmd.PersistentFlags().String("sentry-dsn", "", "Sentry DSN")
	rootCmd.PersistentFlags().String("sentry-env", "", "Sentry environment")

	viper.BindPFlag("database.name", rootCmd.PersistentFlags().Lookup("database-name"))
	viper.BindPFlag("database.port", rootCmd.PersistentFlags().Lookup("database-port"))
	viper.BindPFlag("database.hostname", rootCmd.PersistentFlags().Lookup("database-hostname"))
	viper.BindPFlag("database.user", rootCmd.PersistentFlags().Lookup("database-user"))
	viper.BindPFlag("database.password", rootCmd.PersistentFlags().Lookup("database-password"))
	viper.BindPFlag("client.ipcPath", rootCmd.PersistentFlags().Lookup("client-ipcPath"))
	viper.BindPFlag("exporter.fileName", rootCmd.PersistentFlags().Lookup("exporter-name"))
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("sentry.dsn", rootCmd.PersistentFlags().Lookup("sentry-dsn"))
	viper.BindPFlag("sentry.env", rootCmd.PersistentFlags().Lookup("sentry-env"))

}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err == nil {
			logrus.Infof("Using config file: %s\n\n", viper.ConfigFileUsed())
		} else {
			invalidConfigError := "couldn't read config file"
			logrus.Fatalf("%s: %s", invalidConfigError, err.Error())
		}
	} else {
		logrus.Warn("No config file passed with --config flag; attempting to use env vars")
	}
}

func getBlockChain() *eth.BlockChain {
	rpcClient, ethClient := getClients()
	vdbEthClient := client.NewEthClient(ethClient)
	vdbNode := node.MakeNode(rpcClient)
	transactionConverter := converters.NewTransactionConverter(ethClient)
	return eth.NewBlockChain(vdbEthClient, rpcClient, vdbNode, transactionConverter)
}

func getClients() (client.RpcClient, *ethclient.Client) {
	rawRpcClient, err := rpc.Dial(ipc)

	if err != nil {
		LogWithCommand.Fatal(err)
	}
	rpcClient := client.NewRpcClient(rawRpcClient, ipc)
	ethClient := ethclient.NewClient(rawRpcClient)

	return rpcClient, ethClient
}

func prepConfig() (config.Plugin, error) {
	return config.PreparePluginConfig(SubCommand)
}

func exportTransformers(genConfig config.Plugin) ([]event.TransformerInitializer, []storage.TransformerInitializer, []transformer.ContractTransformerInitializer, error) {
	// Get the plugin path and load the plugin
	_, pluginPath, pathErr := genConfig.GetPluginPaths()
	if pathErr != nil {
		return nil, nil, nil, fmt.Errorf("SubCommand %v: failed to get plugin paths: %v", SubCommand, pathErr)
	}

	LogWithCommand.Info("linking plugin ", pluginPath)
	plug, openErr := plugin.Open(pluginPath)
	if openErr != nil {
		return nil, nil, nil, fmt.Errorf("SubCommand %v: linking plugin failed: %v", SubCommand, openErr)
	}

	// Load the `Exporter` symbol from the plugin
	LogWithCommand.Info("loading transformers from plugin")
	symExporter, lookupErr := plug.Lookup("Exporter")
	if lookupErr != nil {
		return nil, nil, nil, fmt.Errorf("SubCommand %v: loading Exporter symbol failed: %v", SubCommand, lookupErr)
	}

	// Assert that the symbol is of type Exporter
	exporter, ok := symExporter.(Exporter)
	if !ok {
		return nil, nil, nil, fmt.Errorf("SubCommand %v: plugged-in symbol not of type Exporter", SubCommand)
	}

	// Use the Exporters export method to load the EventTransformerInitializer, StorageTransformerInitializer, and ContractTransformerInitializer sets
	eventTransformerInitializers, storageTransformerInitializers, contractTransformerInitializers := exporter.Export()

	return eventTransformerInitializers, storageTransformerInitializers, contractTransformerInitializers, nil
}

func validateBlockNumberArg(blockNumber int64, argName string) error {
	if blockNumber == -1 {
		return fmt.Errorf("SubCommand: %v: %s argument is required and no value was given", SubCommand, argName)
	}
	return nil
}
