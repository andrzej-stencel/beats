// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package api

import (
	"fmt"
	"net/http"
	"net/url"

	"go.uber.org/multierr"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// RegistryLookupFunc is used to look up a registry by its path inside a
// containing registry.
func RegistryLookupFunc(root *monitoring.Registry) LookupFunc {
	return func(s string) *monitoring.Registry {
		return root.GetRegistry(s)
	}
}

type LookupFunc func(string) *monitoring.Registry

// NewWithDefaultRoutes creates a new server with default API routes.
func NewWithDefaultRoutes(log *logp.Logger, config *config.C,
	info, state, stats, inputs *monitoring.Registry) (*Server, error) {
	api, err := New(log, config)
	if err != nil {
		return nil, err
	}

	err = multierr.Combine(
		api.AttachHandler("/", makeRootAPIHandler(makeAPIHandler(info))),
		api.AttachHandler("/state", makeAPIHandler(state)),
		api.AttachHandler("/stats", makeAPIHandler(stats)),
		api.AttachHandler("/dataset", makeAPIHandler(inputs)),
	)
	if err != nil {
		return nil, err
	}

	return api, nil
}

func makeRootAPIHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		handler(w, r)
	}
}

func makeAPIHandler(registry *monitoring.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		data := monitoring.CollectStructSnapshot(
			registry,
			monitoring.Full,
			false,
		)

		prettyPrint(w, data, r.URL)
	}
}

func prettyPrint(w http.ResponseWriter, data mapstr.M, u *url.URL) {
	query := u.Query()
	if _, ok := query["pretty"]; ok {
		fmt.Fprint(w, data.StringToPrint())
	} else {
		fmt.Fprint(w, data.String())
	}
}
