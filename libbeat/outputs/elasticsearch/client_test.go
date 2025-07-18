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

//go:build !integration

package elasticsearch

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	e "github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/idxmgmt"
	"github.com/elastic/beats/v7/libbeat/internal/testutil"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/libbeat/version"
	c "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	libversion "github.com/elastic/elastic-agent-libs/version"
)

type batchMock struct {
	events      []publisher.Event
	ack         bool
	drop        bool
	canSplit    bool
	didSplit    bool
	retryEvents []publisher.Event
}

func (bm batchMock) Events() []publisher.Event {
	return bm.events
}
func (bm *batchMock) ACK() {
	bm.ack = true
}
func (bm *batchMock) Drop() {
	bm.drop = true
}
func (bm *batchMock) Retry()     { panic("unimplemented") }
func (bm *batchMock) Cancelled() { panic("unimplemented") }
func (bm *batchMock) SplitRetry() bool {
	if bm.canSplit {
		bm.didSplit = true
	}
	return bm.canSplit
}
func (bm *batchMock) RetryEvents(events []publisher.Event) {
	bm.retryEvents = events
}

func TestPublish(t *testing.T) {

	logger := logptest.NewTestingLogger(t, "")
	makePublishTestClient := func(t *testing.T, url string) (*Client, *monitoring.Registry) {
		reg := monitoring.NewRegistry()
		client, err := NewClient(
			clientSettings{
				observer:      outputs.NewStats(reg),
				connection:    eslegclient.ConnectionSettings{URL: url},
				indexSelector: testIndexSelector{},
			},
			nil,
			logger,
		)
		require.NoError(t, err)
		return client, reg
	}
	makePublishGzipTestClient := func(t *testing.T, url string, compressionLevel int) (*Client, *monitoring.Registry) {
		reg := monitoring.NewRegistry()
		client, err := NewClient(
			clientSettings{
				observer: outputs.NewStats(reg),
				connection: eslegclient.ConnectionSettings{
					URL:              url,
					CompressionLevel: compressionLevel,
				},
				indexSelector: testIndexSelector{},
			},
			nil,
			logger,
		)
		require.NoError(t, err)
		return client, reg
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	event1 := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 1}}}
	event2 := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 2}}}
	event3 := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 3}}}

	t.Run("splits large batches on status code 413", func(t *testing.T) {
		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_, _ = w.Write([]byte("Request failed to get to the server (status code: 413)")) // actual response from ES
		}))
		defer esMock.Close()
		client, reg := makePublishTestClient(t, esMock.URL)

		// Try publishing a batch that can be split
		batch := encodeBatch(client, &batchMock{
			events:   []publisher.Event{event1},
			canSplit: true,
		})
		err := client.Publish(ctx, batch)

		assert.NoError(t, err, "Publish should split the batch without error")
		assert.True(t, batch.didSplit, "batch should be split")
		assertRegistryUint(t, reg, "events.failed", 1, "Splitting a batch should report the event as failed/retried")
		assertRegistryUint(t, reg, "events.dropped", 0, "Splitting a batch should not report any dropped events")

		// Try publishing a batch that cannot be split
		batch = encodeBatch(client, &batchMock{
			events:   []publisher.Event{event1},
			canSplit: false,
		})
		err = client.Publish(ctx, batch)

		assert.NoError(t, err, "Publish should drop the batch without error")
		assert.False(t, batch.didSplit, "batch should not be split")
		assert.True(t, batch.drop, "unsplittable batch should be dropped")
		assertRegistryUint(t, reg, "events.failed", 1, "Failed batch split should not report any more retryable failures")
		assertRegistryUint(t, reg, "events.dropped", 1, "Failed batch split should report a dropped event")

	})

	t.Run("retries the batch if bad HTTP status", func(t *testing.T) {
		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer esMock.Close()
		client, reg := makePublishTestClient(t, esMock.URL)

		batch := encodeBatch(client, &batchMock{
			events: []publisher.Event{event1, event2},
		})

		err := client.Publish(ctx, batch)

		assert.Error(t, err)
		assert.False(t, batch.ack, "should not be acknowledged")
		assert.Len(t, batch.retryEvents, 2, "all events should be retried")
		assertRegistryUint(t, reg, "events.failed", 2, "HTTP failure should report failed events")
	})

	t.Run("live batches, still too big after split", func(t *testing.T) {
		// Test a live (non-mocked) batch where all three events by themselves are
		// rejected by the server as too large after the initial batch splits.
		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_, _ = w.Write([]byte("Request failed to get to the server (status code: 413)")) // actual response from ES
		}))
		defer esMock.Close()
		client, reg := makePublishTestClient(t, esMock.URL)

		// Because our tests don't use a live eventConsumer routine,
		// everything will happen synchronously and it's safe to track
		// test results directly without atomics/mutexes.
		done := false
		retryCount := 0
		var retryBatches []publisher.Batch
		batch := encodeBatch(client, pipeline.NewBatchForTesting(
			[]publisher.Event{event1, event2, event3},
			func(b publisher.Batch) {
				// The retry function sends the batch back through Publish.
				// In a live pipeline it would instead be sent to eventConsumer
				// and then back to Publish when an output worker was available.
				retryCount++
				retryBatches = append(retryBatches, b)
			},
			func() { done = true },
		))
		retryBatches = []publisher.Batch{batch}
		// Loop until all pending retries are complete, the same as a pipeline caller would.
		for len(retryBatches) > 0 {
			batch := retryBatches[0]
			retryBatches = retryBatches[1:]
			err := client.Publish(ctx, batch)
			assert.NoError(t, err, "Publish should return without error")
		}

		// For three events there should be four retries in total:
		// {[event1], [event2, event3]}, then {[event2], [event3]}.
		// "done" should be true because after splitting into individual
		// events, all 3 will fail and be dropped.
		assert.Equal(t, 4, retryCount, "3-event batch should produce 4 total retries")
		assert.True(t, done, "batch should be marked as done")
		// Metrics should report:
		// 8 total events (3 + 1 + 2 + 1 + 1 from the batches described above)
		// 3 dropped events (each event is dropped once)
		// 5 failed events (8 - 3, for each event's attempted publish calls before being dropped)
		// 0 active events (because Publish is complete)
		assertRegistryUint(t, reg, "events.total", 8, "Publish is called on 8 events total")
		assertRegistryUint(t, reg, "events.dropped", 3, "All 3 events should be dropped")
		assertRegistryUint(t, reg, "events.failed", 5, "Split batches should retry 5 events before dropping them")
		assertRegistryUint(t, reg, "events.active", 0, "Active events should be zero when Publish returns")
	})

	t.Run("live batches, one event too big after split", func(t *testing.T) {
		// Test a live (non-mocked) batch where a single event is too large
		// for the server to ingest but the others are ok.
		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			body := string(b)
			// Reject the batch as too large only if it contains event1
			if strings.Contains(body, "\"field\":1") {
				// Report batch too large
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				_, _ = w.Write([]byte("Request failed to get to the server (status code: 413)")) // actual response from ES
			} else {
				// Report success with no events dropped
				w.WriteHeader(200)
				_, _ = io.WriteString(w, "{\"items\": [{\"index\":{\"status\":200}},{\"index\":{\"status\":200}},{\"index\":{\"status\":200}}]}")
			}
		}))
		defer esMock.Close()
		client, reg := makePublishTestClient(t, esMock.URL)

		// Because our tests don't use a live eventConsumer routine,
		// everything will happen synchronously and it's safe to track
		// test results directly without atomics/mutexes.
		done := false
		retryCount := 0
		var retryBatches []publisher.Batch
		batch := encodeBatch(client, pipeline.NewBatchForTesting(
			[]publisher.Event{event1, event2, event3},
			func(b publisher.Batch) {
				// The retry function sends the batch back through Publish.
				// In a live pipeline it would instead be sent to eventConsumer
				// and then back to Publish when an output worker was available.
				retryCount++
				retryBatches = append(retryBatches, b)
			},
			func() { done = true },
		))
		retryBatches = []publisher.Batch{batch}
		for len(retryBatches) > 0 {
			batch := retryBatches[0]
			retryBatches = retryBatches[1:]
			err := client.Publish(ctx, batch)
			assert.NoError(t, err, "Publish should return without error")
		}

		// There should be two retries: {[event1], [event2, event3]}.
		// The first split batch should fail and be dropped since it contains
		// event1, the other one should succeed.
		// "done" should be true because both split batches are completed
		// (one with failure, one with success).
		assert.Equal(t, 2, retryCount, "splitting with one large event should produce two retries")
		assert.True(t, done, "batch should be marked as done")
		// The metrics should show:
		// 6 total events (3 + 1 + 2)
		// 1 dropped event (because only one event is uningestable)
		// 2 acked events (because the other two ultimately succeed)
		// 3 failed events (because all events fail and are retried on the first call)
		// 0 active events (because Publish is finished)
		assertRegistryUint(t, reg, "events.total", 6, "Publish is called on 6 events total")
		assertRegistryUint(t, reg, "events.dropped", 1, "One event should be dropped")
		assertRegistryUint(t, reg, "events.failed", 3, "Split batches should retry 3 events before dropping them")
		assertRegistryUint(t, reg, "events.active", 0, "Active events should be zero when Publish returns")
	})

	t.Run("sends telemetry headers", func(t *testing.T) {
		events := []publisher.Event{event1, event2, event3}
		eventsRaw := `{"index":{"_index":"test","_type":"doc"}}
{"@timestamp":"0001-01-01T00:00:00.000Z","field":1}

{"index":{"_index":"test","_type":"doc"}}
{"@timestamp":"0001-01-01T00:00:00.000Z","field":2}

{"index":{"_index":"test","_type":"doc"}}
{"@timestamp":"0001-01-01T00:00:00.000Z","field":3}

`
		batch := &batchMock{
			events: events,
		}

		var requestCount, uncompressedLength, eventCount atomic.Int64
		buf := bytes.NewBuffer(nil)
		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)

			ul := r.Header.Get(eslegclient.HeaderUncompressedLength)
			if ul != "" {
				l, _ := strconv.ParseInt(ul, 10, 64)
				uncompressedLength.Store(l)
			}
			ec := r.Header.Get(HeaderEventCount)
			if ec != "" {
				c, _ := strconv.ParseInt(ec, 10, 64)
				eventCount.Store(c)
			}

			body := r.Body
			if r.Header.Get("Content-Encoding") == "gzip" {
				body, _ = gzip.NewReader(body)
			}
			io.Copy(buf, body)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"took": 30, "errors": false, "items": [] }`))
		}))
		defer esMock.Close()
		client, _ := makePublishTestClient(t, esMock.URL)
		gzipClient, _ := makePublishGzipTestClient(t, esMock.URL, 5)

		cases := []struct {
			name   string
			client *Client
		}{
			{
				name:   "uncompressed",
				client: client,
			},
			{
				name:   "compressed",
				client: gzipClient,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				buf.Reset()
				requestCount.Store(0)
				// set to -1 to assert the assignment later
				uncompressedLength.Store(-1)
				eventCount.Store(-1)

				batch := encodeBatch(tc.client, batch)
				err := tc.client.Publish(ctx, batch)

				assert.NoError(t, err, "Publish should drop the batch without error")
				assert.Equal(t, int64(1), requestCount.Load(), "Should process only one request")
				assert.Equal(t, int64(len(eventsRaw)), uncompressedLength.Load(), "Should have the correct %s header", eslegclient.HeaderUncompressedLength)
				assert.Equal(t, int64(3), eventCount.Load(), "Should have the correct %s header", HeaderEventCount)
				assert.Equal(t, eventsRaw, buf.String(), "Should have the correct body")
			})
		}
	})
}

func assertRegistryUint(t *testing.T, reg *monitoring.Registry, key string, expected uint64, message string) {
	t.Helper()
	value := reg.Get(key).(*monitoring.Uint)
	assert.NotNilf(t, value, "expected registry entry for key '%v'", key)
	assert.Equal(t, expected, value.Get(), message)
}

func TestCollectPublishFailsNone(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	client, err := NewClient(
		clientSettings{
			observer: outputs.NewNilObserver(),
		},
		nil,
		logger,
	)
	assert.NoError(t, err)

	N := 100
	item := `{"create": {"status": 200}},`
	response := []byte(`{"items": [` + strings.Repeat(item, N) + `]}`)

	event := mapstr.M{"field": 1}
	events := make([]publisher.Event, N)
	for i := 0; i < N; i++ {
		events[i] = publisher.Event{Content: beat.Event{Fields: event}}
	}

	res, _ := client.bulkCollectPublishFails(bulkResult{
		events:   encodeEvents(client, events),
		status:   200,
		response: response,
	})
	assert.Equal(t, 0, len(res))
}

func TestCollectPublishFailMiddle(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	client, err := NewClient(
		clientSettings{
			observer: outputs.NewNilObserver(),
		},
		nil,
		logger,
	)
	assert.NoError(t, err)

	response := []byte(`
    { "items": [
      {"create": {"status": 200}},
      {"create": {"status": 429, "error": "ups"}},
      {"create": {"status": 200}}
    ]}
  `)

	event1 := encodeEvent(client, publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 1}}})
	event2 := encodeEvent(client, publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 2}}})
	eventFail := encodeEvent(client, publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 3}}})
	events := []publisher.Event{event1, eventFail, event2}

	res, stats := client.bulkCollectPublishFails(bulkResult{
		events:   events,
		status:   200,
		response: response,
	})
	assert.Equal(t, 1, len(res))
	if len(res) == 1 {
		assert.Equal(t, eventFail, res[0])
	}
	assert.Equal(t, bulkResultStats{acked: 2, fails: 1, tooMany: 1}, stats)
}

func TestCollectPublishFailDeadLetterSuccess(t *testing.T) {
	const deadLetterIndex = "test_index"
	logger := logptest.NewTestingLogger(t, "")
	client, err := NewClient(
		clientSettings{
			observer:        outputs.NewNilObserver(),
			deadLetterIndex: deadLetterIndex,
		},
		nil,
		logger,
	)
	assert.NoError(t, err)

	const errorMessage = "test error message"
	// Return a successful response
	response := []byte(`{"items": [{"create": {"status": 200}}]}`)

	event1 := encodeEvent(client, publisher.Event{Content: beat.Event{Fields: mapstr.M{"bar": 1}}})
	event1.EncodedEvent.(*encodedEvent).setDeadLetter(deadLetterIndex, 123, errorMessage)
	events := []publisher.Event{event1}

	// The event should be successful after being set to dead letter, so it
	// should be reported in the metrics as deadLetter
	res, stats := client.bulkCollectPublishFails(bulkResult{
		events:   events,
		status:   200,
		response: response,
	})
	assert.Equal(t, bulkResultStats{acked: 0, deadLetter: 1}, stats)
	assert.Equal(t, 0, len(res))
}

func TestCollectPublishFailFatalErrorNotRetried(t *testing.T) {
	// Test that a fatal error sending to the dead letter index is reported as
	// a dropped event, and is not retried forever
	const deadLetterIndex = "test_index"
	logger := logptest.NewTestingLogger(t, "")
	client, err := NewClient(
		clientSettings{
			observer:        outputs.NewNilObserver(),
			deadLetterIndex: deadLetterIndex,
		},
		nil,
		logger,
	)
	assert.NoError(t, err)

	const errorMessage = "test error message"
	// Return a fatal error
	response := []byte(`{"items": [{"create": {"status": 499}}]}`)

	event1 := encodeEvent(client, publisher.Event{Content: beat.Event{Fields: mapstr.M{"bar": 1}}})
	event1.EncodedEvent.(*encodedEvent).setDeadLetter(deadLetterIndex, 123, errorMessage)
	events := []publisher.Event{event1}

	// The event should fail permanently while being sent to the dead letter
	// index, so it should be dropped instead of retrying.
	res, stats := client.bulkCollectPublishFails(bulkResult{
		events:   events,
		status:   200,
		response: response,
	})
	assert.Equal(t, bulkResultStats{acked: 0, nonIndexable: 1}, stats)
	assert.Equal(t, 0, len(res))
}

func TestCollectPublishFailInvalidBulkIndexResponse(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	client, err := NewClient(
		clientSettings{observer: outputs.NewNilObserver()},
		nil,
		logger,
	)
	assert.NoError(t, err)

	// Return a truncated response without valid item data
	response := []byte(`{"items": [...`)

	event1 := encodeEvent(client, publisher.Event{Content: beat.Event{Fields: mapstr.M{"bar": 1}}})
	events := []publisher.Event{event1}

	// The event should be successful after being set to dead letter, so it
	// should be reported in the metrics as deadLetter
	res, stats := client.bulkCollectPublishFails(bulkResult{
		events:   events,
		status:   200,
		response: response,
	})
	// The event should be returned for retry, and should appear in aggregated
	// stats as failed (retryable error)
	assert.Equal(t, bulkResultStats{acked: 0, fails: 1}, stats)
	assert.Equal(t, 1, len(res))
	if len(res) > 0 {
		assert.Equal(t, event1, res[0])
	}
}

func TestCollectPublishFailDeadLetterIndex(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	const deadLetterIndex = "test_index"
	client, err := NewClient(
		clientSettings{
			observer:        outputs.NewNilObserver(),
			deadLetterIndex: deadLetterIndex,
		},
		nil,
		logger,
	)
	assert.NoError(t, err)

	const errorMessage = "test error message"
	response := []byte(`
{
	"items": [
		{"create": {"status": 200}},
		{
			"create": {
				"error" : "` + errorMessage + `",
				"status" : 400
			}
		},
		{"create": {"status": 200}}
	]
}`)

	event1 := encodeEvent(client, publisher.Event{Content: beat.Event{Fields: mapstr.M{"bar": 1}}})
	event2 := encodeEvent(client, publisher.Event{Content: beat.Event{Fields: mapstr.M{"bar": 2}}})
	eventFail := encodeEvent(client, publisher.Event{Content: beat.Event{Fields: mapstr.M{"bar": "bar1"}}})
	events := []publisher.Event{event1, eventFail, event2}

	res, stats := client.bulkCollectPublishFails(bulkResult{
		events:   events,
		status:   200,
		response: response,
	})
	assert.Equal(t, bulkResultStats{acked: 2, fails: 1, nonIndexable: 0}, stats)
	assert.Equal(t, 1, len(res))
	if len(res) == 1 {
		assert.Equalf(t, eventFail, res[0], "bulkCollectPublishFails should return failed event")
		encodedEvent, ok := res[0].EncodedEvent.(*encodedEvent)
		require.True(t, ok, "event must be encoded as *encodedEvent")
		assert.True(t, encodedEvent.deadLetter, "failed event's dead letter flag should be set")
		assert.Equalf(t, deadLetterIndex, encodedEvent.index, "failed event's index should match dead letter index")
		assert.Contains(t, string(encodedEvent.encoding), errorMessage, "dead letter event should include associated error message")
	}
}

func TestCollectPublishFailDrop(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	client, err := NewClient(
		clientSettings{
			observer:        outputs.NewNilObserver(),
			deadLetterIndex: "",
		},
		nil,
		logger,
	)
	assert.NoError(t, err)

	response := []byte(`
    { "items": [
      {"create": {"status": 200}},
      {"create": {
		  "error" : {
			"root_cause" : [
			  {
				"type" : "mapper_parsing_exception",
				"reason" : "failed to parse field [bar] of type [long] in document with id '1'. Preview of field's value: 'bar1'"
			  }
			],
			"type" : "mapper_parsing_exception",
			"reason" : "failed to parse field [bar] of type [long] in document with id '1'. Preview of field's value: 'bar1'",
			"caused_by" : {
			  "type" : "illegal_argument_exception",
			  "reason" : "For input string: \"bar1\""
			}
		  },
		  "status" : 400
		}
      },
      {"create": {"status": 200}}
    ]}
  `)

	event := publisher.Event{Content: beat.Event{Fields: mapstr.M{"bar": 1}}}
	eventFail := publisher.Event{Content: beat.Event{Fields: mapstr.M{"bar": "bar1"}}}
	events := encodeEvents(client, []publisher.Event{event, eventFail, event})

	res, stats := client.bulkCollectPublishFails(bulkResult{
		events:   events,
		status:   200,
		response: response,
	})
	assert.Equal(t, 0, len(res))
	assert.Equal(t, bulkResultStats{acked: 2, fails: 0, nonIndexable: 1}, stats)
}

func TestCollectPublishFailAll(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	client, err := NewClient(
		clientSettings{
			observer: outputs.NewNilObserver(),
		},
		nil,
		logger,
	)
	assert.NoError(t, err)

	response := []byte(`
    { "items": [
      {"create": {"status": 429, "error": "ups"}},
      {"create": {"status": 429, "error": "ups"}},
      {"create": {"status": 429, "error": "ups"}}
    ]}
  `)

	event := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 2}}}
	events := encodeEvents(client, []publisher.Event{event, event, event})

	res, stats := client.bulkCollectPublishFails(bulkResult{
		events:   events,
		status:   200,
		response: response,
	})
	assert.Equal(t, 3, len(res))
	assert.Equal(t, events, res)
	assert.Equal(t, stats, bulkResultStats{fails: 3, tooMany: 3})
}

func TestCollectPipelinePublishFail(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	client, err := NewClient(
		clientSettings{
			observer: outputs.NewNilObserver(),
		},
		nil,
		logger,
	)
	assert.NoError(t, err)

	response := []byte(`{
      "took": 0, "ingest_took": 0, "errors": true,
      "items": [
        {
          "index": {
            "_index": "filebeat-2016.08.10",
            "_type": "log",
            "_id": null,
            "status": 500,
            "error": {
              "type": "exception",
              "reason": "java.lang.IllegalArgumentException: java.lang.IllegalArgumentException: field [fail_on_purpose] not present as part of path [fail_on_purpose]",
              "caused_by": {
                "type": "illegal_argument_exception",
                "reason": "java.lang.IllegalArgumentException: field [fail_on_purpose] not present as part of path [fail_on_purpose]",
                "caused_by": {
                  "type": "illegal_argument_exception",
                  "reason": "field [fail_on_purpose] not present as part of path [fail_on_purpose]"
                }
              },
              "header": {
                "processor_type": "lowercase"
              }
            }
          }
        }
      ]
    }`)

	event := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 2}}}
	events := encodeEvents(client, []publisher.Event{event})

	res, _ := client.bulkCollectPublishFails(bulkResult{
		events:   events,
		status:   200,
		response: response,
	})
	assert.Equal(t, 1, len(res))
	assert.Equal(t, events, res)
}

func TestPublishResultForStats(t *testing.T) {
	// publishResultForStats should return errTooMany if it is given
	// stats with tooMany > 0, and nil otherwise (all other errors are
	// either caused by encoding or connection failures, or are
	// immediately retryable).
	stats := bulkResultStats{
		acked:        1,
		duplicates:   2,
		fails:        3,
		nonIndexable: 4,
		deadLetter:   5,
		tooMany:      1,
	}

	assert.Equal(t, errTooMany, publishResultForStats(stats), "publishResultForStats should return errTooMany if tooMany > 0")

	stats.tooMany = 0
	assert.Nil(t, publishResultForStats(stats), "publishResultForStats should return nil if tooMany == 0")
}

func BenchmarkCollectPublishFailsNone(b *testing.B) {

	client, err := NewClient(
		clientSettings{
			observer:        outputs.NewNilObserver(),
			deadLetterIndex: "",
		},
		nil,
		logp.NewNopLogger(), // we use no-op logger so that it does not skew benchmark results
	)
	assert.NoError(b, err)

	response := []byte(`
    { "items": [
      {"create": {"status": 200}},
      {"create": {"status": 200}},
      {"create": {"status": 200}}
    ]}
  `)

	event := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 1}}}
	events := encodeEvents(client, []publisher.Event{event, event, event})

	for i := 0; i < b.N; i++ {
		res, _ := client.bulkCollectPublishFails(bulkResult{
			events:   events,
			status:   200,
			response: response,
		})
		if len(res) != 0 {
			b.Fail()
		}
	}
}

func BenchmarkCollectPublishFailMiddle(b *testing.B) {
	client, err := NewClient(
		clientSettings{
			observer: outputs.NewNilObserver(),
		},
		nil,
		logp.NewNopLogger(),
	)
	assert.NoError(b, err)

	response := []byte(`
    { "items": [
      {"create": {"status": 200}},
      {"create": {"status": 429, "error": "ups"}},
      {"create": {"status": 200}}
    ]}
  `)

	event := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 1}}}
	eventFail := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 2}}}
	events := encodeEvents(client, []publisher.Event{event, eventFail, event})

	for i := 0; i < b.N; i++ {
		res, _ := client.bulkCollectPublishFails(bulkResult{
			events:   events,
			status:   200,
			response: response,
		})
		if len(res) != 1 {
			b.Fail()
		}
	}
}

func BenchmarkCollectPublishFailAll(b *testing.B) {
	client, err := NewClient(
		clientSettings{
			observer: outputs.NewNilObserver(),
		},
		nil,
		logp.NewNopLogger(),
	)
	assert.NoError(b, err)

	response := []byte(`
    { "items": [
      {"creatMiddlee": {"status": 429, "error": "ups"}},
      {"creatMiddlee": {"status": 429, "error": "ups"}},
      {"creatMiddlee": {"status": 429, "error": "ups"}}
    ]}
  `)

	event := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 2}}}
	events := encodeEvents(client, []publisher.Event{event, event, event})

	for i := 0; i < b.N; i++ {
		res, _ := client.bulkCollectPublishFails(bulkResult{
			events:   events,
			status:   200,
			response: response,
		})
		if len(res) != 3 {
			b.Fail()
		}
	}
}

func BenchmarkPublish(b *testing.B) {
	tests := []struct {
		Name   string
		Events []beat.Event
	}{
		{
			Name:   "5 events",
			Events: testutil.GenerateEvents(50, 5, 3),
		},
		{
			Name:   "50 events",
			Events: testutil.GenerateEvents(500, 5, 3),
		},
		{
			Name:   "500 events",
			Events: testutil.GenerateEvents(500, 5, 3),
		},
	}

	levels := []int{1, 4, 7, 9}

	requestCount := 0

	// start a mock HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(b, "testing value", r.Header.Get("X-Test"))
		// from the documentation: https://golang.org/pkg/net/http/
		// For incoming requests, the Host header is promoted to the
		// Request.Host field and removed from the Header map.
		assert.Equal(b, "myhost.local", r.Host)

		var response string
		if r.URL.Path == "/" {
			response = `{ "version": { "number": "7.6.0" } }`
		} else {
			response = `{"items":[{"index":{}},{"index":{}},{"index":{}}]}`

		}
		fmt.Fprintln(w, response)
		requestCount++
	}))
	defer ts.Close()

	// Indexing to _bulk api
	for _, test := range tests {
		for _, l := range levels {
			b.Run(fmt.Sprintf("%s with compression level %d", test.Name, l), func(b *testing.B) {
				client, err := NewClient(
					clientSettings{
						connection: eslegclient.ConnectionSettings{
							URL: ts.URL,
							Headers: map[string]string{
								"host":   "myhost.local",
								"X-Test": "testing value",
							},
							CompressionLevel: l,
						},
					},
					nil,
					logp.NewNopLogger(),
				)
				assert.NoError(b, err)
				batch := encodeBatch(client, outest.NewBatch(test.Events...))

				// It uses gzip encoder internally for encoding data
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					err := client.Publish(context.Background(), batch)
					assert.NoError(b, err)
				}
			})

		}
	}

}

func TestClientWithHeaders(t *testing.T) {
	requestCount := 0
	// start a mock HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "testing value", r.Header.Get("X-Test"))
		// from the documentation: https://golang.org/pkg/net/http/
		// For incoming requests, the Host header is promoted to the
		// Request.Host field and removed from the Header map.
		assert.Equal(t, "myhost.local", r.Host)

		var response string
		if r.URL.Path == "/" {
			response = `{ "version": { "number": "7.6.0" } }`
		} else {
			response = `{"items":[{"index":{}},{"index":{}},{"index":{}}]}`

		}
		fmt.Fprintln(w, response)
		requestCount++
	}))
	defer ts.Close()

	logger := logptest.NewTestingLogger(t, "")
	client, err := NewClient(clientSettings{
		observer: outputs.NewNilObserver(),
		connection: eslegclient.ConnectionSettings{
			URL: ts.URL,
			Headers: map[string]string{
				"host":   "myhost.local",
				"X-Test": "testing value",
			},
		},
		indexSelector: outil.MakeSelector(outil.ConstSelectorExpr("test", outil.SelectorLowerCase)),
	}, nil, logger)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	// simple ping
	err = client.Connect(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, requestCount)

	// bulk request
	event := beat.Event{Fields: mapstr.M{
		"@timestamp": common.Time(time.Now()),
		"type":       "libbeat",
		"message":    "Test message from libbeat",
	}}

	batch := encodeBatch(client, outest.NewBatch(event, event, event))
	err = client.Publish(context.Background(), batch)
	assert.NoError(t, err)
	assert.Equal(t, 2, requestCount)
}

func TestBulkEncodeEvents(t *testing.T) {
	cases := map[string]struct {
		version string
		docType string
		config  mapstr.M
		events  []mapstr.M
	}{
		"6.x": {
			version: "6.8.0",
			docType: "doc",
			config:  mapstr.M{},
			events:  []mapstr.M{{"message": "test"}},
		},
		"latest": {
			version: version.GetDefaultVersion(),
			docType: "",
			config:  mapstr.M{},
			events:  []mapstr.M{{"message": "test"}},
		},
	}

	for name, test := range cases {
		test := test
		t.Run(name, func(t *testing.T) {
			logger := logptest.NewTestingLogger(t, "")
			cfg := c.MustNewConfigFrom(test.config)
			info := beat.Info{
				IndexPrefix: "test",
				Version:     test.version,
				Logger:      logger,
			}

			im, err := idxmgmt.DefaultSupport(info, c.NewConfig())
			require.NoError(t, err)

			index, pipeline, err := buildSelectors(im, info, cfg)
			require.NoError(t, err)

			client, err := NewClient(
				clientSettings{
					observer:         outputs.NewNilObserver(),
					indexSelector:    index,
					pipelineSelector: pipeline,
				},
				nil,
				logger,
			)
			assert.NoError(t, err)

			events := make([]publisher.Event, len(test.events))
			for i, fields := range test.events {
				events[i] = publisher.Event{
					Content: beat.Event{
						Timestamp: time.Now(),
						Fields:    fields,
					},
				}
			}
			encodeEvents(client, events)

			encoded, bulkItems := client.bulkEncodePublishRequest(*libversion.MustNew(test.version), events)
			assert.Equal(t, len(events), len(encoded), "all events should have been encoded")
			assert.Equal(t, 2*len(events), len(bulkItems), "incomplete bulk")

			// check meta-data for each event
			for i := 0; i < len(bulkItems); i += 2 {
				var meta eslegclient.BulkMeta
				switch v := bulkItems[i].(type) {
				case eslegclient.BulkCreateAction:
					meta = v.Create
				case eslegclient.BulkIndexAction:
					meta = v.Index
				default:
					panic("unknown type")
				}

				assert.NotEqual(t, "", meta.Index)
				assert.Equal(t, test.docType, meta.DocType)
			}

			// TODO: customer per test case validation
		})
	}
}

func TestBulkEncodeEventsWithOpType(t *testing.T) {
	cases := []mapstr.M{
		{"_id": "111", "op_type": e.OpTypeIndex, "message": "test 1", "bulkIndex": 0},
		{"_id": "112", "message": "test 2", "bulkIndex": 2},
		{"_id": "", "op_type": e.OpTypeDelete, "message": "test 6", "bulkIndex": -1}, // this won't get encoded due to missing _id
		{"_id": "", "message": "test 3", "bulkIndex": 4},
		{"_id": "114", "op_type": e.OpTypeDelete, "message": "test 4", "bulkIndex": 6},
		{"_id": "115", "op_type": e.OpTypeIndex, "message": "test 5", "bulkIndex": 7},
	}

	cfg := c.MustNewConfigFrom(mapstr.M{})
	logger := logptest.NewTestingLogger(t, "")
	info := beat.Info{
		IndexPrefix: "test",
		Version:     version.GetDefaultVersion(),
		Logger:      logger,
	}

	im, err := idxmgmt.DefaultSupport(info, c.NewConfig())
	require.NoError(t, err)

	index, pipeline, err := buildSelectors(im, info, cfg)
	require.NoError(t, err)

	client, _ := NewClient(
		clientSettings{
			observer:         outputs.NewNilObserver(),
			indexSelector:    index,
			pipelineSelector: pipeline,
		},
		nil,
		logger,
	)

	events := make([]publisher.Event, len(cases))
	for i, fields := range cases {
		meta := mapstr.M{
			"_id": fields["_id"],
		}
		if opType, exists := fields["op_type"]; exists {
			meta[e.FieldMetaOpType] = opType
		}

		events[i] = publisher.Event{
			Content: beat.Event{
				Meta: meta,
				Fields: mapstr.M{
					"message": fields["message"],
				},
			},
		}
	}
	encodeEvents(client, events)

	encoded, bulkItems := client.bulkEncodePublishRequest(*libversion.MustNew(version.GetDefaultVersion()), events)
	require.Equal(t, len(events)-1, len(encoded), "all events should have been encoded")
	require.Equal(t, 9, len(bulkItems), "incomplete bulk")

	for i := 0; i < len(cases); i++ {
		bulkEventIndex, _ := cases[i]["bulkIndex"].(int)
		if bulkEventIndex == -1 {
			continue
		}
		caseOpType := cases[i]["op_type"]
		caseMessage := cases[i]["message"].(string)
		switch bulkItems[bulkEventIndex].(type) {
		case eslegclient.BulkCreateAction:
			validOpTypes := []interface{}{e.OpTypeCreate, nil}
			require.Contains(t, validOpTypes, caseOpType, caseMessage)
		case eslegclient.BulkIndexAction:
			require.Equal(t, e.OpTypeIndex, caseOpType, caseMessage)
		case eslegclient.BulkDeleteAction:
			require.Equal(t, e.OpTypeDelete, caseOpType, caseMessage)
		default:
			require.FailNow(t, "unknown type")
		}
	}

}

func TestClientWithAPIKey(t *testing.T) {
	var headers http.Header

	// Start a mock HTTP server, save request headers
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header
	}))
	defer ts.Close()

	logger := logptest.NewTestingLogger(t, "")

	client, err := NewClient(clientSettings{
		observer: outputs.NewNilObserver(),
		connection: eslegclient.ConnectionSettings{
			URL:    ts.URL,
			APIKey: "hyokHG4BfWk5viKZ172X:o45JUkyuS--yiSAuuxl8Uw",
		},
	}, nil, logger)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	// This connection will fail since the server doesn't return a valid
	// response. This is fine since we're just testing the headers in the
	// original client request.
	//nolint:errcheck // connection doesn't need to succeed
	client.Connect(ctx)
	assert.Equal(t, "ApiKey aHlva0hHNEJmV2s1dmlLWjE3Mlg6bzQ1SlVreXVTLS15aVNBdXV4bDhVdw==", headers.Get("Authorization"))
}

func TestBulkRequestHasFilterPath(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	makePublishTestClient := func(t *testing.T, url string, configParams map[string]string) *Client {
		client, err := NewClient(
			clientSettings{
				observer: outputs.NewNilObserver(),
				connection: eslegclient.ConnectionSettings{
					URL:        url,
					Parameters: configParams,
				},
				indexSelector: testIndexSelector{},
			},
			nil,
			logger,
		)
		require.NoError(t, err)
		return client
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	event1 := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 1}}}

	const filterPathKey = "filter_path"
	const filterPathValue = "errors,items.*.error,items.*.status"
	t.Run("Single event with response filtering", func(t *testing.T) {
		var reqParams url.Values
		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if strings.ContainsAny("_bulk", r.URL.Path) {
				reqParams = r.URL.Query()
				response := []byte(`{"took":85,"errors":false,"items":[{"index":{"status":200}}]}`)
				_, _ = w.Write(response)
			}
			if strings.Contains("/", r.URL.Path) {
				response := []byte(`{}`)
				_, _ = w.Write(response)
			}
		}))
		defer esMock.Close()
		client := makePublishTestClient(t, esMock.URL, nil)

		batch := encodeBatch(client, &batchMock{events: []publisher.Event{event1}})
		result := client.doBulkRequest(ctx, batch)
		require.NoError(t, result.connErr)
		// Only param should be the standard filter path
		require.Equal(t, len(reqParams), 1, "Only bulk request param should be standard filter path")
		require.Equal(t, filterPathValue, reqParams.Get(filterPathKey), "Bulk request should include standard filter path")
	})
	t.Run("Single event with response filtering and preconfigured client params", func(t *testing.T) {
		var configParams = map[string]string{
			"hardcoded": "yes",
		}
		var reqParams url.Values

		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if strings.ContainsAny("_bulk", r.URL.Path) {
				reqParams = r.URL.Query()
				response := []byte(`{"took":85,"errors":false,"items":[{"index":{"status":200}}]}`)
				_, _ = w.Write(response)
			}
			if strings.Contains("/", r.URL.Path) {
				response := []byte(`{}`)
				_, _ = w.Write(response)
			}
		}))
		defer esMock.Close()
		client := makePublishTestClient(t, esMock.URL, configParams)

		batch := encodeBatch(client, &batchMock{events: []publisher.Event{event1}})
		result := client.doBulkRequest(ctx, batch)
		require.NoError(t, result.connErr)
		require.Equal(t, len(reqParams), 2, "Bulk request should include configured parameter and standard filter path")
		require.Equal(t, filterPathValue, reqParams.Get(filterPathKey), "Bulk request should include standard filter path")
	})
}

func TestSetDeadLetter(t *testing.T) {
	dead_letter_index := "dead_index"
	e := &encodedEvent{
		index: "original_index",
	}
	errType := 123
	errStr := "test error string"
	e.setDeadLetter(dead_letter_index, errType, errStr)

	assert.True(t, e.deadLetter, "setDeadLetter should set the event's deadLetter flag")
	assert.Equal(t, dead_letter_index, e.index, "setDeadLetter should overwrite the event's original index")

	var errFields struct {
		ErrType    int    `json:"error.type"`
		ErrMessage string `json:"error.message"`
	}
	err := json.NewDecoder(bytes.NewReader(e.encoding)).Decode(&errFields)
	require.NoError(t, err, "json decoding of encoded event should succeed")
	assert.Equal(t, errType, errFields.ErrType, "encoded error.type should match value in setDeadLetter")
	assert.Equal(t, errStr, errFields.ErrMessage, "encoded error.message should match value in setDeadLetter")
}
