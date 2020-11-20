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

package config_test

import (
	"bytes"

	"github.com/makerdao/vulcanizedb/pkg/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)
var _ = Describe("Plugin Config", func() {
	var testSubCommand = "test"
	var testConfig = []byte(`[exporter]
  home = "github.com/makerdao/vulcanizedb"
  name = "transformerExporter"
  save = true
  schema = "testSchema"
  transformerNames = ["transformer1", "transformer2"]
  [exporter.transformer1]
    contracts = ["CONTRACT1", "CONTRACT2"]
    migrations = "db/migrations"
    path = "path/to/transformer1"
    rank = "0"
    repository = "github.com/transformer-repository"
    type = "eth_event"
  [exporter.transformer2]
    migrations = "db/migrations"
    path = "path/to/transformer2"
    rank = "0"
    repository = "github.com/transformer-repository"
    type = "eth_storage"`)
	BeforeEach(func() {
		viper.SetConfigType("toml")
		readConfigErr := viper.ReadConfig(bytes.NewBuffer(testConfig))
		Expect(readConfigErr).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		viper.Reset()
	})

	Describe("PrepareConfig", func() {
		It("returns a Plugin config struct", func() {
			pluginConfig, err := config.PreparePluginConfig(testSubCommand)
			Expect(err).NotTo(HaveOccurred())
			expectedConfig := config.Plugin{
				Transformers: map[string]config.Transformer{
					"transformer1": {
						Path:           "path/to/transformer1",
						Type:           config.EthEvent,
						MigrationPath:  "db/migrations",
						MigrationRank:  0,
						RepositoryPath: "github.com/transformer-repository",
					},
					"transformer2": {
						Path:           "path/to/transformer2",
						Type:           config.EthStorage,
						MigrationPath:  "db/migrations",
						MigrationRank:  0,
						RepositoryPath: "github.com/transformer-repository",
					},
				},
				FilePath: "$GOPATH/src/github.com/makerdao/vulcanizedb/plugins",
				FileName: "transformerExporter",
				Save:     true,
				Home:     "github.com/makerdao/vulcanizedb",
				Schema:   "testSchema",
			}
			Expect(pluginConfig).To(Equal(expectedConfig))
		})

		It("returns an error if the transformer's path is missing", func() {
			viper.Set("exporter.transformer1",
				map[string]interface{}{
					"contracts":  []string{"CONTRACT1", "CONTRACT2"},
					"migrations": "db/migrations",
					"path":       "",
					"rank":       "0",
					"repository": "github.com/transformer-repository",
					"type":       "eth_event",
				},
			)
			_, err := config.PreparePluginConfig(testSubCommand)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(config.MissingPathErr))
		})

		It("returns an error if the transformer's repository is missing", func() {
			viper.Set("exporter.transformer1",
				map[string]interface{}{
					"contracts":  []string{"CONTRACT1", "CONTRACT2"},
					"migrations": "db/migrations",
					"path":       "path/to/transformer1",
					"rank":       "0",
					"repository": "",
					"type":       "eth_event",
				},
			)
			_, err := config.PreparePluginConfig(testSubCommand)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(config.MissingRepositoryErr))
		})

		It("returns an error if the transformer's migrations is missing", func() {
			viper.Set("exporter.transformer1",
				map[string]interface{}{
					"contracts":  []string{"CONTRACT1", "CONTRACT2"},
					"migrations": "",
					"path":       "path/to/transformer1",
					"rank":       "0",
					"repository": "github.com/transformer-repository",
					"type":       "eth_event",
				},
			)
			_, err := config.PreparePluginConfig(testSubCommand)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(config.MissingMigrationsErr))
		})

		It("returns an error if the transformer's rank is missing", func() {
			viper.Set("exporter.transformer1",
				map[string]interface{}{
					"contracts":  []string{"CONTRACT1", "CONTRACT2"},
					"migrations": "db/migrations",
					"path":       "path/to/transformer1",
					"rank":       "",
					"repository": "github.com/transformer-repository",
					"type":       "eth_event",
				},
			)
			_, err := config.PreparePluginConfig(testSubCommand)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(config.MissingRankErr))
		})

		It("returns an error a transformers rank cannot be parsed into an int", func() {
			viper.Set("exporter.transformer1",
				map[string]interface{}{
					"contracts":  []string{"CONTRACT1", "CONTRACT2"},
					"migrations": "db/migrations",
					"path":       "path/to/transformer1",
					"rank":       "not-an-int",
					"repository": "github.com/transformer-repository",
					"type":       "eth_event",
				},
			)
			_, err := config.PreparePluginConfig(testSubCommand)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(config.RankParsingErr))
		})

		It("returns an error if the transformer's type is missing", func() {
			viper.Set("exporter.transformer1",
				map[string]interface{}{
					"contracts":  []string{"CONTRACT1", "CONTRACT2"},
					"migrations": "db/migrations",
					"path":       "path/to/transformer1",
					"rank":       "0",
					"repository": "github.com/transformer-repository",
				},
			)
			_, err := config.PreparePluginConfig(testSubCommand)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(config.MissingTypeErr))
		})

		It("returns an error if the transformer's type is unknown", func() {
			viper.Set("exporter.transformer1",
				map[string]interface{}{
					"contracts":  []string{"CONTRACT1", "CONTRACT2"},
					"migrations": "db/migrations",
					"path":       "path/to/transformer1",
					"rank":       "0",
					"repository": "github.com/transformer-repository",
					"type":       "not-a-type",
				},
			)
			_, err := config.PreparePluginConfig(testSubCommand)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(config.UnknownTransformerTypeErr))
		})
	})

	Describe("GetPluginPaths", func() {
		It("returns the go and so file paths", func() {
			pluginConfig, prepareErr := config.PreparePluginConfig(testSubCommand)
			Expect(prepareErr).NotTo(HaveOccurred())

			goFile, soFile, getPlugingPathsErr := pluginConfig.GetPluginPaths()
			Expect(getPlugingPathsErr).NotTo(HaveOccurred())
			Expect(goFile).To(MatchRegexp("transformerExporter.go"))
			Expect(soFile).To(MatchRegexp("transformerExporter.so"))
		})
	})

	Describe("GetMigrationsPaths", func() {
		It("returns the migrations paths", func() {
			pluginConfig, prepareErr := config.PreparePluginConfig(testSubCommand)
			Expect(prepareErr).NotTo(HaveOccurred())

			migrationsPaths, getMigraionsErr := pluginConfig.GetMigrationsPaths()
			Expect(getMigraionsErr).NotTo(HaveOccurred())
			Expect(len(migrationsPaths)).To(Equal(1))
			Expect(migrationsPaths[0]).To(MatchRegexp("db/migrations"))
		})
	})

	Describe("GetRepoPaths", func() {
		It("returns the repo path", func() {
			pluginConfig, prepareErr := config.PreparePluginConfig(testSubCommand)
			Expect(prepareErr).NotTo(HaveOccurred())

			repoPaths := pluginConfig.GetRepoPaths()
			Expect(repoPaths).To(Equal(map[string]bool{"github.com/transformer-repository": true}))
		})
	})

	Describe("GetTransformerType", func() {
		It("tranlates the transformer type string to an enum", func() {
			storageTransformerType := config.GetTransformerType("eth_storage")
			Expect(storageTransformerType).To(Equal(config.EthStorage))
			eventTransformerType := config.GetTransformerType("eth_event")
			Expect(eventTransformerType).To(Equal(config.EthEvent))
			unknownTransformerType := config.GetTransformerType("unknown")
			Expect(unknownTransformerType).To(Equal(config.UnknownTransformerType))
		})
	})
})
