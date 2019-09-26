// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package echo

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	"github.com/spf13/pflag"
)

type echo struct {
}

func New() plugins.Plugin {
	return &echo{}
}

func (e *echo) Command() string {
	return "echo"
}

func (e *echo) Description() string {
	return "Prints the provided value"
}

func (e *echo) Example() string {
	return "/echo --val \"text to echo\""
}

func (e *echo) Flags() *pflag.FlagSet {
	flagset := pflag.NewFlagSet(e.Command(), pflag.ContinueOnError)
	flagset.StringP("value", "v", "", "Echo value")
	return flagset
}

func (e *echo) Run(flagset *pflag.FlagSet, client github.Client, event *github.GenericRequestEvent) error {
	val, err := flagset.GetString("value")
	if err != nil {
		return err
	}
	return client.Respond(event, fmt.Sprintf("@%s: %s", event.GetAuthorName(), val))
}