package campaign

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"sync"

	"bitbucket.org/backend/core/entities"
	"bitbucket.org/backend/core/facebook/internal"
	"bitbucket.org/backend/core/genetic"
	"bitbucket.org/backend/core/logger"
)

const (
	populationSize = 30
	selectionSize  = 5
)

var (
	errInvalidToken = errors.New("Facebook access token has expired or is invalid")
	// configuration parameters errors
	errorMissingSegment      = errors.New("Request missing segment name")
	errorInvalidMutationRage = errors.New("Request invalid mutation rate")
	errorMissingAdAccount    = errors.New("Request missing ad account")

	// campaign parameters errors
	errorMissingCampaignName      = errors.New("Request missing campaign name")
	errorInvalidBudget            = errors.New("Request budget is less than minimun budget")
	errorMissingSpecialAdCategory = errors.New("Request missing special ad category")
	errorMissingCampaignObjective = errors.New("Request missing campaign objective")

	// adsets parameters errors
	errorMissingStartTime = errors.New("Request missing start time")
	errorMissingEndTime   = errors.New("Request missing end time")
	// TODO add campaign without endtime
	errorMissingLocation = errors.New("Request missing end time")
	// TODO add gender and age property to segment
	errorMissingGender = errors.New("Request missing gender")
	errMissingAge      = errors.New("Request missing age")

	// ads and creatives parameters errors
	errorMissingPage          = errors.New("Request missing page")
	errorMissingCallToAction  = errors.New("Request missing call to action")
	errorMissingCreativeName  = errors.New("Request missing creative name")
	errorMissingCreativeMedia = errors.New("Request missing ads media")
)

type newCampaign struct {
	Name              string   `json:"name"`
	Objective         string   `json:"objective"`
	DailyBudget       string   `json:"daily_budget"`
	BidStrategy       string   `json:"bid_strategy"`
	Status            string   `json:"status"`
	SpecialAdCategory []string `json:"special_ad_categories"`
	Token             string   `json:"access_token"`
}

type newAdSet struct {
	Name           string         `json:"name"`
	BillingEvent   string         `json:"billing_event"`
	CampaignID     string         `json:"campaign_id"`
	PromotedObject promotedObject `json:"promoted_object,omitempty"`
	Targeting      *targeting     `json:"targeting"`
	Status         string         `json:"status"`
	StartTime      string         `json:"start_time"`
	EndTime        string         `json:"end_time"`
	AccessToken    string         `json:"access_token"`
}

type promotedObject struct {
	PageID       string `json:"page_id,omitempty"`
	PixelID      string `json:"pixel_id,omitempty"`
	ProductSetID string `json:"product_set_id,omitempty"`
}

type newCreative struct {
	// Title for a page likes ad
	Title           string `json:"title,omitempty"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	ObjectStroySpec struct {
		PageID    string    `json:"page_id"`
		LinkData  linkData  `json:"link_data,omitempty"`
		VideoData videoData `json:"video_data,omitempty"`
	} `json:"object_story_spec,omitempty"`
	AccessToken string `json:"access_token"`
}

type linkData struct {
	ImageHash string `json:"image_hash"`
	Link      string `json:"link"`
	// Message is the main body text of the post
	Message      string       `json:"message"`
	CallToAction callToAction `json:"call_to_action"`
}

type videoData struct {
	// Image hash for the image to use as thumbnail
	ImageHash string `json:"image_hash"`
	// Title of the video, can't be used with a like page cta
	Title        string       `json:"title"`
	Message      string       `json:"message"`
	VideoID      string       `json:"video_id"`
	CallToAction callToAction `json:"call_to_action"`
}

type creative struct {
	CreativeID string `json:"creative_id"`
}

type newAd struct {
	Name        string   `json:"name"`
	AdsetID     string   `json:"adset_id"`
	Creative    creative `json:"creative"`
	Status      string   `json:"status"`
	AccessToken string   `json:"access_token"`
}

func (f *facebook) Create(userID string, req *Request) (*entities.Campaign, error) {
	var (
		c *entities.Campaign
		u *entities.Facebook
	)

	if err := checkRequest(req); err != nil {
		return nil, &logger.Error{
			Level:   "Error",
			Message: "Invalid Request",
			Err:     err,
		}
	}
	u, valid, err := f.auth.GetUser(userID)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, &logger.Error{
			Level:   "Warning",
			Message: "Unable to create campaign because the user access token is not valid",
			Err:     errInvalidToken,
		}
	}

	// initialice quality structure to enable
	// selection algorithm to query facebook with
	// a valid access token
	f.quality.accessToken = u.AccessToken

	campaignID, err := f.createCampaign(req.AdAccount, req.Name, u.AccessToken, req.Objective, req.Budget, req.SpecialAdCategory)
	if err != nil {
		return nil, err
	}

	// get current segment population
	initialPopulation, err := f.store.GetSegment(userID, req.Segment)
	if err != nil {
		return nil, &logger.Error{
			Level:   "panic",
			Message: "Unable to get segment population from data base",
			Err:     err,
		}
	}

	// optimize current population using genetic algorithm
	newPopulation, err := f.newPopulation(initialPopulation, req.MutationRate)
	if err != nil {
		return nil, err
	}

	// create adsets and update IDs to the segment population
	// to point to the new adsets
	//
	// IDs are used by the quality function to retrieve performance
	// data from facebook
	var promotedObjectID string
	switch req.Objective {
	case "PAGE_LIKES":
		promotedObjectID = req.Page.ID
	case "CONVERSIONS":
		promotedObjectID = req.PixelID
	}
	adSets, err := f.createAdSets(req.AdAccount, campaignID, req.Objective, promotedObjectID, req.StartTime, req.EndTime,
		u.AccessToken, req.Location, req.Gender, req.AgeMin, req.AgeMax, newPopulation)
	if err != nil {
		return nil, err
	}

	// update segment current population
	err = f.store.SetSegment(userID, req.Segment, newPopulation)
	if err != nil {
		return nil, &logger.Error{
			Level:   "panic",
			Message: "Unable to update segment population in the data base",
			Err:     err,
		}
	}

	creativeID, err := f.createCreative(req, u.AccessToken)
	if err != nil {
		return nil, err
	}

	_, err = f.createAds(req.AdAccount, adSets, creativeID, u.AccessToken)
	if err != nil {
		return nil, err
	}

	c = &entities.Campaign{
		ID:        campaignID,
		Budget:    req.Budget,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Targeting: newPopulation,
		Media: []entities.Media{
			{
				Title:     req.Title,
				Body:      req.Message,
				VideoID:   req.VideoID,
				URL:       req.MediaURL,
				ImageHash: req.ImageHash,
			},
		},
	}

	err = f.store.StoreCampaign(userID, "facebook", req.AdAccount, req.Segment, c)
	if err != nil {
		return nil, &logger.Error{
			Level:   "panic",
			Message: "Unable to store user campaign.",
			Err:     err,
			Context: c,
		}
	}

	return c, nil
}

func checkRequest(req *Request) error {
	switch {
	// configuration parameters
	case req.Segment == "":
		return errorMissingSegment
	case req.MutationRate == 0.0 || req.MutationRate > 0.20:
		return errorInvalidMutationRage
	case req.AdAccount == "":
		return errorMissingAdAccount

	// campaign parameters
	case req.Name == "":
		return errorMissingCampaignName
	case req.Budget < "3000" || req.Budget == "":
		return errorInvalidBudget
	case req.SpecialAdCategory == nil:
		return errorMissingSpecialAdCategory
	case req.Objective == "":
		return errorMissingCampaignObjective

	// adset parameters
	case req.StartTime == "":
		return errorMissingStartTime
	case req.EndTime == "":
		return errorMissingEndTime
	case reflect.DeepEqual(req.Location, geolocation{}):
		return errorMissingLocation
	case len(req.Gender) == 0:
		return errorMissingGender
	case req.AgeMin == 0 || req.AgeMax == 0:
		return errMissingAge

	// ads && creative parameters
	case req.Page.ID == "":
		return errorMissingPage
	// call to action can be of type NONE but not an empty parameter
	case req.CallToAction == callToAction{}:
		return errorMissingCallToAction
	case req.CreativeName == "":
		return errorMissingCreativeName
	case req.ImageHash == "" && req.VideoID == "":
		return errorMissingCreativeMedia

	default:
		return nil
	}
}

func (f *facebook) createCampaign(adAccount, name, accessToken, objective, budget string, specialAdCategory []string) (string, error) {
	var (
		result = struct {
			ID    string                  `json:"id"`
			Error *internal.FacebookError `json:"error"`
		}{}
		newCampaign = newCampaign{
			Name:              name,
			Objective:         objective,
			DailyBudget:       budget,
			BidStrategy:       "LOWEST_COST_WITHOUT_CAP",
			Status:            f.status,
			SpecialAdCategory: specialAdCategory,
			Token:             accessToken,
		}
	)
	u := internal.SetURL(fmt.Sprintf("%s/campaigns", adAccount), nil)
	b, err := json.Marshal(newCampaign)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to marshal data to create a campaign.",
			Err:     err,
		}
	}
	resp, err := f.client.Post(u, bytes.NewReader(b))
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to perform request to create a campaign.",
			Err:     err,
		}
	}
	defer resp.Body.Close()
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to read response from request to create a campaign.",
			Err:     err,
		}
	}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to unmarshal response from request to create a camapign.",
			Err:     err,
		}
	}
	if result.Error != nil {
		return "", &logger.Error{
			Level:   "Error",
			Message: "Response from the creation of a campaign contained an error.",
			Context: []interface{}{newCampaign, u},
			Err:     result.Error,
		}
	}

	return result.ID, nil
}

func (f *facebook) newPopulation(initialPopulation []*genetic.Chromosome, mutationRate float64) ([]*genetic.Chromosome, error) {
	// compute initial population fitness
	err := f.selection.Fitness(initialPopulation)
	if err != nil {
		return nil, err
	}

	// compute selected population from initial population
	selected, err := f.selection.Selection(initialPopulation, selectionSize)
	if err != nil {
		return nil, err
	}

	var population = []*genetic.Chromosome{}
	population = append(population, selected...)
	idx := 0
	for len(population) < populationSize {
		if idx == selectionSize {
			idx = 0
		}
		c := *selected[idx]
		f.selection.Mutate(&c, mutationRate)
		population = append(population, &c)
		idx++
	}

	return population, nil
}

func (f *facebook) createAdSets(adAccount, campaignID, campaignObjective, promotedObject, startTime, endTime, accessToken string, location geolocation, gender [2]int, ageMin, ageMax int, population []*genetic.Chromosome) ([]string, error) {
	var adSets = make([]string, len(population))
	for i, c := range population {
		var t = &targeting{
			GeoLocation: location,
			Gender:      gender,
			AgeMin:      ageMin,
			AgeMax:      ageMax,
		}
		genotype := f.selection.Genesis(c)
		var wg sync.WaitGroup
		wg.Add(4)
		go func(wg *sync.WaitGroup) {
			t.Behaviors = setTargeting(genotype["behaviors"])
			wg.Done()
		}(&wg)
		go func(wg *sync.WaitGroup) {
			t.LifeEvents = setTargeting(genotype["interests"])
			wg.Done()
		}(&wg)
		go func(wg *sync.WaitGroup) {
			t.FamilyStatuses = setTargeting(genotype["family_statuses"])
			wg.Done()
		}(&wg)
		go func(wg *sync.WaitGroup) {
			t.Industries = setTargeting(genotype["industries"])
			wg.Done()
		}(&wg)
		wg.Wait()

		adSetID, err := f.createAdSet(adAccount, campaignID, campaignObjective, promotedObject, startTime, endTime, accessToken, t)
		if err != nil {
			return nil, err
		}
		adSets[i] = adSetID
		population[i].ID = adSetID
	}

	return adSets, nil
}

func setTargeting(genes []*genetic.Gene) []targetingByType {
	t := []targetingByType{}
	for _, gene := range genes {
		t = append(t, targetingByType{
			ID:   gene.ID,
			Name: gene.Name,
		})
	}

	return t
}

func (f *facebook) createAdSet(adAccount, campaignID, campaignObjective, promotedObjectID, startTime, endTime, accessToken string, t *targeting) (string, error) {
	var (
		result = struct {
			ID    string                  `json:"id"`
			Error *internal.FacebookError `json:"error"`
		}{}
		newAdSet = newAdSet{
			BillingEvent: f.billingEvent,
			CampaignID:   campaignID,
			Targeting:    t,
			Status:       f.status,
			StartTime:    startTime,
			EndTime:      endTime,
			AccessToken:  accessToken,
		}
	)
	switch campaignObjective {
	case "PAGE_LIKES":
		newAdSet.PromotedObject = promotedObject{
			PageID: promotedObjectID,
		}
	case "CONVERSIONS":
		newAdSet.PromotedObject = promotedObject{
			PixelID: promotedObjectID,
		}
	}

	name, err := randomName()
	if err != nil {
		return "", err
	}
	newAdSet.Name = name
	b, err := json.Marshal(newAdSet)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to marshal new adset data.",
			Err:     err,
		}
	}
	u := internal.SetURL(fmt.Sprintf("%s/adsets", adAccount), nil)
	resp, err := f.client.Post(u, bytes.NewReader(b))
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to perform post request to create adset.",
			Err:     err,
		}
	}
	defer resp.Body.Close()
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to read response to create adset.",
			Err:     err,
		}
	}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to unmarshal response to create adset.",
			Err:     err,
		}
	}

	if result.Error != nil {
		return "", &logger.Error{
			Level:   "Error",
			Message: "Response to create an adset contained an error.",
			Err:     result.Error,
		}
	}

	return result.ID, nil
}

func (f *facebook) createCreative(req *Request, accessToken string) (string, error) {
	var (
		result = struct {
			ID    string                  `json:"id"`
			Error *internal.FacebookError `json:"error"`
		}{}
		newCreative = newCreative{}
	)
	newCreative.Status = f.status
	if req.Objective == "PAGE_LIKES" {
		newCreative.Title = req.Title
	}
	newCreative.ObjectStroySpec.PageID = req.Page.ID
	switch {
	case req.VideoID != "":
		newCreative.Name = req.CreativeName
		newCreative.ObjectStroySpec.VideoData.ImageHash = req.ImageHash
		newCreative.ObjectStroySpec.VideoData.VideoID = req.VideoID
		newCreative.ObjectStroySpec.VideoData.Message = req.Message
		newCreative.ObjectStroySpec.VideoData.CallToAction = req.CallToAction

	case req.Objective == "PAGE_LIKES":
		newCreative.Name = req.CreativeName
		newCreative.ObjectStroySpec.LinkData.ImageHash = req.ImageHash
		newCreative.ObjectStroySpec.LinkData.Link = fmt.Sprintf("https://facebook.com/%s", req.Page.ID)
		newCreative.ObjectStroySpec.LinkData.Message = req.Message
		newCreative.ObjectStroySpec.LinkData.CallToAction = req.CallToAction

	default:
		newCreative.Name = req.CreativeName
		newCreative.ObjectStroySpec.LinkData.ImageHash = req.ImageHash
		newCreative.ObjectStroySpec.LinkData.Link = req.CallToAction.Value.Link
		newCreative.ObjectStroySpec.LinkData.Message = req.Message
		newCreative.ObjectStroySpec.LinkData.CallToAction = req.CallToAction
	}

	newCreative.AccessToken = accessToken
	b, err := json.Marshal(newCreative)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to marshal new creative object.",
			Err:     err,
		}
	}
	u := internal.SetURL(fmt.Sprintf("%s/adcreatives", req.AdAccount), nil)
	resp, err := f.client.Post(u, bytes.NewReader(b))
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to perform request to create creative.",
			Err:     err,
		}
	}
	defer resp.Body.Close()
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to read response from request to create creative.",
			Err:     err,
		}
	}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to unmarshal result from request to create creative.",
			Err:     err,
		}
	}

	if result.Error != nil {
		return "", &logger.Error{
			Level:   "Error",
			Message: "The response to create a creative contained an error.",
			Err:     result.Error,
		}
	}

	return result.ID, nil
}

func (f *facebook) createAds(adAccount string, adSets []string, creativeID, accessToken string) ([]string, error) {
	var ads = make([]string, len(adSets))
	for i, adset := range adSets {
		id, err := f.createAd(adAccount, adset, creativeID, accessToken)
		if err != nil {
			return nil, err
		}
		ads[i] = id
	}

	return ads, nil
}

func (f *facebook) createAd(adAccount, adsetID, creativeID, accessToken string) (string, error) {
	name, err := randomName()
	if err != nil {
		return "", err
	}
	var (
		result = struct {
			ID    string                  `json:"id"`
			Error *internal.FacebookError `json:"error"`
		}{}
		newAd = newAd{
			Name:    name,
			AdsetID: adsetID,
			Creative: creative{
				CreativeID: creativeID,
			},
			Status:      f.status,
			AccessToken: accessToken,
		}
	)
	b, err := json.Marshal(newAd)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to marshal new ad object.",
			Err:     err,
		}
	}
	u := internal.SetURL(fmt.Sprintf("%s/ads", adAccount), nil)
	resp, err := f.client.Post(u, bytes.NewReader(b))
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to perform request to create ad.",
			Err:     err,
		}
	}
	defer resp.Body.Close()
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to read response to create ad.",
			Err:     err,
		}
	}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to unmarshal response to create ad.",
			Err:     err,
		}
	}
	if result.Error != nil {
		return "", &logger.Error{
			Level:   "Error",
			Message: "The response to create an ad contained an error.",
			Err:     err,
		}
	}

	return result.ID, nil
}
