// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatprocessor_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/elastic/beats/v7/x-pack/libbeat/common/otelbeat/oteltestcol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkNoProcessor(b *testing.B) {
	config := `
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      exporters:
        - debug
  telemetry:
    metrics:
      level: none
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: benchmark
          id: produce-one-event
          count: 1
          message: "event-from-benchmark-input"
    processors: []
    queue.mem.flush.timeout: 0s
    path.home: %s
exporters:
  debug:
    verbosity: normal
`
	config = fmt.Sprintf(config, b.TempDir())

	for b.Loop() {
		collector := oteltestcol.New(b, config)
		require.EventuallyWithT(b,
			func(ct *assert.CollectT) {
				benchmarkEventLogs := collector.
					ObservedLogs().
					FilterMessageSnippet(`"message":"event-from-benchmark-input"`).
					All()
				require.Len(ct, benchmarkEventLogs, 1, "expected exactly one benchmark event log")
			}, 10*time.Second, 1*time.Millisecond)
	}
}

func BenchmarkProcessorInFilebeatReceiver(b *testing.B) {
	config := `
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      exporters:
        - debug
  telemetry:
    metrics:
      level: none
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: benchmark
          id: produce-one-event
          count: 1
          message: "event-from-benchmark-input"
    processors:
      - add_fields:
          fields:
            custom_field: custom-value
    queue.mem.flush.timeout: 0s
    path.home: %s
exporters:
  debug:
    verbosity: normal
`
	config = fmt.Sprintf(config, b.TempDir())

	for b.Loop() {
		collector := oteltestcol.New(b, config)
		require.EventuallyWithT(b,
			func(ct *assert.CollectT) {
				benchmarkEventLogs := collector.
					ObservedLogs().
					FilterMessageSnippet(`"message":"event-from-benchmark-input"`).
					All()
				require.Len(ct, benchmarkEventLogs, 1, "expected exactly one benchmark event log")
			}, 10*time.Second, 1*time.Millisecond)
	}
}

func BenchmarkProcessingInBeatProcessor(b *testing.B) {
	config := `
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      processors:
        - beat
      exporters:
        - debug
  telemetry:
    metrics:
      level: none
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: benchmark
          id: produce-one-event
          count: 1
          message: "event-from-benchmark-input"
    processors: []
    queue.mem.flush.timeout: 0s
    path.home: %s
processors:
  beat:
    processors:
      - add_fields:
          fields:
            custom_field: custom-value
exporters:
  debug:
    verbosity: normal
`
	config = fmt.Sprintf(config, b.TempDir())

	for b.Loop() {
		collector := oteltestcol.New(b, config)
		require.EventuallyWithT(b,
			func(ct *assert.CollectT) {
				benchmarkEventLogs := collector.
					ObservedLogs().
					FilterMessageSnippet(`"message":"event-from-benchmark-input"`).
					All()
				require.Len(ct, benchmarkEventLogs, 1, "expected exactly one benchmark event log")
			}, 10*time.Second, 1*time.Millisecond)
	}
}

func TestOteltestcolNew(t *testing.T) {
	config := `
service:
  pipelines:
    logs:
      receivers:
        - filebeatreceiver
      exporters:
        - debug
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: benchmark
          id: produce-one-event
          count: 1
          message: "event-from-benchmark-input"
    processors:
      - add_fields:
          fields:
            custom_field: custom-value
    queue.mem.flush.timeout: 0s
    path.home: %s
exporters:
  debug:
    verbosity: normal
`
	config = fmt.Sprintf(config, t.TempDir())

	collector := oteltestcol.New(t, config)
	require.EventuallyWithT(t,
		func(ct *assert.CollectT) {
			benchmarkEventLogs := collector.
				ObservedLogs().
				FilterMessageSnippet(`"message":"event-from-benchmark-input"`).
				All()
			assert.Len(ct, benchmarkEventLogs, 1, "expected exactly one benchmark event log")
		}, 10*time.Second, 10*time.Millisecond)

	// for _, log := range collector.ObservedLogs().All() {
	// 	fmt.Println(log.Message)
	// }
}
