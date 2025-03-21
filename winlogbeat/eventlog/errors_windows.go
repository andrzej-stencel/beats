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

package eventlog

import (
	"errors"
	"io"

	win "github.com/elastic/beats/v7/winlogbeat/sys/wineventlog"
)

// IsRecoverable returns a boolean indicating whether the error represents
// a condition where the Windows Event Log session can be recovered through a
// reopening of the handle (Close, Open).
//
//nolint:errorlint // These are never wrapped.
func IsRecoverable(err error, isFile bool) bool {
	return err == win.ERROR_INVALID_HANDLE ||
		err == win.RPC_S_SERVER_UNAVAILABLE ||
		err == win.RPC_S_CALL_CANCELLED ||
		err == win.ERROR_EVT_QUERY_RESULT_STALE ||
		err == win.ERROR_INVALID_PARAMETER ||
		err == win.ERROR_EVT_PUBLISHER_DISABLED ||
		(!isFile && errors.Is(err, io.EOF)) ||
		(!isFile && errors.Is(err, win.ERROR_EVT_CHANNEL_NOT_FOUND))
}
