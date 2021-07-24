package campaigns

import (
	"bitbucket.org/backend/core/entities"
	"bitbucket.org/backend/core/genetic"
)

// Storage interface to get campaign information from database
type Storage interface {
	// Campaign storage
	StoreCampaign(userID, platform, adAccount, segment string, c *entities.Campaign) error
	GetCampaign(campaignID string) (*entities.Campaign, error)
	GetUserCampaigns(userID string) (map[string][]string, error)
	// GetActiveCampaigns returns a maping from userID to active campaigns' ID
	GetActiveCampaigns(platform string) (map[string][]string, error)

	// Segment Storage
	// SetSegment initialices a segment to the provided initial targeting population
	SetSegment(userID, segment string, initialPopulation []*genetic.Chromosome) error
	// GetSegment returns initial targeting population of the segment
	GetSegment(userID, segment string) ([]*genetic.Chromosome, error)
	// GetSegments returns the names of user defined segments
	GetSegments(userID string) ([]string, error)
	// GetSegmentCampaigns returns all campaigns created by a
	// user initialized segment sorted in decesing order by end time
	GetSegmentCampaigns(userID, segment string) ([]string, error)
}
