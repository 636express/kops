/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package model

import (
	"fmt"
	"github.com/golang/glog"
	"k8s.io/kops/pkg/apis/kops"
	"k8s.io/kops/upup/pkg/fi/cloudup/awstasks"
)

func (b *KopsModelContext) SecurityGroupName(role kops.InstanceGroupRole) string {
	switch role {
	case kops.InstanceGroupRoleBastion:
		return "bastion." + b.ClusterName()

	case kops.InstanceGroupRoleNode:
		return "nodes." + b.ClusterName()

	case kops.InstanceGroupRoleMaster:
		return "masters." + b.ClusterName()

	default:
		glog.Fatalf("unknown role: %v", role)
		return ""
	}
}

func (b *KopsModelContext) LinkToSecurityGroup(role kops.InstanceGroupRole) *awstasks.SecurityGroup {
	name := b.SecurityGroupName(role)
	return &awstasks.SecurityGroup{Name: &name}
}

func (b *KopsModelContext) AutoscalingGroupName(ig *kops.InstanceGroup) string {
	switch ig.Spec.Role {
	case kops.InstanceGroupRoleMaster:
		// We need to keep this back-compatible, so we introduce the masters name,
		// though the IG name suffices for uniqueness, and with sensible naming masters
		// should be redundant...
		return ig.ObjectMeta.Name + ".masters." + b.ClusterName()
	case kops.InstanceGroupRoleNode, kops.InstanceGroupRoleBastion:
		return ig.ObjectMeta.Name + "." + b.ClusterName()

	default:
		glog.Fatalf("unknown InstanceGroup Role: %v", ig.Spec.Role)
		return ""
	}
}

func (b *KopsModelContext) LinkToAutoscalingGroup(ig *kops.InstanceGroup) *awstasks.AutoscalingGroup {
	name := b.AutoscalingGroupName(ig)
	return &awstasks.AutoscalingGroup{Name: &name}
}

func (b *KopsModelContext) ELBSecurityGroupName(prefix string) string {
	return prefix + "-elb." + b.ClusterName()
}

func (b *KopsModelContext) LinkToELBSecurityGroup(prefix string) *awstasks.SecurityGroup {
	name := b.ELBSecurityGroupName(prefix)
	return &awstasks.SecurityGroup{Name: &name}
}

func (b *KopsModelContext) ELBName(prefix string) string {
	return prefix + "." + b.ClusterName()
}

func (b *KopsModelContext) LinkToELB(prefix string) *awstasks.LoadBalancer {
	name := b.ELBName(prefix)
	return &awstasks.LoadBalancer{Name: &name}
}

func (b *KopsModelContext) LinkToVPC() *awstasks.VPC {
	name := b.ClusterName()
	return &awstasks.VPC{Name: &name}
}

func (b *KopsModelContext) LinkToDNSZone() *awstasks.DNSZone {
	name := b.NameForDNSZone()
	return &awstasks.DNSZone{Name: &name}
}

func (b *KopsModelContext) NameForDNSZone() string {
	name := b.Cluster.Spec.DNSZone
	return name
}

func (b *KopsModelContext) IAMName(role kops.InstanceGroupRole) string {
	var name string

	switch role {
	case kops.InstanceGroupRoleMaster:
		name = "masters." + b.ClusterName()

	case kops.InstanceGroupRoleBastion:
		name = "bastions." + b.ClusterName()

	case kops.InstanceGroupRoleNode:
		name = "nodes." + b.ClusterName()

	default:
		glog.Fatalf("unknown InstanceGroup Role: %q", role)
	}

	return name
}

func (b *KopsModelContext) LinkToIAMInstanceProfile(ig *kops.InstanceGroup) *awstasks.IAMInstanceProfile {
	name := b.IAMName(ig.Spec.Role)
	return &awstasks.IAMInstanceProfile{Name: &name}
}

// SSHKeyName computes a unique SSH key name, combining the cluster name and the SSH public key fingerprint
func (c *KopsModelContext) SSHKeyName() (string, error) {
	fingerprint, err := awstasks.ComputeOpenSSHKeyFingerprint(string(c.SSHPublicKeys[0]))
	if err != nil {
		return "", err
	}

	name := "kubernetes." + c.Cluster.ObjectMeta.Name + "-" + fingerprint
	return name, nil
}

func (b *KopsModelContext) LinkToSSHKey() (*awstasks.SSHKey, error) {
	sshKeyName, err := b.SSHKeyName()
	if err != nil {
		return nil, err
	}

	return &awstasks.SSHKey{Name: &sshKeyName}, nil
}

func (b *KopsModelContext) LinkToSubnet(z *kops.ClusterSubnetSpec) *awstasks.Subnet {
	name := z.Name + "." + b.ClusterName()

	return &awstasks.Subnet{Name: &name}
}

func (b *KopsModelContext) LinkToPublicSubnetInZone(zoneName string) (*awstasks.Subnet, error) {
	var matches []*kops.ClusterSubnetSpec
	for i := range b.Cluster.Spec.Subnets {
		z := &b.Cluster.Spec.Subnets[i]
		if z.Zone != zoneName {
			continue
		}
		if z.Type != kops.SubnetTypePublic {
			continue
		}
		matches = append(matches, z)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("could not find public subnet in zone: %q", zoneName)
	}
	if len(matches) > 1 {
		// TODO: Support this (arbitrary choice I think, for ELBs)
		return nil, fmt.Errorf("found multiple public subnets in zone: %q", zoneName)
	}

	return b.LinkToSubnet(matches[0]), nil
}

func (b *KopsModelContext) LinkToUtilitySubnetInZone(zoneName string) (*awstasks.Subnet, error) {
	var matches []*kops.ClusterSubnetSpec
	for i := range b.Cluster.Spec.Subnets {
		s := &b.Cluster.Spec.Subnets[i]
		if s.Zone != zoneName {
			continue
		}
		if s.Type != kops.SubnetTypeUtility {
			continue
		}
		matches = append(matches, s)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("could not find utility subnet in zone: %q", zoneName)
	}
	if len(matches) > 1 {
		// TODO: Support this
		return nil, fmt.Errorf("found multiple utility subnets in zone: %q", zoneName)
	}

	return b.LinkToSubnet(matches[0]), nil
}

func (b *KopsModelContext) NamePrivateRouteTableInZone(zoneName string) string {
	return "private-" + zoneName + "." + b.ClusterName()
}

func (b *KopsModelContext) LinkToPrivateRouteTableInZone(zoneName string) *awstasks.RouteTable {
	return &awstasks.RouteTable{Name: s(b.NamePrivateRouteTableInZone(zoneName))}
}
