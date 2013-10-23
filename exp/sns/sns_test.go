package sns_test

import (
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/exp/sns"
	"github.com/crowdmob/goamz/testutil"
	"launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) {
	gocheck.TestingT(t)
}

var _ = gocheck.Suite(&S{})

type S struct {
	sns *sns.SNS
}

var testServer = testutil.NewHTTPServer()

func (s *S) SetUpSuite(c *gocheck.C) {
	testServer.Start()
	auth := aws.Auth{AccessKey: "abc", SecretKey: "123"}
	s.sns = sns.New(auth, aws.Region{SNSEndpoint: testServer.URL})
}

func (s *S) TearDownTest(c *gocheck.C) {
	testServer.Flush()
}

func (s *S) TestListTopicsOK(c *gocheck.C) {
	testServer.Response(200, nil, TestListTopicsXmlOK)

	resp, err := s.sns.ListTopics(nil)
	req := testServer.WaitRequest()

	c.Assert(req.Method, gocheck.Equals, "GET")
	c.Assert(req.URL.Path, gocheck.Equals, "/")
	c.Assert(req.Header["Date"], gocheck.Not(gocheck.Equals), "")

	c.Assert(resp.ResponseMetadata.RequestId, gocheck.Equals, "bd10b26c-e30e-11e0-ba29-93c3aca2f103")
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestCreateTopic(c *gocheck.C) {
	testServer.Response(200, nil, TestCreateTopicXmlOK)

	resp, err := s.sns.CreateTopic("My-Topic")
	req := testServer.WaitRequest()

	c.Assert(req.Method, gocheck.Equals, "GET")
	c.Assert(req.URL.Path, gocheck.Equals, "/")
	c.Assert(req.Header["Date"], gocheck.Not(gocheck.Equals), "")

	c.Assert(resp.Topic.TopicArn, gocheck.Equals, "arn:aws:sns:us-east-1:123456789012:My-Topic")
	c.Assert(resp.ResponseMetadata.RequestId, gocheck.Equals, "a8dec8b3-33a4-11df-8963-01868b7c937a")
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestDeleteTopic(c *gocheck.C) {
	testServer.Response(200, nil, TestDeleteTopicXmlOK)

	t := sns.Topic{nil, "arn:aws:sns:us-east-1:123456789012:My-Topic"}
	resp, err := s.sns.DeleteTopic(t)
	req := testServer.WaitRequest()

	c.Assert(req.Method, gocheck.Equals, "GET")
	c.Assert(req.URL.Path, gocheck.Equals, "/")
	c.Assert(req.Header["Date"], gocheck.Not(gocheck.Equals), "")

	c.Assert(resp.ResponseMetadata.RequestId, gocheck.Equals, "f3aa9ac9-3c3d-11df-8235-9dab105e9c32")
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestListSubscriptions(c *gocheck.C) {
	testServer.Response(200, nil, TestListSubscriptionsXmlOK)

	resp, err := s.sns.ListSubscriptions(nil)
	req := testServer.WaitRequest()

	c.Assert(req.Method, gocheck.Equals, "GET")
	c.Assert(req.URL.Path, gocheck.Equals, "/")
	c.Assert(req.Header["Date"], gocheck.Not(gocheck.Equals), "")

	c.Assert(len(resp.Subscriptions), gocheck.Not(gocheck.Equals), 0)
	c.Assert(resp.Subscriptions[0].Protocol, gocheck.Equals, "email")
	c.Assert(resp.Subscriptions[0].Endpoint, gocheck.Equals, "example@amazon.com")
	c.Assert(resp.Subscriptions[0].SubscriptionArn, gocheck.Equals, "arn:aws:sns:us-east-1:123456789012:My-Topic:80289ba6-0fd4-4079-afb4-ce8c8260f0ca")
	c.Assert(resp.Subscriptions[0].TopicArn, gocheck.Equals, "arn:aws:sns:us-east-1:698519295917:My-Topic")
	c.Assert(resp.Subscriptions[0].Owner, gocheck.Equals, "123456789012")
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestGetTopicAttributes(c *gocheck.C) {
	testServer.Response(200, nil, TestGetTopicAttributesXmlOK)

	resp, err := s.sns.GetTopicAttributes("arn:aws:sns:us-east-1:123456789012:My-Topic")
	req := testServer.WaitRequest()

	c.Assert(req.Method, gocheck.Equals, "GET")
	c.Assert(req.URL.Path, gocheck.Equals, "/")
	c.Assert(req.Header["Date"], gocheck.Not(gocheck.Equals), "")

	c.Assert(len(resp.Attributes), gocheck.Not(gocheck.Equals), 0)
	c.Assert(resp.Attributes[0].Key, gocheck.Equals, "Owner")
	c.Assert(resp.Attributes[0].Value, gocheck.Equals, "123456789012")
	c.Assert(resp.Attributes[1].Key, gocheck.Equals, "Policy")
	c.Assert(resp.Attributes[1].Value, gocheck.Equals, `{"Version":"2008-10-17","Id":"us-east-1/698519295917/test__default_policy_ID","Statement" : [{"Effect":"Allow","Sid":"us-east-1/698519295917/test__default_statement_ID","Principal" : {"AWS": "*"},"Action":["SNS:GetTopicAttributes","SNS:SetTopicAttributes","SNS:AddPermission","SNS:RemovePermission","SNS:DeleteTopic","SNS:Subscribe","SNS:ListSubscriptionsByTopic","SNS:Publish","SNS:Receive"],"Resource":"arn:aws:sns:us-east-1:698519295917:test","Condition" : {"StringLike" : {"AWS:SourceArn": "arn:aws:*:*:698519295917:*"}}}]}`)
	c.Assert(resp.ResponseMetadata.RequestId, gocheck.Equals, "057f074c-33a7-11df-9540-99d0768312d3")
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestPublish(c *gocheck.C) {
	testServer.Response(200, nil, TestPublishXmlOK)

	pubOpt := &sns.PublishOpt{"foobar", "", "subject", "arn:aws:sns:us-east-1:123456789012:My-Topic"}
	resp, err := s.sns.Publish(pubOpt)
	req := testServer.WaitRequest()

	c.Assert(req.Method, gocheck.Equals, "GET")
	c.Assert(req.URL.Path, gocheck.Equals, "/")
	c.Assert(req.Header["Date"], gocheck.Not(gocheck.Equals), "")

	c.Assert(resp.MessageId, gocheck.Equals, "94f20ce6-13c5-43a0-9a9e-ca52d816e90b")
	c.Assert(resp.ResponseMetadata.RequestId, gocheck.Equals, "f187a3c1-376f-11df-8963-01868b7c937a")
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestSetTopicAttributes(c *gocheck.C) {
	testServer.Response(200, nil, TestSetTopicAttributesXmlOK)

	resp, err := s.sns.SetTopicAttributes("DisplayName", "MyTopicName", "arn:aws:sns:us-east-1:123456789012:My-Topic")
	req := testServer.WaitRequest()

	c.Assert(req.Method, gocheck.Equals, "GET")
	c.Assert(req.URL.Path, gocheck.Equals, "/")
	c.Assert(req.Header["Date"], gocheck.Not(gocheck.Equals), "")

	c.Assert(resp.ResponseMetadata.RequestId, gocheck.Equals, "a8763b99-33a7-11df-a9b7-05d48da6f042")
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestSubscribe(c *gocheck.C) {
	testServer.Response(200, nil, TestSubscribeXmlOK)

	resp, err := s.sns.Subscribe("example@amazon.com", "email", "arn:aws:sns:us-east-1:123456789012:My-Topic")
	req := testServer.WaitRequest()

	c.Assert(req.Method, gocheck.Equals, "GET")
	c.Assert(req.URL.Path, gocheck.Equals, "/")
	c.Assert(req.Header["Date"], gocheck.Not(gocheck.Equals), "")

	c.Assert(resp.SubscriptionArn, gocheck.Equals, "pending confirmation")
	c.Assert(resp.ResponseMetadata.RequestId, gocheck.Equals, "a169c740-3766-11df-8963-01868b7c937a")
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestUnsubscribe(c *gocheck.C) {
	testServer.Response(200, nil, TestUnsubscribeXmlOK)

	resp, err := s.sns.Unsubscribe("arn:aws:sns:us-east-1:123456789012:My-Topic:a169c740-3766-11df-8963-01868b7c937a")
	req := testServer.WaitRequest()

	c.Assert(req.Method, gocheck.Equals, "GET")
	c.Assert(req.URL.Path, gocheck.Equals, "/")
	c.Assert(req.Header["Date"], gocheck.Not(gocheck.Equals), "")

	c.Assert(resp.ResponseMetadata.RequestId, gocheck.Equals, "18e0ac39-3776-11df-84c0-b93cc1666b84")
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestConfirmSubscription(c *gocheck.C) {
	testServer.Response(200, nil, TestConfirmSubscriptionXmlOK)

	opt := &sns.ConfirmSubscriptionOpt{"", "51b2ff3edb475b7d91550e0ab6edf0c1de2a34e6ebaf6", "arn:aws:sns:us-east-1:123456789012:My-Topic"}
	resp, err := s.sns.ConfirmSubscription(opt)
	req := testServer.WaitRequest()

	c.Assert(req.Method, gocheck.Equals, "GET")
	c.Assert(req.URL.Path, gocheck.Equals, "/")
	c.Assert(req.Header["Date"], gocheck.Not(gocheck.Equals), "")

	c.Assert(resp.SubscriptionArn, gocheck.Equals, "arn:aws:sns:us-east-1:123456789012:My-Topic:80289ba6-0fd4-4079-afb4-ce8c8260f0ca")
	c.Assert(resp.ResponseMetadata.RequestId, gocheck.Equals, "7a50221f-3774-11df-a9b7-05d48da6f042")
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestAddPermission(c *gocheck.C) {
	testServer.Response(200, nil, TestAddPermissionXmlOK)
	perm := make([]sns.Permission, 2)
	perm[0].ActionName = "Publish"
	perm[1].ActionName = "GetTopicAttributes"
	perm[0].AccountId = "987654321000"
	perm[1].AccountId = "876543210000"

	resp, err := s.sns.AddPermission(perm, "NewPermission", "arn:aws:sns:us-east-1:123456789012:My-Topic")
	req := testServer.WaitRequest()

	c.Assert(req.Method, gocheck.Equals, "GET")
	c.Assert(req.URL.Path, gocheck.Equals, "/")
	c.Assert(req.Header["Date"], gocheck.Not(gocheck.Equals), "")

	c.Assert(resp.RequestId, gocheck.Equals, "6a213e4e-33a8-11df-9540-99d0768312d3")
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestRemovePermission(c *gocheck.C) {
	testServer.Response(200, nil, TestRemovePermissionXmlOK)

	resp, err := s.sns.RemovePermission("NewPermission", "arn:aws:sns:us-east-1:123456789012:My-Topic")
	req := testServer.WaitRequest()

	c.Assert(req.Method, gocheck.Equals, "GET")
	c.Assert(req.URL.Path, gocheck.Equals, "/")
	c.Assert(req.Header["Date"], gocheck.Not(gocheck.Equals), "")

	c.Assert(resp.RequestId, gocheck.Equals, "d170b150-33a8-11df-995a-2d6fbe836cc1")
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestListSubscriptionByTopic(c *gocheck.C) {
	testServer.Response(200, nil, TestListSubscriptionsByTopicXmlOK)

	opt := &sns.ListSubscriptionByTopicOpt{"", "arn:aws:sns:us-east-1:123456789012:My-Topic"}
	resp, err := s.sns.ListSubscriptionByTopic(opt)
	req := testServer.WaitRequest()

	c.Assert(req.Method, gocheck.Equals, "GET")
	c.Assert(req.URL.Path, gocheck.Equals, "/")
	c.Assert(req.Header["Date"], gocheck.Not(gocheck.Equals), "")

	c.Assert(len(resp.Subscriptions), gocheck.Not(gocheck.Equals), 0)
	c.Assert(resp.Subscriptions[0].TopicArn, gocheck.Equals, "arn:aws:sns:us-east-1:123456789012:My-Topic")
	c.Assert(resp.Subscriptions[0].SubscriptionArn, gocheck.Equals, "arn:aws:sns:us-east-1:123456789012:My-Topic:80289ba6-0fd4-4079-afb4-ce8c8260f0ca")
	c.Assert(resp.Subscriptions[0].Owner, gocheck.Equals, "123456789012")
	c.Assert(resp.Subscriptions[0].Endpoint, gocheck.Equals, "example@amazon.com")
	c.Assert(resp.Subscriptions[0].Protocol, gocheck.Equals, "email")
	c.Assert(err, gocheck.IsNil)
}
