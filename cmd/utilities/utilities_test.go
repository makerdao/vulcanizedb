package utilities_test

import (
	"github.com/makerdao/vulcanizedb/cmd/utilities"
	"github.com/makerdao/vulcanizedb/pkg/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

var _ = Describe("Command Utilities", func() {
	var testSubCommand = "test"
	Describe("PrepareConfig", func() {
		BeforeEach(func() {
			viper.AddConfigPath("$GOPATH/src/github.com/makerdao/vulcanizedb/cmd/utilities/test_data")
			viper.SetConfigName("testConfig")
			readConfigErr := viper.ReadInConfig()
			Expect(readConfigErr).NotTo(HaveOccurred())
		})

		It("returns a Plugin config struct", func() {
			pluginConfig, err := utilities.PrepConfig(testSubCommand)
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
			_, err := utilities.PrepConfig(testSubCommand)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(utilities.MissingPathErr))
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
			_, err := utilities.PrepConfig(testSubCommand)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(utilities.MissingRepositoryErr))
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
			_, err := utilities.PrepConfig(testSubCommand)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(utilities.MissingMigrationsErr))
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
			_, err := utilities.PrepConfig(testSubCommand)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(utilities.MissingRankErr))
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
			_, err := utilities.PrepConfig(testSubCommand)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(utilities.RankParsingErr))
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
			_, err := utilities.PrepConfig(testSubCommand)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(utilities.MissingTypeErr))
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
			_, err := utilities.PrepConfig(testSubCommand)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(utilities.UnknownTransformerTypeErr))
		})
	})
})
