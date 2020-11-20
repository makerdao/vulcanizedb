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

package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/makerdao/vulcanizedb/pkg/plugin/helpers"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Plugin struct {
	Transformers map[string]Transformer
	FilePath     string
	FileName     string
	Save         bool
	Home         string
	Schema       string
}

type Transformer struct {
	Path           string
	Type           TransformerType
	MigrationPath  string
	MigrationRank  uint64
	RepositoryPath string
}

var (
	PluginFilePath            = "$GOPATH/src/github.com/makerdao/vulcanizedb/plugins"
	MissingPathErr            = errors.New("transformer config is missing `path` value")
	MissingRepositoryErr      = errors.New("transformer config is missing `repository` value")
	MissingMigrationsErr      = errors.New("transformer config is missing `migrations` value")
	MissingRankErr            = errors.New("transformer config is missing `rank` value")
	RankParsingErr            = errors.New("migration `rank` can't be converted to an unsigned integer")
	MissingTypeErr            = errors.New("transformer config is missing `type` value")
	UnknownTransformerTypeErr = errors.New(`unknown transformer type in exporter config accepted types are "eth_event", "eth_storage"`)
)

func PreparePluginConfig(subCommand string) (Plugin, error) {
	LogWithCommand := *logrus.WithField("SubCommand", subCommand)
	LogWithCommand.Info("configuring plugin")
	names := viper.GetStringSlice("exporter.transformerNames")
	transformers := make(map[string]Transformer)
	for _, name := range names {
		transformer := viper.GetStringMapString("exporter." + name)
		p, pOK := transformer["path"]
		if !pOK || p == "" {
			return Plugin{}, fmt.Errorf("%w: %s", MissingPathErr, name)
		}
		r, rOK := transformer["repository"]
		if !rOK || r == "" {
			return Plugin{}, fmt.Errorf("%w: %s", MissingRepositoryErr, name)
		}
		m, mOK := transformer["migrations"]
		if !mOK || m == "" {
			return Plugin{}, fmt.Errorf("%w: %s", MissingMigrationsErr, name)
		}
		mr, mrOK := transformer["rank"]
		if !mrOK || mr == "" {
			return Plugin{}, fmt.Errorf("%w: %s", MissingRankErr, name)
		}
		rank, err := strconv.ParseUint(mr, 10, 64)
		if err != nil {
			return Plugin{}, fmt.Errorf("%w: %s", RankParsingErr, name)
		}
		t, tOK := transformer["type"]
		if !tOK {
			return Plugin{}, fmt.Errorf("%w: %s", MissingTypeErr, name)
		}
		transformerType := GetTransformerType(t)
		if transformerType == UnknownTransformerType {
			return Plugin{}, UnknownTransformerTypeErr
		}

		transformers[name] = Transformer{
			Path:           p,
			Type:           transformerType,
			RepositoryPath: r,
			MigrationPath:  m,
			MigrationRank:  rank,
		}
	}

	return Plugin{
		Transformers: transformers,
		FilePath:     PluginFilePath,
		Schema:       viper.GetString("exporter.schema"),
		FileName:     viper.GetString("exporter.name"),
		Save:         viper.GetBool("exporter.save"),
		Home:         viper.GetString("exporter.home"),
	}, nil
}

func (pluginConfig *Plugin) GetPluginPaths() (string, string, error) {
	path, err := helpers.CleanPath(pluginConfig.FilePath)
	if err != nil {
		return "", "", err
	}

	name := strings.Split(pluginConfig.FileName, ".")[0]
	goFile := filepath.Join(path, name+".go")
	soFile := filepath.Join(path, name+".so")

	return goFile, soFile, nil
}

// Removes duplicate migration paths and returns them in ranked order
func (pluginConfig *Plugin) GetMigrationsPaths() ([]string, error) {
	paths := make(map[uint64]string)
	highestRank := -1
	for name, transformer := range pluginConfig.Transformers {
		repo := transformer.RepositoryPath
		mig := transformer.MigrationPath
		path := filepath.Join("$GOPATH/src", pluginConfig.Home, "vendor", repo, mig)
		cleanPath, err := helpers.CleanPath(path)
		if err != nil {
			return nil, err
		}
		// If there is a different path with the same rank then we have a conflict
		_, ok := paths[transformer.MigrationRank]
		if ok {
			conflictingPath := paths[transformer.MigrationRank]
			if conflictingPath != cleanPath {
				return nil, errors.New(fmt.Sprintf("transformer %s has the same migration rank (%d) as another transformer", name, transformer.MigrationRank))
			}
		}
		paths[transformer.MigrationRank] = cleanPath
		if int(transformer.MigrationRank) >= highestRank {
			highestRank = int(transformer.MigrationRank)
		}
	}
	// Check for gaps and duplicates
	if len(paths) != (highestRank + 1) {
		return []string{}, errors.New("number of distinct ranks does not match number of distinct migration paths")
	}
	if anyDupes(paths) {
		return []string{}, errors.New("duplicate paths with different ranks present")
	}

	sortedPaths := make([]string, len(paths))
	for rank, path := range paths {
		sortedPaths[rank] = path
	}

	return sortedPaths, nil
}

// Removes duplicate repo paths before returning them
func (pluginConfig *Plugin) GetRepoPaths() map[string]bool {
	paths := make(map[string]bool)
	for _, transformer := range pluginConfig.Transformers {
		paths[transformer.RepositoryPath] = true
	}

	return paths
}

type TransformerType int

const (
	UnknownTransformerType TransformerType = iota
	EthEvent
	EthStorage
	EthContract
)

func (transformerType TransformerType) String() string {
	names := [...]string{
		"Unknown",
		"eth_event",
		"eth_storage",
		"eth_contract",
	}

	if transformerType > EthContract || transformerType < EthEvent {
		return "Unknown"
	}

	return names[transformerType]
}

func GetTransformerType(str string) TransformerType {
	types := [...]TransformerType{
		EthEvent,
		EthStorage,
		EthContract,
	}

	for _, ty := range types {
		if ty.String() == str {
			return ty
		}
	}

	return UnknownTransformerType
}

func anyDupes(list map[uint64]string) bool {
	seen := make([]string, 0, len(list))
	for _, str := range list {
		dupe := inList(str, seen)
		if dupe {
			return true
		}
		seen = append(seen, str)
	}
	return false
}

func inList(str string, list []string) bool {
	for _, element := range list {
		if str == element {
			return true
		}
	}
	return false
}
