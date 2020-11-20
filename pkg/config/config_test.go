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

var vulcanizeConfig = []byte(`
[database]
name = "dbname"
hostname = "localhost"
port = 5432

[client]
ipcPath = "IPCPATH/geth.ipc"
`)

var _ = Describe("Plugin Config", func() {
	It("reads the private config using the environment", func() {
		viper.SetConfigName("config")
		viper.AddConfigPath("$GOPATH/src/github.com/makerdao/vulcanizedb/environments/")
		Expect(viper.Get("client.ipcpath")).To(BeNil())

		testConfig := viper.New()
		testConfig.SetConfigType("toml")
		err := testConfig.ReadConfig(bytes.NewBuffer(vulcanizeConfig))
		Expect(err).To(BeNil())
		Expect(testConfig.Get("database.hostname")).To(Equal("localhost"))
		Expect(testConfig.Get("database.name")).To(Equal("dbname"))
		Expect(testConfig.Get("database.port")).To(Equal(int64(5432)))
		Expect(testConfig.Get("client.ipcpath")).To(Equal("IPCPATH/geth.ipc"))
	})

	Describe("PrepareConfig", func() {
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
})
