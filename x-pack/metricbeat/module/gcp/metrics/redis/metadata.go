// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	redis "cloud.google.com/go/redis/apiv1"
	"cloud.google.com/go/redis/apiv1/redispb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
	"github.com/elastic/elastic-agent-libs/logp"
)

// NewMetadataService returns the specific Metadata service for a GCP Redis resource
func NewMetadataService(
	ctx context.Context,
	projectID, zone, region string,
	regions []string,
	organizationID, organizationName, projectName string,
	cacheRegistry *gcp.CacheRegistry,
	logger *logp.Logger,
	opt ...option.ClientOption) (gcp.MetadataService, error) {
	mc := &metadataCollector{
		projectID:        projectID,
		projectName:      projectName,
		organizationID:   organizationID,
		organizationName: organizationName,
		zone:             zone,
		region:           region,
		regions:          regions,
		opt:              opt,
		instanceCache:    cacheRegistry.Redis,
		logger:           logger.Named("metrics-redis"),
	}

	// Freshen up the cache, later all we have to do is look up the instance
	err := mc.instanceCache.EnsureFresh(func() (map[string]*redispb.Instance, error) {
		instances := make(map[string]*redispb.Instance)
		r := backoff.NewRetryer(3, time.Second, 30*time.Second)

		err := r.Retry(ctx, func() error {
			var err error
			instances, err = mc.fetchRedisInstances(ctx)
			return err
		})

		return instances, err
	})

	return mc, err
}

// redisMetadata is an object to store data in between the extraction and the writing in the destination (to uncouple
// reading and writing in the same method)
type redisMetadata struct {
	region       string
	instanceID   string
	instanceName string
	machineType  string

	User     map[string]string
	Metadata map[string]string
	Metrics  interface{}
	System   interface{}
}

type metadataCollector struct {
	projectID        string
	projectName      string
	organizationID   string
	organizationName string
	zone             string
	region           string
	regions          []string
	opt              []option.ClientOption
	instanceCache    *gcp.Cache[*redispb.Instance]
	logger           *logp.Logger
}

// Metadata implements googlecloud.MetadataCollector to the known set of labels from a Redis TimeSeries single point of data.
func (s *metadataCollector) Metadata(ctx context.Context, resp *monitoringpb.TimeSeries) (gcp.MetadataCollectorData, error) {
	metadata, err := s.instanceMetadata(ctx, s.instanceID(resp), s.instanceRegion(resp))
	if err != nil {
		return gcp.MetadataCollectorData{}, err
	}

	stackdriverLabels := gcp.NewStackdriverMetadataServiceForTimeSeries(resp, s.organizationID, s.organizationName, s.projectName)

	metadataCollectorData, err := stackdriverLabels.Metadata(ctx, resp)
	if err != nil {
		return gcp.MetadataCollectorData{}, err
	}

	if resp.Resource != nil && resp.Resource.Labels != nil {
		_, _ = metadataCollectorData.ECS.Put(gcp.ECSCloudInstanceIDKey, resp.Resource.Labels[gcp.TimeSeriesResponsePathForECSInstanceID])
	}

	_, _ = metadataCollectorData.ECS.Put(gcp.ECSCloudInstanceNameKey, metadata.instanceName)

	if metadata.machineType != "" {
		lastIndex := strings.LastIndex(metadata.machineType, "/")
		_, _ = metadataCollectorData.ECS.Put(gcp.ECSCloudMachineTypeKey, metadata.machineType[lastIndex+1:])
	}

	metadata.Metrics = metadataCollectorData.Labels[gcp.LabelMetrics]
	metadata.System = metadataCollectorData.Labels[gcp.LabelSystem]

	if metadata.User != nil {
		metadataCollectorData.Labels[gcp.LabelUser] = metadata.User
	}

	return metadataCollectorData, nil
}

// instanceMetadata returns the labels of an instance
func (s *metadataCollector) instanceMetadata(ctx context.Context, instanceID, region string) (*redisMetadata, error) {
	metadata := &redisMetadata{
		instanceID: instanceID,
		region:     region,
	}

	instance, ok := s.instanceCache.Get(instanceID)
	if !ok {
		s.logger.Warnf("Instance %s not found in redis cache.", instanceID)
		return metadata, nil
	}

	if instance.Name != "" {
		parts := strings.Split(instance.Name, "/")
		metadata.instanceName = parts[len(parts)-1]
	}

	if instance.Labels != nil {
		metadata.User = instance.Labels
	}

	if instance.Tier.String() != "" {
		metadata.machineType = instance.Tier.String()
	}

	return metadata, nil
}

func (s *metadataCollector) instanceID(ts *monitoringpb.TimeSeries) string {
	if ts.Resource != nil && ts.Resource.Labels != nil {
		return ts.Resource.Labels[gcp.TimeSeriesResponsePathForECSInstanceID]
	}

	return ""
}

func (s *metadataCollector) instanceRegion(ts *monitoringpb.TimeSeries) string {
	if ts.Resource != nil && ts.Resource.Labels != nil {
		return ts.Resource.Labels["region"]
	}

	return ""
}

func (s *metadataCollector) fetchRedisInstances(ctx context.Context) (map[string]*redispb.Instance, error) {
	s.logger.Debug("get redis instances with ListInstances API")

	client, err := redis.NewCloudRedisClient(ctx, s.opt...)
	if err != nil {
		s.logger.Errorf("error getting client from redis service: %v", err)
		return nil, err
	}

	defer client.Close()

	// Use locations - (wildcard) to fetch all instances.
	// https://pkg.go.dev/cloud.google.com/go/redis@v1.10.0/apiv1#CloudRedisClient.ListInstances
	it := client.ListInstances(ctx, &redispb.ListInstancesRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", s.projectID),
	})
	fetchedInstances := make(map[string]*redispb.Instance)

	for {
		instance, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			s.logger.Errorf("redis ListInstances error: %v", err)
			return nil, fmt.Errorf("error iterating redis instances: %w", err)
		}

		fetchedInstances[instance.GetName()] = instance
	}

	return fetchedInstances, nil
}
