package aws

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/testing"
	"github.com/stretchr/testify/require"
)

// Vpc is an Amazon Virtual Private Cloud.
type Vpc struct {
	Id      string            // The ID of the VPC
	Name    string            // The name of the VPC
	Subnets []Subnet          // A list of subnets in the VPC
	Tags    map[string]string // The tags associated with the VPC
}

// Subnet is a subnet in an availability zone.
type Subnet struct {
	Id               string            // The ID of the Subnet
	AvailabilityZone string            // The Availability Zone the subnet is in
	DefaultForAz     bool              // If the subnet is default for the Availability Zone
	Tags             map[string]string // The tags associated with the subnet
}

const vpcIDFilterName = "vpc-id"
const resourceTypeFilterName = "resource-id"
const resourceIdFilterName = "resource-type"
const vpcResourceTypeFilterValue = "vpc"
const subnetResourceTypeFilterValue = "subnet"
const isDefaultFilterName = "isDefault"
const isDefaultFilterValue = "true"
const defaultVPCName = "Default"

// GetDefaultVpc fetches information about the default VPC in the given region.
func GetDefaultVpc(t testing.TestingT, region string) *Vpc {
	vpc, err := GetDefaultVpcE(t, region)
	require.NoError(t, err)
	return vpc
}

// GetDefaultVpcE fetches information about the default VPC in the given region.
func GetDefaultVpcE(t testing.TestingT, region string) (*Vpc, error) {
	defaultVpcFilter := ec2.Filter{Name: aws.String(isDefaultFilterName), Values: []*string{aws.String(isDefaultFilterValue)}}
	vpcs, err := GetVpcsE(t, []*ec2.Filter{&defaultVpcFilter}, region)

	numVpcs := len(vpcs)
	if numVpcs != 1 {
		return nil, fmt.Errorf("Expected to find one default VPC in region %s but found %s", region, strconv.Itoa(numVpcs))
	}

	return vpcs[0], err
}

// GetVpcById fetches information about a VPC with given Id in the given region.
func GetVpcById(t testing.TestingT, vpcId string, region string) *Vpc {
	vpc, err := GetVpcByIdE(t, vpcId, region)
	require.NoError(t, err)
	return vpc
}

// GetVpcByIdE fetches information about a VPC with given Id in the given region.
func GetVpcByIdE(t testing.TestingT, vpcId string, region string) (*Vpc, error) {
	vpcIdFilter := ec2.Filter{Name: aws.String(vpcIDFilterName), Values: []*string{&vpcId}}
	vpcs, err := GetVpcsE(t, []*ec2.Filter{&vpcIdFilter}, region)

	numVpcs := len(vpcs)
	if numVpcs != 1 {
		return nil, fmt.Errorf("Expected to find one VPC with ID %s in region %s but found %s", vpcId, region, strconv.Itoa(numVpcs))
	}

	return vpcs[0], err
}

// GetVpcsE fetches informations about VPCs from given regions limited by filters
func GetVpcsE(t testing.TestingT, filters []*ec2.Filter, region string) ([]*Vpc, error) {
	client, err := NewEc2ClientE(t, region)
	if err != nil {
		return nil, err
	}

	vpcs, err := client.DescribeVpcs(&ec2.DescribeVpcsInput{Filters: filters})
	if err != nil {
		return nil, err
	}

	numVpcs := len(vpcs.Vpcs)
	retVal := make([]*Vpc, numVpcs)

	for i, vpc := range vpcs.Vpcs {
		subnets, err := GetSubnetsForVpcE(t, aws.StringValue(vpc.VpcId), region)
		if err != nil {
			return nil, err
		}

		tags, err := GetTagsForVpcE(t, aws.StringValue(vpc.VpcId), region)
		if err != nil {
			return nil, err
		}

		retVal[i] = &Vpc{Id: aws.StringValue(vpc.VpcId), Name: FindVpcName(vpc), Subnets: subnets, Tags: tags}
	}

	return retVal, nil
}

// FindVpcName extracts the VPC name from its tags (if any). Fall back to "Default" if it's the default VPC or empty string
// otherwise.
func FindVpcName(vpc *ec2.Vpc) string {
	for _, tag := range vpc.Tags {
		if *tag.Key == "Name" {
			return *tag.Value
		}
	}

	if *vpc.IsDefault {
		return defaultVPCName
	}

	return ""
}

// GetSubnetsForVpc gets the subnets in the specified VPC.
func GetSubnetsForVpc(t testing.TestingT, vpcID string, region string) []Subnet {
	subnets, err := GetSubnetsForVpcE(t, vpcID, region)
	if err != nil {
		t.Fatal(err)
	}
	return subnets
}

// GetSubnetsForVpcE gets the subnets in the specified VPC.
func GetSubnetsForVpcE(t testing.TestingT, vpcID string, region string) ([]Subnet, error) {
	client, err := NewEc2ClientE(t, region)
	if err != nil {
		return nil, err
	}

	vpcIDFilter := ec2.Filter{Name: aws.String(vpcIDFilterName), Values: []*string{&vpcID}}
	subnetOutput, err := client.DescribeSubnets(&ec2.DescribeSubnetsInput{Filters: []*ec2.Filter{&vpcIDFilter}})
	if err != nil {
		return nil, err
	}

	subnets := []Subnet{}

	for _, ec2Subnet := range subnetOutput.Subnets {
		subnetTags := GetTagsForSubnet(t, *ec2Subnet.SubnetId, region)
		subnet := Subnet{Id: aws.StringValue(ec2Subnet.SubnetId), AvailabilityZone: aws.StringValue(ec2Subnet.AvailabilityZone), DefaultForAz: aws.BoolValue(ec2Subnet.DefaultForAz), Tags: subnetTags}
		subnets = append(subnets, subnet)
	}

	return subnets, nil
}

// GetTagsForVpc gets the tags for the specified VPC.
func GetTagsForVpc(t testing.TestingT, vpcID string, region string) map[string]string {
	tags, err := GetTagsForVpcE(t, vpcID, region)
	require.NoError(t, err)

	return tags
}

// GetTagsForVpcE gets the tags for the specified VPC.
func GetTagsForVpcE(t testing.TestingT, vpcID string, region string) (map[string]string, error) {
	client, err := NewEc2ClientE(t, region)
	require.NoError(t, err)

	vpcResourceTypeFilter := ec2.Filter{Name: aws.String(resourceIdFilterName), Values: []*string{aws.String(vpcResourceTypeFilterValue)}}
	vpcResourceIdFilter := ec2.Filter{Name: aws.String(resourceTypeFilterName), Values: []*string{&vpcID}}
	tagsOutput, err := client.DescribeTags(&ec2.DescribeTagsInput{Filters: []*ec2.Filter{&vpcResourceTypeFilter, &vpcResourceIdFilter}})
	require.NoError(t, err)

	tags := map[string]string{}
	for _, tag := range tagsOutput.Tags {
		tags[aws.StringValue(tag.Key)] = aws.StringValue(tag.Value)
	}

	return tags, nil
}

// GetDefaultSubnetIDsForVpc gets the ids of the subnets that are the default subnet for the AvailabilityZone
func GetDefaultSubnetIDsForVpc(t testing.TestingT, vpc Vpc) []string {
	subnetIDs, err := GetDefaultSubnetIDsForVpcE(t, vpc)
	require.NoError(t, err)
	return subnetIDs
}

// GetDefaultSubnetIDsForVpcE gets the ids of the subnets that are the default subnet for the AvailabilityZone
func GetDefaultSubnetIDsForVpcE(t testing.TestingT, vpc Vpc) ([]string, error) {
	if vpc.Name != defaultVPCName {
		// You cannot create a default subnet in a nondefault VPC
		// https://docs.aws.amazon.com/vpc/latest/userguide/default-vpc.html
		return nil, fmt.Errorf("Only default VPCs have default subnets but VPC with id %s is not default VPC", vpc.Id)
	}
	subnetIDs := []string{}
	numSubnets := len(vpc.Subnets)
	if numSubnets == 0 {
		return nil, fmt.Errorf("Expected to find at least one subnet in vpc with ID %s but found zero", vpc.Id)
	}

	for _, subnet := range vpc.Subnets {
		if subnet.DefaultForAz {
			subnetIDs = append(subnetIDs, subnet.Id)
		}
	}
	return subnetIDs, nil
}

// GetTagsForSubnet gets the tags for the specified subnet.
func GetTagsForSubnet(t testing.TestingT, subnetId string, region string) map[string]string {
	tags, err := GetTagsForSubnetE(t, subnetId, region)
	require.NoError(t, err)

	return tags
}

// GetTagsForSubnetE gets the tags for the specified subnet.
func GetTagsForSubnetE(t testing.TestingT, subnetId string, region string) (map[string]string, error) {
	client, err := NewEc2ClientE(t, region)
	require.NoError(t, err)

	subnetResourceTypeFilter := ec2.Filter{Name: aws.String(resourceIdFilterName), Values: []*string{aws.String(subnetResourceTypeFilterValue)}}
	subnetResourceIdFilter := ec2.Filter{Name: aws.String(resourceTypeFilterName), Values: []*string{&subnetId}}
	tagsOutput, err := client.DescribeTags(&ec2.DescribeTagsInput{Filters: []*ec2.Filter{&subnetResourceTypeFilter, &subnetResourceIdFilter}})
	require.NoError(t, err)

	tags := map[string]string{}
	for _, tag := range tagsOutput.Tags {
		tags[aws.StringValue(tag.Key)] = aws.StringValue(tag.Value)
	}

	return tags, nil
}

// IsPublicSubnet returns True if the subnet identified by the given id in the provided region is public.
func IsPublicSubnet(t testing.TestingT, subnetId string, region string) bool {
	isPublic, err := IsPublicSubnetE(t, subnetId, region)
	require.NoError(t, err)
	return isPublic
}

// IsPublicSubnetE returns True if the subnet identified by the given id in the provided region is public.
func IsPublicSubnetE(t testing.TestingT, subnetId string, region string) (bool, error) {
	subnetIdFilterName := "association.subnet-id"

	subnetIdFilter := ec2.Filter{
		Name:   &subnetIdFilterName,
		Values: []*string{&subnetId},
	}

	client, err := NewEc2ClientE(t, region)
	if err != nil {
		return false, err
	}

	rts, err := client.DescribeRouteTables(&ec2.DescribeRouteTablesInput{Filters: []*ec2.Filter{&subnetIdFilter}})
	if err != nil {
		return false, err
	}

	if len(rts.RouteTables) == 0 {
		// Subnets not explicitly associated with any route table are implicitly associated with the main route table
		rts, err = getImplicitRouteTableForSubnetE(t, subnetId, region)
		if err != nil {
			return false, err
		}
	}

	for _, rt := range rts.RouteTables {
		for _, r := range rt.Routes {
			if strings.HasPrefix(aws.StringValue(r.GatewayId), "igw-") {
				return true, nil
			}
		}
	}

	return false, nil
}

func getImplicitRouteTableForSubnetE(t testing.TestingT, subnetId string, region string) (*ec2.DescribeRouteTablesOutput, error) {
	mainRouteFilterName := "association.main"
	mainRouteFilterValue := "true"
	subnetFilterName := "subnet-id"

	client, err := NewEc2ClientE(t, region)
	if err != nil {
		return nil, err
	}

	subnetFilter := ec2.Filter{
		Name:   &subnetFilterName,
		Values: []*string{&subnetId},
	}
	subnetOutput, err := client.DescribeSubnets(&ec2.DescribeSubnetsInput{Filters: []*ec2.Filter{&subnetFilter}})
	if err != nil {
		return nil, err
	}
	numSubnets := len(subnetOutput.Subnets)
	if numSubnets != 1 {
		return nil, fmt.Errorf("Expected to find one subnet with id %s but found %s", subnetId, strconv.Itoa(numSubnets))
	}

	mainRouteFilter := ec2.Filter{
		Name:   &mainRouteFilterName,
		Values: []*string{&mainRouteFilterValue},
	}
	vpcFilter := ec2.Filter{
		Name:   aws.String(vpcIDFilterName),
		Values: []*string{subnetOutput.Subnets[0].VpcId},
	}
	return client.DescribeRouteTables(&ec2.DescribeRouteTablesInput{Filters: []*ec2.Filter{&mainRouteFilter, &vpcFilter}})
}

// GetRandomPrivateCidrBlock gets a random CIDR block from the range of acceptable private IP addresses per RFC 1918
// (https://tools.ietf.org/html/rfc1918#section-3)
// The routingPrefix refers to the "/28" in 1.2.3.4/28.
// Note that, as written, this function will return a subset of all valid ranges. Since we will probably use this function
// mostly for generating random CIDR ranges for VPCs and Subnets, having comprehensive set coverage is not essential.
func GetRandomPrivateCidrBlock(routingPrefix int) string {

	var o1, o2, o3, o4 int

	switch routingPrefix {
	case 32:
		o1 = random.RandomInt([]int{10, 172, 192})

		switch o1 {
		case 10:
			o2 = random.Random(0, 255)
			o3 = random.Random(0, 255)
			o4 = random.Random(0, 255)
		case 172:
			o2 = random.Random(16, 31)
			o3 = random.Random(0, 255)
			o4 = random.Random(0, 255)
		case 192:
			o2 = 168
			o3 = random.Random(0, 255)
			o4 = random.Random(0, 255)
		}

	case 31, 30, 29, 28, 27, 26, 25:
		fallthrough
	case 24:
		o1 = random.RandomInt([]int{10, 172, 192})

		switch o1 {
		case 10:
			o2 = random.Random(0, 255)
			o3 = random.Random(0, 255)
			o4 = 0
		case 172:
			o2 = 16
			o3 = 0
			o4 = 0
		case 192:
			o2 = 168
			o3 = 0
			o4 = 0
		}
	case 23, 22, 21, 20, 19:
		fallthrough
	case 18:
		o1 = random.RandomInt([]int{10, 172, 192})

		switch o1 {
		case 10:
			o2 = 0
			o3 = 0
			o4 = 0
		case 172:
			o2 = 16
			o3 = 0
			o4 = 0
		case 192:
			o2 = 168
			o3 = 0
			o4 = 0
		}
	}
	return fmt.Sprintf("%d.%d.%d.%d/%d", o1, o2, o3, o4, routingPrefix)
}

// GetFirstTwoOctets gets the first two octets from a CIDR block.
func GetFirstTwoOctets(cidrBlock string) string {
	ipAddr := strings.Split(cidrBlock, "/")[0]
	octets := strings.Split(ipAddr, ".")
	return octets[0] + "." + octets[1]
}
