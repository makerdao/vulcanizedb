package utilities

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/makerdao/vulcanizedb/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	MissingPathErr = errors.New("transformer config is missing `path` value")
	MissingRepositoryErr = errors.New("transformer config is missing `repository` value")
	MissingMigrationsErr = errors.New("transformer config is missing `migrations` value")
	MissingRankErr = errors.New("transformer config is missing `rank` value")
	RankParsingErr = errors.New("migration `rank` can't be converted to an unsigned integer")
	MissingTypeErr = errors.New("transformer config is missing `type` value")
	UnknownTransformerTypeErr = errors.New(`unknown transformer type in exporter config accepted types are "eth_event", "eth_storage"`)
)

func PrepConfig(subCommand string) (config.Plugin, error) {
	LogWithCommand := *logrus.WithField("SubCommand", subCommand)
	LogWithCommand.Info("configuring plugin")
	names := viper.GetStringSlice("exporter.transformerNames")
	transformers := make(map[string]config.Transformer)
	for _, name := range names {
		transformer := viper.GetStringMapString("exporter." + name)
		p, pOK := transformer["path"]
		if !pOK || p == "" {
			return config.Plugin{}, fmt.Errorf("%w: %s", MissingPathErr, name)
		}
		r, rOK := transformer["repository"]
		if !rOK || r == "" {
			return config.Plugin{}, fmt.Errorf("%w: %s", MissingRepositoryErr, name)
		}
		m, mOK := transformer["migrations"]
		if !mOK || m == "" {
			return config.Plugin{}, fmt.Errorf("%w: %s", MissingMigrationsErr, name)
		}
		mr, mrOK := transformer["rank"]
		if !mrOK || mr == "" {
			return config.Plugin{}, fmt.Errorf("%w: %s", MissingRankErr, name)
		}
		rank, err := strconv.ParseUint(mr, 10, 64)
		if err != nil {
			return config.Plugin{}, fmt.Errorf("%w: %s", RankParsingErr, name)
		}
		t, tOK := transformer["type"]
		if !tOK {
			return config.Plugin{}, fmt.Errorf("%w: %s", MissingTypeErr, name)
		}
		transformerType := config.GetTransformerType(t)
		if transformerType == config.UnknownTransformerType {
			return config.Plugin{}, UnknownTransformerTypeErr
		}

		transformers[name] = config.Transformer{
			Path:           p,
			Type:           transformerType,
			RepositoryPath: r,
			MigrationPath:  m,
			MigrationRank:  rank,
		}
	}

	return config.Plugin{
		Transformers: transformers,
		FilePath:     "$GOPATH/src/github.com/makerdao/vulcanizedb/plugins",
		Schema:       viper.GetString("exporter.schema"),
		FileName:     viper.GetString("exporter.name"),
		Save:         viper.GetBool("exporter.save"),
		Home:         viper.GetString("exporter.home"),
	}, nil
}

