package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Cache stores the list of instances
type Cache struct {
	instances []*ec2.Instance
	time      time.Time
}

// API is a wrapper around the EC2 API, with caching
type API struct {
	session       *session.Session
	service       *ec2.EC2
	cache         *Cache
	instancesChan chan []*ec2.Instance
	errChan       chan error
}

// NewAPI builds an API struct
func NewAPI() *API {
	sess := session.Must(session.NewSession())
	service := ec2.New(sess)
	cache := &Cache{[]*ec2.Instance{}, time.Unix(0, 0)}
	instancesChan := make(chan []*ec2.Instance)
	errChan := make(chan error)

	return &API{sess, service, cache, instancesChan, errChan}
}

// List returns a list of EC2 instances
func (api *API) List(nameFilter string) ([]*ec2.Instance, error) {
	if api.cache.time.Add(time.Duration(15) * time.Second).After(time.Now()) {
		api.instancesChan <- api.cache.instances
		return api.cache.instances, nil
	}

	params := &ec2.DescribeInstancesInput{}
	if nameFilter != "" {
		params.Filters = []*ec2.Filter{
			{
				Name: aws.String("tag:Name"),
				Values: []*string{
					aws.String(strings.Join([]string{"*", nameFilter, "*"}, "")),
				},
			},
		}
	}

	resp, err := api.service.DescribeInstances(params)
	api.errChan <- err
	if err != nil {
		return nil, err
	}
	if len(resp.Reservations) == 0 {
		return nil, nil
	}

	instances := []*ec2.Instance{}
	for _, r := range resp.Reservations {
		for _, i := range r.Instances {
			if *i.State.Name != "terminated" {
				instances = append(instances, i)
			}
		}
	}
	sort.Slice(instances, func(i, j int) bool {
		return GetName(instances[i]) < GetName(instances[j])
	})

	api.cache.instances = instances
	api.cache.time = time.Now()
	api.instancesChan <- instances

	return instances, nil
}

// ExampleList returns a fake list of instances (used for testing)
func ExampleList() []*ec2.Instance {
	count := 5
	list := []*ec2.Instance{}
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("i-%d", i)
		hostname := fmt.Sprintf("ec2-52-59-245-%d.eu-central-1.compute.amazonaws.com", i)
		instance := &ec2.Instance{
			InstanceId:    &id,
			PublicDnsName: &hostname,
		}
		list = append(list, instance)
	}

	return list
}
