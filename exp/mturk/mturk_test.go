package mturk_test

import (
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/exp/mturk"
	"github.com/crowdmob/goamz/testutil"
	"launchpad.net/gocheck"
	"net/url"
	"testing"
)

func Test(t *testing.T) {
	gocheck.TestingT(t)
}

var _ = gocheck.Suite(&S{})

type S struct {
	mturk *mturk.MTurk
}

var testServer = testutil.NewHTTPServer()

func (s *S) SetUpSuite(c *gocheck.C) {
	testServer.Start()
	auth := aws.Auth{AccessKey: "abc", SecretKey: "123"}
	u, err := url.Parse(testServer.URL)
	if err != nil {
		panic(err.Error())
	}

	s.mturk = &mturk.MTurk{
		Auth: auth,
		URL:  u,
	}
}

func (s *S) TearDownTest(c *gocheck.C) {
	testServer.Flush()
}

func (s *S) TestCreateHIT(c *gocheck.C) {
	testServer.Response(200, nil, BasicHitResponse)

	question := mturk.ExternalQuestion{
		ExternalURL: "http://www.amazon.com",
		FrameHeight: 200,
	}
	reward := mturk.Price{
		Amount:       "0.01",
		CurrencyCode: "USD",
	}
	hit, err := s.mturk.CreateHIT("title", "description", question, reward, 1, 2, "key1,key2", 3, nil, "annotation")

	testServer.WaitRequest()

	c.Assert(err, gocheck.IsNil)
	c.Assert(hit, gocheck.NotNil)

	c.Assert(hit.HITId, gocheck.Equals, "28J4IXKO2L927XKJTHO34OCDNASCDW")
	c.Assert(hit.HITTypeId, gocheck.Equals, "2XZ7D1X3V0FKQVW7LU51S7PKKGFKDF")
}

func (s *S) TestSearchHITs(c *gocheck.C) {
	testServer.Response(200, nil, SearchHITResponse)

	hitResult, err := s.mturk.SearchHITs()

	c.Assert(err, gocheck.IsNil)
	c.Assert(hitResult, gocheck.NotNil)

	c.Assert(hitResult.NumResults, gocheck.Equals, uint(1))
	c.Assert(hitResult.PageNumber, gocheck.Equals, uint(1))
	c.Assert(hitResult.TotalNumResults, gocheck.Equals, uint(1))

	c.Assert(len(hitResult.HITs), gocheck.Equals, 1)
	c.Assert(hitResult.HITs[0].HITId, gocheck.Equals, "2BU26DG67D1XTE823B3OQ2JF2XWF83")
	c.Assert(hitResult.HITs[0].HITTypeId, gocheck.Equals, "22OWJ5OPB0YV6IGL5727KP9U38P5XR")
	c.Assert(hitResult.HITs[0].CreationTime, gocheck.Equals, "2011-12-28T19:56:20Z")
	c.Assert(hitResult.HITs[0].Title, gocheck.Equals, "test hit")
	c.Assert(hitResult.HITs[0].Description, gocheck.Equals, "please disregard, testing only")
	c.Assert(hitResult.HITs[0].HITStatus, gocheck.Equals, "Reviewable")
	c.Assert(hitResult.HITs[0].MaxAssignments, gocheck.Equals, uint(1))
	c.Assert(hitResult.HITs[0].Reward.Amount, gocheck.Equals, "0.01")
	c.Assert(hitResult.HITs[0].Reward.CurrencyCode, gocheck.Equals, "USD")
	c.Assert(hitResult.HITs[0].AutoApprovalDelayInSeconds, gocheck.Equals, uint(2592000))
	c.Assert(hitResult.HITs[0].AssignmentDurationInSeconds, gocheck.Equals, uint(30))
	c.Assert(hitResult.HITs[0].NumberOfAssignmentsPending, gocheck.Equals, uint(0))
	c.Assert(hitResult.HITs[0].NumberOfAssignmentsAvailable, gocheck.Equals, uint(1))
	c.Assert(hitResult.HITs[0].NumberOfAssignmentsCompleted, gocheck.Equals, uint(0))
}
