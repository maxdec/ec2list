package main

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
)

// type Instance []struct {
// 	ImageID      string `json:"ImageId"`
// 	InstanceID   string `json:"InstanceId"`
// 	InstanceType string `json:"InstanceType"`
// 	LaunchTime   string `json:"LaunchTime"`
// 	Placement    struct {
// 		AvailabilityZone string `json:"AvailabilityZone"`
// 		GroupName        string `json:"GroupName"`
// 		Tenancy          string `json:"Tenancy"`
// 	} `json:"Placement"`
// 	PrivateDNSName   string `json:"PrivateDnsName"`
// 	PrivateIPAddress string `json:"PrivateIpAddress"`
// 	PublicDNSName    string `json:"PublicDnsName"`
// 	PublicIPAddress  string `json:"PublicIpAddress"`
// 	State            struct {
// 		Code int    `json:"Code"`
// 		Name string `json:"Name"`
// 	} `json:"State"`
// 	Tags []struct {
// 		Key   string `json:"Key"`
// 		Value string `json:"Value"`
// 	} `json:"Tags"`
// 	VpcID string `json:"VpcId"`
// }

// ToRow converts an instance into its row representation
func ToRow(i *ec2.Instance) []string {
	publicDNS := ""
	if i.PublicDnsName != nil {
		publicDNS = strings.Split(*i.PublicDnsName, ".")[0]
	}

	privateDNS := ""
	if i.PrivateDnsName != nil {
		privateDNS = strings.Split(*i.PrivateDnsName, ".")[0]
	}

	name := getTagValue(i, "Purpose")
	worksOnJobs := getTagValue(i, "WorksOnJobs")
	if worksOnJobs == "True" { // backend
		name = name + " (BE)"
	} else if worksOnJobs == "False" { // frontend
		name = name + " (FE)"
	}

	return []string{
		*i.InstanceId,
		name,
		publicDNS,
		privateDNS,
		*i.InstanceType,
		*i.Placement.AvailabilityZone,
		i.LaunchTime.In(time.Local).Format("Jan _2, 2006 at 15:04"),
		*i.State.Name,
	}
}

// Filter filters the list of instances against a query string
func Filter(instances []*ec2.Instance, q string) []*ec2.Instance {
	if q == "" {
		return instances
	}

	filteredInstances := []*ec2.Instance{}
	for _, i := range instances {
		if Matches(i, q) {
			filteredInstances = append(filteredInstances, i)
		}
	}

	return filteredInstances
}

// Matches returns true if some instance fields contain the query string
func Matches(i *ec2.Instance, q string) bool {
	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(*i.InstanceId), q) ||
		strings.Contains(strings.ToLower(*i.PublicDnsName), q) ||
		strings.Contains(strings.ToLower(getTagValue(i, "Name")), q) ||
		strings.Contains(strings.ToLower(*i.PrivateDnsName), q) ||
		strings.Contains(strings.ToLower(*i.State.Name), q) ||
		strings.Contains(strings.ToLower(*i.Placement.AvailabilityZone), q)
}

func getTagValue(i *ec2.Instance, name string) string {
	for _, t := range i.Tags {
		if *t.Key == name {
			return *t.Value
		}
	}

	return ""
}

// GetName gets the name of the instance from the tags
func GetName(i *ec2.Instance) string {
	return getTagValue(i, "Name")
}
