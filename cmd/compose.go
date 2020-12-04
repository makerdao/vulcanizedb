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
	"github.com/makerdao/vulcanizedb/pkg/config"
	"github.com/makerdao/vulcanizedb/pkg/plugin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// composeCmd represents the compose command
var composeCmd = &cobra.Command{
	Use:   "compose",
	Short: "Composes transformer initializer plugin",
	Long: `Run this command in order to write and build a go plugin with a list
of transformers specified in the config file. The plugin is loaded and the set
of transformer initializers can be executed over by the appropriate watcher.

This command needs a config file location specified:
./vulcanizedb compose --config=./environments/config_name.toml`,
	Run: func(cmd *cobra.Command, args []string) {
		SubCommand = cmd.CalledAs()
		LogWithCommand = *logrus.WithField("SubCommand", SubCommand)
		compose()
	},
}

func compose() {
	// Build plugin generator config
	genConfig, configErr := prepConfig()
	if configErr != nil {
		LogWithCommand.Fatalf("failed to prepare config: %s", configErr.Error())
	}

	composeTransformers(genConfig)

	// TODO: Embed versioning info in the .so files so we know which version of vulcanizedb to run them with
	_, pluginPath, pathErr := genConfig.GetPluginPaths()
	if pathErr != nil {
		LogWithCommand.Fatalf("getting plugin path failed: %s", pathErr.Error())
	}
	LogWithCommand.Info("plugin .so file output to ", pluginPath)
}

func init() {
	rootCmd.AddCommand(composeCmd)
}

func composeTransformers(genConfig config.Plugin) {
	// Generate code to build the plugin according to the config file
	LogWithCommand.Info("generating plugin")
	generator, constructorErr := plugin.NewGenerator(genConfig, databaseConfig)
	if constructorErr != nil {
		LogWithCommand.Fatalf("initializing plugin generator failed: %s", constructorErr.Error())
	}
	generateErr := generator.GenerateExporterPlugin()
	if generateErr != nil {
		LogWithCommand.Fatalf("generating plugin failed: %s", generateErr.Error())
	}
}
