package ec2_test

import (
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/ec2"
	"github.com/crowdmob/goamz/testutil"
	"launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) {
	gocheck.TestingT(t)
}

var _ = gocheck.Suite(&S{})

type S struct {
	ec2 *ec2.EC2
}

var testServer = testutil.NewHTTPServer()

func (s *S) SetUpSuite(c *gocheck.C) {
	testServer.Start()
	auth := aws.Auth{AccessKey: "abc", SecretKey: "123"}
	s.ec2 = ec2.New(auth, aws.Region{EC2Endpoint: testServer.URL})
}

func (s *S) TearDownTest(c *gocheck.C) {
	testServer.Flush()
}

func (s *S) TestRunInstancesErrorDump(c *gocheck.C) {
	testServer.Response(400, nil, ErrorDump)

	options := ec2.RunInstancesOptions{
		ImageId:      "ami-a6f504cf", // Ubuntu Maverick, i386, instance store
		InstanceType: "t1.micro",     // Doesn't work with micro, results in 400.
	}

	msg := `AMIs with an instance-store root device are not supported for the instance type 't1\.micro'\.`

	resp, err := s.ec2.RunInstances(&options)

	testServer.WaitRequest()

	c.Assert(resp, gocheck.IsNil)
	c.Assert(err, gocheck.ErrorMatches, msg+` \(UnsupportedOperation\)`)

	ec2err, ok := err.(*ec2.Error)
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(ec2err.StatusCode, gocheck.Equals, 400)
	c.Assert(ec2err.Code, gocheck.Equals, "UnsupportedOperation")
	c.Assert(ec2err.Message, gocheck.Matches, msg)
	c.Assert(ec2err.RequestId, gocheck.Equals, "0503f4e9-bbd6-483c-b54f-c4ae9f3b30f4")
}

func (s *S) TestRunInstancesErrorWithoutXML(c *gocheck.C) {
	testServer.Response(500, nil, "")
	options := ec2.RunInstancesOptions{ImageId: "image-id"}

	resp, err := s.ec2.RunInstances(&options)

	testServer.WaitRequest()

	c.Assert(resp, gocheck.IsNil)
	c.Assert(err, gocheck.ErrorMatches, "500 Internal Server Error")

	ec2err, ok := err.(*ec2.Error)
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(ec2err.StatusCode, gocheck.Equals, 500)
	c.Assert(ec2err.Code, gocheck.Equals, "")
	c.Assert(ec2err.Message, gocheck.Equals, "500 Internal Server Error")
	c.Assert(ec2err.RequestId, gocheck.Equals, "")
}

func (s *S) TestRunInstancesExample(c *gocheck.C) {
	testServer.Response(200, nil, RunInstancesExample)

	options := ec2.RunInstancesOptions{
		KeyName:               "my-keys",
		ImageId:               "image-id",
		InstanceType:          "inst-type",
		SecurityGroups:        []ec2.SecurityGroup{{Name: "g1"}, {Id: "g2"}, {Name: "g3"}, {Id: "g4"}},
		UserData:              []byte("1234"),
		KernelId:              "kernel-id",
		RamdiskId:             "ramdisk-id",
		AvailZone:             "zone",
		PlacementGroupName:    "group",
		Monitoring:            true,
		SubnetId:              "subnet-id",
		DisableAPITermination: true,
		ShutdownBehavior:      "terminate",
		PrivateIPAddress:      "10.0.0.25",
	}
	resp, err := s.ec2.RunInstances(&options)

	req := testServer.WaitRequest()
	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"RunInstances"})
	c.Assert(req.Form["ImageId"], gocheck.DeepEquals, []string{"image-id"})
	c.Assert(req.Form["MinCount"], gocheck.DeepEquals, []string{"1"})
	c.Assert(req.Form["MaxCount"], gocheck.DeepEquals, []string{"1"})
	c.Assert(req.Form["KeyName"], gocheck.DeepEquals, []string{"my-keys"})
	c.Assert(req.Form["InstanceType"], gocheck.DeepEquals, []string{"inst-type"})
	c.Assert(req.Form["SecurityGroup.1"], gocheck.DeepEquals, []string{"g1"})
	c.Assert(req.Form["SecurityGroup.2"], gocheck.DeepEquals, []string{"g3"})
	c.Assert(req.Form["SecurityGroupId.1"], gocheck.DeepEquals, []string{"g2"})
	c.Assert(req.Form["SecurityGroupId.2"], gocheck.DeepEquals, []string{"g4"})
	c.Assert(req.Form["UserData"], gocheck.DeepEquals, []string{"MTIzNA=="})
	c.Assert(req.Form["KernelId"], gocheck.DeepEquals, []string{"kernel-id"})
	c.Assert(req.Form["RamdiskId"], gocheck.DeepEquals, []string{"ramdisk-id"})
	c.Assert(req.Form["Placement.AvailabilityZone"], gocheck.DeepEquals, []string{"zone"})
	c.Assert(req.Form["Placement.GroupName"], gocheck.DeepEquals, []string{"group"})
	c.Assert(req.Form["Monitoring.Enabled"], gocheck.DeepEquals, []string{"true"})
	c.Assert(req.Form["SubnetId"], gocheck.DeepEquals, []string{"subnet-id"})
	c.Assert(req.Form["DisableApiTermination"], gocheck.DeepEquals, []string{"true"})
	c.Assert(req.Form["InstanceInitiatedShutdownBehavior"], gocheck.DeepEquals, []string{"terminate"})
	c.Assert(req.Form["PrivateIpAddress"], gocheck.DeepEquals, []string{"10.0.0.25"})

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")
	c.Assert(resp.ReservationId, gocheck.Equals, "r-47a5402e")
	c.Assert(resp.OwnerId, gocheck.Equals, "999988887777")
	c.Assert(resp.SecurityGroups, gocheck.DeepEquals, []ec2.SecurityGroup{{Name: "default", Id: "sg-67ad940e"}})
	c.Assert(resp.Instances, gocheck.HasLen, 3)

	i0 := resp.Instances[0]
	c.Assert(i0.InstanceId, gocheck.Equals, "i-2ba64342")
	c.Assert(i0.InstanceType, gocheck.Equals, "m1.small")
	c.Assert(i0.ImageId, gocheck.Equals, "ami-60a54009")
	c.Assert(i0.Monitoring, gocheck.Equals, "enabled")
	c.Assert(i0.KeyName, gocheck.Equals, "example-key-name")
	c.Assert(i0.AMILaunchIndex, gocheck.Equals, 0)
	c.Assert(i0.VirtType, gocheck.Equals, "paravirtual")
	c.Assert(i0.Hypervisor, gocheck.Equals, "xen")

	i1 := resp.Instances[1]
	c.Assert(i1.InstanceId, gocheck.Equals, "i-2bc64242")
	c.Assert(i1.InstanceType, gocheck.Equals, "m1.small")
	c.Assert(i1.ImageId, gocheck.Equals, "ami-60a54009")
	c.Assert(i1.Monitoring, gocheck.Equals, "enabled")
	c.Assert(i1.KeyName, gocheck.Equals, "example-key-name")
	c.Assert(i1.AMILaunchIndex, gocheck.Equals, 1)
	c.Assert(i1.VirtType, gocheck.Equals, "paravirtual")
	c.Assert(i1.Hypervisor, gocheck.Equals, "xen")

	i2 := resp.Instances[2]
	c.Assert(i2.InstanceId, gocheck.Equals, "i-2be64332")
	c.Assert(i2.InstanceType, gocheck.Equals, "m1.small")
	c.Assert(i2.ImageId, gocheck.Equals, "ami-60a54009")
	c.Assert(i2.Monitoring, gocheck.Equals, "enabled")
	c.Assert(i2.KeyName, gocheck.Equals, "example-key-name")
	c.Assert(i2.AMILaunchIndex, gocheck.Equals, 2)
	c.Assert(i2.VirtType, gocheck.Equals, "paravirtual")
	c.Assert(i2.Hypervisor, gocheck.Equals, "xen")
}

func (s *S) TestTerminateInstancesExample(c *gocheck.C) {
	testServer.Response(200, nil, TerminateInstancesExample)

	resp, err := s.ec2.TerminateInstances([]string{"i-1", "i-2"})

	req := testServer.WaitRequest()
	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"TerminateInstances"})
	c.Assert(req.Form["InstanceId.1"], gocheck.DeepEquals, []string{"i-1"})
	c.Assert(req.Form["InstanceId.2"], gocheck.DeepEquals, []string{"i-2"})
	c.Assert(req.Form["UserData"], gocheck.IsNil)
	c.Assert(req.Form["KernelId"], gocheck.IsNil)
	c.Assert(req.Form["RamdiskId"], gocheck.IsNil)
	c.Assert(req.Form["Placement.AvailabilityZone"], gocheck.IsNil)
	c.Assert(req.Form["Placement.GroupName"], gocheck.IsNil)
	c.Assert(req.Form["Monitoring.Enabled"], gocheck.IsNil)
	c.Assert(req.Form["SubnetId"], gocheck.IsNil)
	c.Assert(req.Form["DisableApiTermination"], gocheck.IsNil)
	c.Assert(req.Form["InstanceInitiatedShutdownBehavior"], gocheck.IsNil)
	c.Assert(req.Form["PrivateIpAddress"], gocheck.IsNil)

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")
	c.Assert(resp.StateChanges, gocheck.HasLen, 1)
	c.Assert(resp.StateChanges[0].InstanceId, gocheck.Equals, "i-3ea74257")
	c.Assert(resp.StateChanges[0].CurrentState.Code, gocheck.Equals, 32)
	c.Assert(resp.StateChanges[0].CurrentState.Name, gocheck.Equals, "shutting-down")
	c.Assert(resp.StateChanges[0].PreviousState.Code, gocheck.Equals, 16)
	c.Assert(resp.StateChanges[0].PreviousState.Name, gocheck.Equals, "running")
}

func (s *S) TestDescribeInstancesExample1(c *gocheck.C) {
	testServer.Response(200, nil, DescribeInstancesExample1)

	filter := ec2.NewFilter()
	filter.Add("key1", "value1")
	filter.Add("key2", "value2", "value3")

	resp, err := s.ec2.Instances([]string{"i-1", "i-2"}, nil)

	req := testServer.WaitRequest()
	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"DescribeInstances"})
	c.Assert(req.Form["InstanceId.1"], gocheck.DeepEquals, []string{"i-1"})
	c.Assert(req.Form["InstanceId.2"], gocheck.DeepEquals, []string{"i-2"})

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "98e3c9a4-848c-4d6d-8e8a-b1bdEXAMPLE")
	c.Assert(resp.Reservations, gocheck.HasLen, 2)

	r0 := resp.Reservations[0]
	c.Assert(r0.ReservationId, gocheck.Equals, "r-b27e30d9")
	c.Assert(r0.OwnerId, gocheck.Equals, "999988887777")
	c.Assert(r0.RequesterId, gocheck.Equals, "854251627541")
	c.Assert(r0.SecurityGroups, gocheck.DeepEquals, []ec2.SecurityGroup{{Name: "default", Id: "sg-67ad940e"}})
	c.Assert(r0.Instances, gocheck.HasLen, 1)

	r0i := r0.Instances[0]
	c.Assert(r0i.InstanceId, gocheck.Equals, "i-c5cd56af")
	c.Assert(r0i.PrivateDNSName, gocheck.Equals, "domU-12-31-39-10-56-34.compute-1.internal")
	c.Assert(r0i.DNSName, gocheck.Equals, "ec2-174-129-165-232.compute-1.amazonaws.com")
	c.Assert(r0i.AvailZone, gocheck.Equals, "us-east-1b")
	c.Assert(r0i.IPAddress, gocheck.Equals, "174.129.165.232")
	c.Assert(r0i.PrivateIPAddress, gocheck.Equals, "10.198.85.190")
}

func (s *S) TestDescribeInstancesExample2(c *gocheck.C) {
	testServer.Response(200, nil, DescribeInstancesExample2)

	filter := ec2.NewFilter()
	filter.Add("key1", "value1")
	filter.Add("key2", "value2", "value3")

	resp, err := s.ec2.Instances([]string{"i-1", "i-2"}, filter)

	req := testServer.WaitRequest()
	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"DescribeInstances"})
	c.Assert(req.Form["InstanceId.1"], gocheck.DeepEquals, []string{"i-1"})
	c.Assert(req.Form["InstanceId.2"], gocheck.DeepEquals, []string{"i-2"})
	c.Assert(req.Form["Filter.1.Name"], gocheck.DeepEquals, []string{"key1"})
	c.Assert(req.Form["Filter.1.Value.1"], gocheck.DeepEquals, []string{"value1"})
	c.Assert(req.Form["Filter.1.Value.2"], gocheck.IsNil)
	c.Assert(req.Form["Filter.2.Name"], gocheck.DeepEquals, []string{"key2"})
	c.Assert(req.Form["Filter.2.Value.1"], gocheck.DeepEquals, []string{"value2"})
	c.Assert(req.Form["Filter.2.Value.2"], gocheck.DeepEquals, []string{"value3"})

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")
	c.Assert(resp.Reservations, gocheck.HasLen, 1)

	r0 := resp.Reservations[0]
	r0i := r0.Instances[0]
	c.Assert(r0i.State.Code, gocheck.Equals, 16)
	c.Assert(r0i.State.Name, gocheck.Equals, "running")

	r0t0 := r0i.Tags[0]
	r0t1 := r0i.Tags[1]
	c.Assert(r0t0.Key, gocheck.Equals, "webserver")
	c.Assert(r0t0.Value, gocheck.Equals, "")
	c.Assert(r0t1.Key, gocheck.Equals, "stack")
	c.Assert(r0t1.Value, gocheck.Equals, "Production")
}

func (s *S) TestDescribeImagesExample(c *gocheck.C) {
	testServer.Response(200, nil, DescribeImagesExample)

	filter := ec2.NewFilter()
	filter.Add("key1", "value1")
	filter.Add("key2", "value2", "value3")

	resp, err := s.ec2.Images([]string{"ami-1", "ami-2"}, filter)

	req := testServer.WaitRequest()
	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"DescribeImages"})
	c.Assert(req.Form["ImageId.1"], gocheck.DeepEquals, []string{"ami-1"})
	c.Assert(req.Form["ImageId.2"], gocheck.DeepEquals, []string{"ami-2"})
	c.Assert(req.Form["Filter.1.Name"], gocheck.DeepEquals, []string{"key1"})
	c.Assert(req.Form["Filter.1.Value.1"], gocheck.DeepEquals, []string{"value1"})
	c.Assert(req.Form["Filter.1.Value.2"], gocheck.IsNil)
	c.Assert(req.Form["Filter.2.Name"], gocheck.DeepEquals, []string{"key2"})
	c.Assert(req.Form["Filter.2.Value.1"], gocheck.DeepEquals, []string{"value2"})
	c.Assert(req.Form["Filter.2.Value.2"], gocheck.DeepEquals, []string{"value3"})

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "4a4a27a2-2e7c-475d-b35b-ca822EXAMPLE")
	c.Assert(resp.Images, gocheck.HasLen, 1)

	i0 := resp.Images[0]
	c.Assert(i0.Id, gocheck.Equals, "ami-a2469acf")
	c.Assert(i0.Type, gocheck.Equals, "machine")
	c.Assert(i0.Name, gocheck.Equals, "example-marketplace-amzn-ami.1")
	c.Assert(i0.Description, gocheck.Equals, "Amazon Linux AMI i386 EBS")
	c.Assert(i0.Location, gocheck.Equals, "aws-marketplace/example-marketplace-amzn-ami.1")
	c.Assert(i0.State, gocheck.Equals, "available")
	c.Assert(i0.Public, gocheck.Equals, true)
	c.Assert(i0.OwnerId, gocheck.Equals, "123456789999")
	c.Assert(i0.OwnerAlias, gocheck.Equals, "aws-marketplace")
	c.Assert(i0.Architecture, gocheck.Equals, "i386")
	c.Assert(i0.KernelId, gocheck.Equals, "aki-805ea7e9")
	c.Assert(i0.RootDeviceType, gocheck.Equals, "ebs")
	c.Assert(i0.RootDeviceName, gocheck.Equals, "/dev/sda1")
	c.Assert(i0.VirtualizationType, gocheck.Equals, "paravirtual")
	c.Assert(i0.Hypervisor, gocheck.Equals, "xen")

	c.Assert(i0.BlockDevices, gocheck.HasLen, 1)
	c.Assert(i0.BlockDevices[0].DeviceName, gocheck.Equals, "/dev/sda1")
	c.Assert(i0.BlockDevices[0].SnapshotId, gocheck.Equals, "snap-787e9403")
	c.Assert(i0.BlockDevices[0].VolumeSize, gocheck.Equals, int64(8))
	c.Assert(i0.BlockDevices[0].DeleteOnTermination, gocheck.Equals, true)
}

func (s *S) TestCreateSnapshotExample(c *gocheck.C) {
	testServer.Response(200, nil, CreateSnapshotExample)

	resp, err := s.ec2.CreateSnapshot("vol-4d826724", "Daily Backup")

	req := testServer.WaitRequest()
	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"CreateSnapshot"})
	c.Assert(req.Form["VolumeId"], gocheck.DeepEquals, []string{"vol-4d826724"})
	c.Assert(req.Form["Description"], gocheck.DeepEquals, []string{"Daily Backup"})

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")
	c.Assert(resp.Snapshot.Id, gocheck.Equals, "snap-78a54011")
	c.Assert(resp.Snapshot.VolumeId, gocheck.Equals, "vol-4d826724")
	c.Assert(resp.Snapshot.Status, gocheck.Equals, "pending")
	c.Assert(resp.Snapshot.StartTime, gocheck.Equals, "2008-05-07T12:51:50.000Z")
	c.Assert(resp.Snapshot.Progress, gocheck.Equals, "60%")
	c.Assert(resp.Snapshot.OwnerId, gocheck.Equals, "111122223333")
	c.Assert(resp.Snapshot.VolumeSize, gocheck.Equals, "10")
	c.Assert(resp.Snapshot.Description, gocheck.Equals, "Daily Backup")
}

func (s *S) TestDeleteSnapshotsExample(c *gocheck.C) {
	testServer.Response(200, nil, DeleteSnapshotExample)

	resp, err := s.ec2.DeleteSnapshots([]string{"snap-78a54011"})

	req := testServer.WaitRequest()
	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"DeleteSnapshot"})
	c.Assert(req.Form["SnapshotId.1"], gocheck.DeepEquals, []string{"snap-78a54011"})

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")
}

func (s *S) TestDescribeSnapshotsExample(c *gocheck.C) {
	testServer.Response(200, nil, DescribeSnapshotsExample)

	filter := ec2.NewFilter()
	filter.Add("key1", "value1")
	filter.Add("key2", "value2", "value3")

	resp, err := s.ec2.Snapshots([]string{"snap-1", "snap-2"}, filter)

	req := testServer.WaitRequest()
	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"DescribeSnapshots"})
	c.Assert(req.Form["SnapshotId.1"], gocheck.DeepEquals, []string{"snap-1"})
	c.Assert(req.Form["SnapshotId.2"], gocheck.DeepEquals, []string{"snap-2"})
	c.Assert(req.Form["Filter.1.Name"], gocheck.DeepEquals, []string{"key1"})
	c.Assert(req.Form["Filter.1.Value.1"], gocheck.DeepEquals, []string{"value1"})
	c.Assert(req.Form["Filter.1.Value.2"], gocheck.IsNil)
	c.Assert(req.Form["Filter.2.Name"], gocheck.DeepEquals, []string{"key2"})
	c.Assert(req.Form["Filter.2.Value.1"], gocheck.DeepEquals, []string{"value2"})
	c.Assert(req.Form["Filter.2.Value.2"], gocheck.DeepEquals, []string{"value3"})

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")
	c.Assert(resp.Snapshots, gocheck.HasLen, 1)

	s0 := resp.Snapshots[0]
	c.Assert(s0.Id, gocheck.Equals, "snap-1a2b3c4d")
	c.Assert(s0.VolumeId, gocheck.Equals, "vol-8875daef")
	c.Assert(s0.VolumeSize, gocheck.Equals, "15")
	c.Assert(s0.Status, gocheck.Equals, "pending")
	c.Assert(s0.StartTime, gocheck.Equals, "2010-07-29T04:12:01.000Z")
	c.Assert(s0.Progress, gocheck.Equals, "30%")
	c.Assert(s0.OwnerId, gocheck.Equals, "111122223333")
	c.Assert(s0.Description, gocheck.Equals, "Daily Backup")

	c.Assert(s0.Tags, gocheck.HasLen, 1)
	c.Assert(s0.Tags[0].Key, gocheck.Equals, "Purpose")
	c.Assert(s0.Tags[0].Value, gocheck.Equals, "demo_db_14_backup")
}

func (s *S) TestCreateSecurityGroupExample(c *gocheck.C) {
	testServer.Response(200, nil, CreateSecurityGroupExample)

	resp, err := s.ec2.CreateSecurityGroup("websrv", "Web Servers")

	req := testServer.WaitRequest()
	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"CreateSecurityGroup"})
	c.Assert(req.Form["GroupName"], gocheck.DeepEquals, []string{"websrv"})
	c.Assert(req.Form["GroupDescription"], gocheck.DeepEquals, []string{"Web Servers"})

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")
	c.Assert(resp.Name, gocheck.Equals, "websrv")
	c.Assert(resp.Id, gocheck.Equals, "sg-67ad940e")
}

func (s *S) TestDescribeSecurityGroupsExample(c *gocheck.C) {
	testServer.Response(200, nil, DescribeSecurityGroupsExample)

	resp, err := s.ec2.SecurityGroups([]ec2.SecurityGroup{{Name: "WebServers"}, {Name: "RangedPortsBySource"}}, nil)

	req := testServer.WaitRequest()
	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"DescribeSecurityGroups"})
	c.Assert(req.Form["GroupName.1"], gocheck.DeepEquals, []string{"WebServers"})
	c.Assert(req.Form["GroupName.2"], gocheck.DeepEquals, []string{"RangedPortsBySource"})

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")
	c.Assert(resp.Groups, gocheck.HasLen, 2)

	g0 := resp.Groups[0]
	c.Assert(g0.OwnerId, gocheck.Equals, "999988887777")
	c.Assert(g0.Name, gocheck.Equals, "WebServers")
	c.Assert(g0.Id, gocheck.Equals, "sg-67ad940e")
	c.Assert(g0.Description, gocheck.Equals, "Web Servers")
	c.Assert(g0.IPPerms, gocheck.HasLen, 1)

	g0ipp := g0.IPPerms[0]
	c.Assert(g0ipp.Protocol, gocheck.Equals, "tcp")
	c.Assert(g0ipp.FromPort, gocheck.Equals, 80)
	c.Assert(g0ipp.ToPort, gocheck.Equals, 80)
	c.Assert(g0ipp.SourceIPs, gocheck.DeepEquals, []string{"0.0.0.0/0"})

	g1 := resp.Groups[1]
	c.Assert(g1.OwnerId, gocheck.Equals, "999988887777")
	c.Assert(g1.Name, gocheck.Equals, "RangedPortsBySource")
	c.Assert(g1.Id, gocheck.Equals, "sg-76abc467")
	c.Assert(g1.Description, gocheck.Equals, "Group A")
	c.Assert(g1.IPPerms, gocheck.HasLen, 1)

	g1ipp := g1.IPPerms[0]
	c.Assert(g1ipp.Protocol, gocheck.Equals, "tcp")
	c.Assert(g1ipp.FromPort, gocheck.Equals, 6000)
	c.Assert(g1ipp.ToPort, gocheck.Equals, 7000)
	c.Assert(g1ipp.SourceIPs, gocheck.IsNil)
}

func (s *S) TestDescribeSecurityGroupsExampleWithFilter(c *gocheck.C) {
	testServer.Response(200, nil, DescribeSecurityGroupsExample)

	filter := ec2.NewFilter()
	filter.Add("ip-permission.protocol", "tcp")
	filter.Add("ip-permission.from-port", "22")
	filter.Add("ip-permission.to-port", "22")
	filter.Add("ip-permission.group-name", "app_server_group", "database_group")

	_, err := s.ec2.SecurityGroups(nil, filter)

	req := testServer.WaitRequest()
	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"DescribeSecurityGroups"})
	c.Assert(req.Form["Filter.1.Name"], gocheck.DeepEquals, []string{"ip-permission.from-port"})
	c.Assert(req.Form["Filter.1.Value.1"], gocheck.DeepEquals, []string{"22"})
	c.Assert(req.Form["Filter.2.Name"], gocheck.DeepEquals, []string{"ip-permission.group-name"})
	c.Assert(req.Form["Filter.2.Value.1"], gocheck.DeepEquals, []string{"app_server_group"})
	c.Assert(req.Form["Filter.2.Value.2"], gocheck.DeepEquals, []string{"database_group"})
	c.Assert(req.Form["Filter.3.Name"], gocheck.DeepEquals, []string{"ip-permission.protocol"})
	c.Assert(req.Form["Filter.3.Value.1"], gocheck.DeepEquals, []string{"tcp"})
	c.Assert(req.Form["Filter.4.Name"], gocheck.DeepEquals, []string{"ip-permission.to-port"})
	c.Assert(req.Form["Filter.4.Value.1"], gocheck.DeepEquals, []string{"22"})

	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestDescribeSecurityGroupsDumpWithGroup(c *gocheck.C) {
	testServer.Response(200, nil, DescribeSecurityGroupsDump)

	resp, err := s.ec2.SecurityGroups(nil, nil)

	req := testServer.WaitRequest()
	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"DescribeSecurityGroups"})
	c.Assert(err, gocheck.IsNil)
	c.Check(resp.Groups, gocheck.HasLen, 1)
	c.Check(resp.Groups[0].IPPerms, gocheck.HasLen, 2)

	ipp0 := resp.Groups[0].IPPerms[0]
	c.Assert(ipp0.SourceIPs, gocheck.IsNil)
	c.Check(ipp0.Protocol, gocheck.Equals, "icmp")
	c.Assert(ipp0.SourceGroups, gocheck.HasLen, 1)
	c.Check(ipp0.SourceGroups[0].OwnerId, gocheck.Equals, "12345")
	c.Check(ipp0.SourceGroups[0].Name, gocheck.Equals, "default")
	c.Check(ipp0.SourceGroups[0].Id, gocheck.Equals, "sg-67ad940e")

	ipp1 := resp.Groups[0].IPPerms[1]
	c.Check(ipp1.Protocol, gocheck.Equals, "tcp")
	c.Assert(ipp0.SourceIPs, gocheck.IsNil)
	c.Assert(ipp0.SourceGroups, gocheck.HasLen, 1)
	c.Check(ipp1.SourceGroups[0].Id, gocheck.Equals, "sg-76abc467")
	c.Check(ipp1.SourceGroups[0].OwnerId, gocheck.Equals, "12345")
	c.Check(ipp1.SourceGroups[0].Name, gocheck.Equals, "other")
}

func (s *S) TestDeleteSecurityGroupExample(c *gocheck.C) {
	testServer.Response(200, nil, DeleteSecurityGroupExample)

	resp, err := s.ec2.DeleteSecurityGroup(ec2.SecurityGroup{Name: "websrv"})
	req := testServer.WaitRequest()

	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"DeleteSecurityGroup"})
	c.Assert(req.Form["GroupName"], gocheck.DeepEquals, []string{"websrv"})
	c.Assert(req.Form["GroupId"], gocheck.IsNil)
	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")
}

func (s *S) TestDeleteSecurityGroupExampleWithId(c *gocheck.C) {
	testServer.Response(200, nil, DeleteSecurityGroupExample)

	// ignore return and error - we're only want to check the parameter handling.
	s.ec2.DeleteSecurityGroup(ec2.SecurityGroup{Id: "sg-67ad940e", Name: "ignored"})
	req := testServer.WaitRequest()

	c.Assert(req.Form["GroupName"], gocheck.IsNil)
	c.Assert(req.Form["GroupId"], gocheck.DeepEquals, []string{"sg-67ad940e"})
}

func (s *S) TestAuthorizeSecurityGroupExample1(c *gocheck.C) {
	testServer.Response(200, nil, AuthorizeSecurityGroupIngressExample)

	perms := []ec2.IPPerm{{
		Protocol:  "tcp",
		FromPort:  80,
		ToPort:    80,
		SourceIPs: []string{"205.192.0.0/16", "205.159.0.0/16"},
	}}
	resp, err := s.ec2.AuthorizeSecurityGroup(ec2.SecurityGroup{Name: "websrv"}, perms)

	req := testServer.WaitRequest()

	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"AuthorizeSecurityGroupIngress"})
	c.Assert(req.Form["GroupName"], gocheck.DeepEquals, []string{"websrv"})
	c.Assert(req.Form["IpPermissions.1.IpProtocol"], gocheck.DeepEquals, []string{"tcp"})
	c.Assert(req.Form["IpPermissions.1.FromPort"], gocheck.DeepEquals, []string{"80"})
	c.Assert(req.Form["IpPermissions.1.ToPort"], gocheck.DeepEquals, []string{"80"})
	c.Assert(req.Form["IpPermissions.1.IpRanges.1.CidrIp"], gocheck.DeepEquals, []string{"205.192.0.0/16"})
	c.Assert(req.Form["IpPermissions.1.IpRanges.2.CidrIp"], gocheck.DeepEquals, []string{"205.159.0.0/16"})

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")
}

func (s *S) TestAuthorizeSecurityGroupExample1WithId(c *gocheck.C) {
	testServer.Response(200, nil, AuthorizeSecurityGroupIngressExample)

	perms := []ec2.IPPerm{{
		Protocol:  "tcp",
		FromPort:  80,
		ToPort:    80,
		SourceIPs: []string{"205.192.0.0/16", "205.159.0.0/16"},
	}}
	// ignore return and error - we're only want to check the parameter handling.
	s.ec2.AuthorizeSecurityGroup(ec2.SecurityGroup{Id: "sg-67ad940e", Name: "ignored"}, perms)

	req := testServer.WaitRequest()

	c.Assert(req.Form["GroupName"], gocheck.IsNil)
	c.Assert(req.Form["GroupId"], gocheck.DeepEquals, []string{"sg-67ad940e"})
}

func (s *S) TestAuthorizeSecurityGroupExample2(c *gocheck.C) {
	testServer.Response(200, nil, AuthorizeSecurityGroupIngressExample)

	perms := []ec2.IPPerm{{
		Protocol: "tcp",
		FromPort: 80,
		ToPort:   81,
		SourceGroups: []ec2.UserSecurityGroup{
			{OwnerId: "999988887777", Name: "OtherAccountGroup"},
			{Id: "sg-67ad940e"},
		},
	}}
	resp, err := s.ec2.AuthorizeSecurityGroup(ec2.SecurityGroup{Name: "websrv"}, perms)

	req := testServer.WaitRequest()

	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"AuthorizeSecurityGroupIngress"})
	c.Assert(req.Form["GroupName"], gocheck.DeepEquals, []string{"websrv"})
	c.Assert(req.Form["IpPermissions.1.IpProtocol"], gocheck.DeepEquals, []string{"tcp"})
	c.Assert(req.Form["IpPermissions.1.FromPort"], gocheck.DeepEquals, []string{"80"})
	c.Assert(req.Form["IpPermissions.1.ToPort"], gocheck.DeepEquals, []string{"81"})
	c.Assert(req.Form["IpPermissions.1.Groups.1.UserId"], gocheck.DeepEquals, []string{"999988887777"})
	c.Assert(req.Form["IpPermissions.1.Groups.1.GroupName"], gocheck.DeepEquals, []string{"OtherAccountGroup"})
	c.Assert(req.Form["IpPermissions.1.Groups.2.UserId"], gocheck.IsNil)
	c.Assert(req.Form["IpPermissions.1.Groups.2.GroupName"], gocheck.IsNil)
	c.Assert(req.Form["IpPermissions.1.Groups.2.GroupId"], gocheck.DeepEquals, []string{"sg-67ad940e"})

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")
}

func (s *S) TestRevokeSecurityGroupExample(c *gocheck.C) {
	// RevokeSecurityGroup is implemented by the same code as AuthorizeSecurityGroup
	// so there's no need to duplicate all the tests.
	testServer.Response(200, nil, RevokeSecurityGroupIngressExample)

	resp, err := s.ec2.RevokeSecurityGroup(ec2.SecurityGroup{Name: "websrv"}, nil)

	req := testServer.WaitRequest()

	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"RevokeSecurityGroupIngress"})
	c.Assert(req.Form["GroupName"], gocheck.DeepEquals, []string{"websrv"})
	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")
}

func (s *S) TestCreateTags(c *gocheck.C) {
	testServer.Response(200, nil, CreateTagsExample)

	resp, err := s.ec2.CreateTags([]string{"ami-1a2b3c4d", "i-7f4d3a2b"}, []ec2.Tag{{"webserver", ""}, {"stack", "Production"}})

	req := testServer.WaitRequest()
	c.Assert(req.Form["ResourceId.1"], gocheck.DeepEquals, []string{"ami-1a2b3c4d"})
	c.Assert(req.Form["ResourceId.2"], gocheck.DeepEquals, []string{"i-7f4d3a2b"})
	c.Assert(req.Form["Tag.1.Key"], gocheck.DeepEquals, []string{"webserver"})
	c.Assert(req.Form["Tag.1.Value"], gocheck.DeepEquals, []string{""})
	c.Assert(req.Form["Tag.2.Key"], gocheck.DeepEquals, []string{"stack"})
	c.Assert(req.Form["Tag.2.Value"], gocheck.DeepEquals, []string{"Production"})

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")
}

func (s *S) TestStartInstances(c *gocheck.C) {
	testServer.Response(200, nil, StartInstancesExample)

	resp, err := s.ec2.StartInstances("i-10a64379")
	req := testServer.WaitRequest()

	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"StartInstances"})
	c.Assert(req.Form["InstanceId.1"], gocheck.DeepEquals, []string{"i-10a64379"})

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")

	s0 := resp.StateChanges[0]
	c.Assert(s0.InstanceId, gocheck.Equals, "i-10a64379")
	c.Assert(s0.CurrentState.Code, gocheck.Equals, 0)
	c.Assert(s0.CurrentState.Name, gocheck.Equals, "pending")
	c.Assert(s0.PreviousState.Code, gocheck.Equals, 80)
	c.Assert(s0.PreviousState.Name, gocheck.Equals, "stopped")
}

func (s *S) TestStopInstances(c *gocheck.C) {
	testServer.Response(200, nil, StopInstancesExample)

	resp, err := s.ec2.StopInstances("i-10a64379")
	req := testServer.WaitRequest()

	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"StopInstances"})
	c.Assert(req.Form["InstanceId.1"], gocheck.DeepEquals, []string{"i-10a64379"})

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")

	s0 := resp.StateChanges[0]
	c.Assert(s0.InstanceId, gocheck.Equals, "i-10a64379")
	c.Assert(s0.CurrentState.Code, gocheck.Equals, 64)
	c.Assert(s0.CurrentState.Name, gocheck.Equals, "stopping")
	c.Assert(s0.PreviousState.Code, gocheck.Equals, 16)
	c.Assert(s0.PreviousState.Name, gocheck.Equals, "running")
}

func (s *S) TestRebootInstances(c *gocheck.C) {
	testServer.Response(200, nil, RebootInstancesExample)

	resp, err := s.ec2.RebootInstances("i-10a64379")
	req := testServer.WaitRequest()

	c.Assert(req.Form["Action"], gocheck.DeepEquals, []string{"RebootInstances"})
	c.Assert(req.Form["InstanceId.1"], gocheck.DeepEquals, []string{"i-10a64379"})

	c.Assert(err, gocheck.IsNil)
	c.Assert(resp.RequestId, gocheck.Equals, "59dbff89-35bd-4eac-99ed-be587EXAMPLE")
}

func (s *S) TestSignatureWithEndpointPath(c *gocheck.C) {
	ec2.FakeTime(true)
	defer ec2.FakeTime(false)

	testServer.Response(200, nil, RebootInstancesExample)

	// https://bugs.launchpad.net/goamz/+bug/1022749
	ec2 := ec2.New(s.ec2.Auth, aws.Region{EC2Endpoint: testServer.URL + "/services/Cloud"})

	_, err := ec2.RebootInstances("i-10a64379")
	c.Assert(err, gocheck.IsNil)

	req := testServer.WaitRequest()
	c.Assert(req.Form["Signature"], gocheck.DeepEquals, []string{"klxs+VwDa1EKHBsxlDYYN58wbP6An+RVdhETv1Fm/os="})
}
