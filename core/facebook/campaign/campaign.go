package campaign

import (
	"bitbucket.org/backend/core/entities"
	"bitbucket.org/backend/core/facebook/auth"
	"bitbucket.org/backend/core/genetic"
	"bitbucket.org/backend/core/server"
	"bitbucket.org/backend/core/storage/campaigns"
	"github.com/aws/aws-sdk-go/aws/session"
)

// Request contains the data required to create
// a new facebook campaign
type Request struct {
	// Campaign data
	Name              string   `json:"name"`
	Objective         string   `json:"objective"`
	Budget            string   `json:"budget"`
	SpecialAdCategory []string `json:"special_ad_categories"`
	// Population and optimization data
	Segment      string  `json:"segment"`
	MutationRate float64 `json:"mutation_rate"`
	// Ad set data
	PixelID   string      `json:"pixel_id"`
	StartTime string      `json:"start"`
	EndTime   string      `json:"end"`
	Location  geolocation `json:"locations"`
	Gender    [2]int      `json:"gender"`
	AgeMin    int         `json:"age_min"`
	AgeMax    int         `json:"age_max"`
	// Creative data
	Page         entities.Page `json:"page"`
	CreativeName string        `json:"creative_name"`
	ImageHash    string        `json:"image_hash"`
	MediaURL     string        `json:"media_url"`
	VideoID      string        `json:"video_id"`
	Title        string        `json:"title"`
	Message      string        `json:"message"`
	CallToAction callToAction  `json:"call_to_action"`
	// Account data
	AdAccount string `json:"ad_account"`
}

type callToAction struct {
	Type  string            `json:"type"`
	Value callToActionValue `json:"value"`
}

type callToActionValue struct {
	Link string `json:"link"`
	Page string `json:"page"`
}

type targeting struct {
	GeoLocation    geolocation       `json:"geo_locations"`
	Gender         [2]int            `json:"genders,omitempty"`
	AgeMin         int               `json:"age_min,omitempty"`
	AgeMax         int               `json:"age_max,omitempty"`
	Behaviors      []targetingByType `json:"behaviors,omitempty"`
	Interests      []targetingByType `json:"interests,omitempty"`
	LifeEvents     []targetingByType `json:"life_events,omitempty"`
	FamilyStatuses []targetingByType `json:"family_statuses,omitempty"`
	Industries     []targetingByType `json:"industries,omitempty"`
}

type geolocation struct {
	Cities []struct {
		Key    string `json:"key"`
		Radius string `json:"radius"`
	} `json:"cities"`
	CountryGroups []string `json:"country_group"`
	Countries     []string `json:"countries"`
	Regions       []struct {
		Key string `json:"key"`
	} `json:"regions"`
	Zips []struct {
		Key string `json:"key"`
	} `json:"zips"`
}

type geneticTargeting struct {
	Behaviors      []targetingByType `json:"behaviors"`
	Interests      []targetingByType `json:"interests"`
	LifeEvents     []targetingByType `json:"life_events"`
	FamilyStatuses []targetingByType `json:"family_statuses"`
	Industries     []targetingByType `json:"industries"`
}

type targetingByType struct {
	RawName string   `json:"raw_name"`
	Name    string   `json:"name"`
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Path    []string `json:"path"`
}

// Campaign methods for facebook
type Campaign interface {
	Create(userID string, req *Request) (*entities.Campaign, error)
}

type facebook struct {
	store  campaigns.Storage
	client server.Client
	auth   auth.Auth
	// campaign objects configuration
	status string
	// billing event for adsets
	billingEvent string
	// optimization algorithm
	quality   *q
	selection genetic.Genetic
}

// New campaign facebook interface
func New(sess *session.Session, config ...func(*facebook)) Campaign {
	f := &facebook{
		auth:         auth.New(sess),
		store:        campaigns.New(sess),
		client:       server.New(),
		status:       "ACTIVE",
		billingEvent: "IMPRESSIONS",
		quality:      quality(),
	}
	f.selection = genetic.New(f.quality.compute)

	for _, fn := range config {
		fn(f)
	}

	return f
}
