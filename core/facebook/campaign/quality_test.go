package campaign

import (
	"testing"

	"bitbucket.org/backend/core/server"
)

type qualityClient struct {
	// fail the request
	failRequest bool

	// fail api operation with a facebook error
	fail bool

	failBodyRead  bool
	failUnmarshal bool

	// zeroDays is used to return an adset with 0 days life
	zeroDays bool

	// fail to parse data providing incorrect data
	failStartTimeParse bool
	failEndTimeParse   bool
	failReachParse     bool
	failUniqueCTRParse bool
	failCMPParse       bool

	server.Client
	t *testing.T
}

func TestQuality(t *testing.T) {

}
