/*
    ACME-dispatcher -- Dispatch ACME challenge for a multihomed server
    Copyright (C) 2017 Star Brilliant <m13253@hotmail.com>

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
)

type config struct {
	Listen				string		`toml:"listen"`
	Path				string		`toml:"path"`
	Forward				[]string	`toml:"forward"`
	CircularPrevention	string		`toml:"circular_prevention"`
}

func loadConfig(path string) (*config, error) {
	conf := &config {}
	metaData, err := toml.DecodeFile(path, conf)
	if err != nil {
		return nil, err
	}
	for _, key := range metaData.Undecoded() {
		return nil, &configError { fmt.Sprintf("unknown option %q", key.String()) }
	}

	if conf.Listen == "" {
		conf.Listen = "[::1]:44046"
	}
	if conf.Path == "" {
		conf.Path = "/.well-known/acme-challenge/"
	}
	if conf.CircularPrevention == "" {
		conf.CircularPrevention = "X-ACME-Dispatcher"
	}
	return conf, nil
}

type configError struct {
	err		string
}

func (e *configError) Error() string {
	return e.err
}
