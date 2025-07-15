// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatprocessor

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors/add_host_metadata"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

type beatProcessor struct {
	hostProcessor beat.Processor
}

func newBeatProcessor() (*beatProcessor, error) {
	hostProcessor, err := add_host_metadata.New(config.NewConfig(), logp.NewLogger("beatprocessor"))
	if err != nil {
		return nil, err
	}
	processor := &beatProcessor{
		hostProcessor: hostProcessor,
	}
	return processor, nil
}

func (p *beatProcessor) ConsumeLogs(_ context.Context, logs plog.Logs) (plog.Logs, error) {
	fmt.Println("OTelBeatProcessor: logs:", logs.LogRecordCount())
	dummyEvent := &beat.Event{}
	dummyEvent.Fields = mapstr.M{}
	dummyEvent.Meta = mapstr.M{}
	dummyEventWithHostMetadata, err := p.hostProcessor.Run(dummyEvent)
	if err != nil {
		fmt.Println("Error processing host metadata:", err)
	}
	hostMap := dummyEventWithHostMetadata.Fields["host"].(mapstr.M)
	otelMap := toOtelMap(&hostMap)
	for _, resourceLogs := range logs.ResourceLogs().All() {
		for _, scopeLogs := range resourceLogs.ScopeLogs().All() {
			for _, logRecord := range scopeLogs.LogRecords().All() {
				bodyMap := logRecord.Body().Map().PutEmptyMap("host")
				otelMap.CopyTo(bodyMap)
			}
		}
	}
	return logs, nil
}

func toOtelMap(m *mapstr.M) pcommon.Map {
	otelMap := pcommon.NewMap()
	for key, value := range *m {
		switch typedValue := value.(type) {
		case mapstr.M:
			subMap := toOtelMap(&typedValue)
			otelSubMap := otelMap.PutEmptyMap(key)
			subMap.MoveTo(otelSubMap)
		case []string:
			otelValue := otelMap.PutEmptySlice(key)
			for _, item := range typedValue {
				otelValue.AppendEmpty().SetStr(item)
			}
		default:
			otelValue := otelMap.PutEmpty(key)
			otelValue.FromRaw(typedValue)
		}
	}
	return otelMap
}
