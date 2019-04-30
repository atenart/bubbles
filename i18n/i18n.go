// Copyright (C) 2019 Antoine Tenart <antoine.tenart@ack.tf>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package i18n

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	yaml "gopkg.in/yaml.v2"
)

type Bundle struct {
	*i18n.Bundle
}

type Localizer struct {
	*i18n.Localizer
}

// Initialize the i18n process.
func Init() (*Bundle, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

	err := filepath.Walk("i18n/lc", func(path string, info os.FileInfo, err error) error {
		stat, err := os.Stat(path)
		if os.IsNotExist(err) {
			return err
		}

		// Check if we deal with a regular file.
		if stat.Mode() & os.ModeType != 0 {
			return nil
		}

		if !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		if _, err := bundle.LoadMessageFile(path); err != nil {
			return err
		}

		return nil
	})

	return &Bundle{ bundle }, err
}

// Returns a new localizer.
func (b *Bundle) Localizer(langs ...string) *Localizer {
	return &Localizer{ i18n.NewLocalizer(b.Bundle, langs...) }
}

// Translate ("localize") an message idientified by an id into a string.
func (l *Localizer) Localize(id string) string {
	return l.MustLocalize(&i18n.LocalizeConfig{
		MessageID: id,
		// Fallback to the id (english).
		DefaultMessage: &i18n.Message{
			ID:    id,
			Other: id,
		},
	})
}
