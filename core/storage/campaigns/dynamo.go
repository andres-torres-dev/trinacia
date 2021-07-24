package campaigns

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"bitbucket.org/backend/core/entities"
	"bitbucket.org/backend/core/genetic"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type dynamo struct {
	svc *dynamodb.DynamoDB
}

// New isntanciates a session dynamo session to query information
// about users' campaigns
func New(sess *session.Session) Storage {
	return &dynamo{
		svc: dynamodb.New(sess),
	}
}

type sortKeys struct {
	Partition  string `json:"partition"`
	Key        string `json:"key"`
	Sort       string `json:"sort"`
	SecondSort string `json:"secondSort"`
	ThirdSort  string `json:"thirdSort"`
	FourthSort string `json:"fourthSort"`
	FifthSort  string `json:"fifthSort"`
}

const (
	// TableName is the table used to store the data
	TableName = "trinacia"
)

var (
	// ErrorMissingUserID missing user id
	ErrorMissingUserID = errors.New("Missing User ID")
	// ErrorMissingPlatform missing platform
	ErrorMissingPlatform = errors.New("Missing Platform")
	// ErrorMissingAdAccount missing ad account
	ErrorMissingAdAccount = errors.New("Missing ad account")
	// ErrorMissingSegment missing segment
	ErrorMissingSegment = errors.New("Missing segment")
	// ErrorInvalidCampaign one or more fields is missing to create a campaign
	ErrorInvalidCampaign = errors.New("Missing some of the required campaign fields")
	// ErrorMissingCampaignID missing campaign id
	ErrorMissingCampaignID = errors.New("Missing campaign id")
	// ErrorUnableToFindCampaign get campaign found no result
	ErrorUnableToFindCampaign = errors.New("Unable to find the campaign")
)

func (d *dynamo) StoreCampaign(userID, platform, adAccount, segment string, c *entities.Campaign) error {
	if userID == "" {
		return ErrorMissingUserID
	}
	if platform == "" {
		return ErrorMissingPlatform
	}
	if adAccount == "" {
		return ErrorMissingAdAccount
	}
	if segment == "" {
		return ErrorMissingSegment
	}
	if c == nil || c.ID == "" || c.StartTime == "" || c.EndTime == "" || c.Budget == "" || len(c.Targeting) == 0 || len(c.Media) == 0 {
		return ErrorInvalidCampaign
	}

	targeting, err := dynamodbattribute.Marshal(c.Targeting)
	if err != nil {
		return err
	}
	media, err := dynamodbattribute.Marshal(c.Media)
	if err != nil {
		return err
	}

	in := &dynamodb.UpdateItemInput{
		TableName: aws.String(TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"partition": {
				S: aws.String("campaigns"),
			},
			"key": {
				S: aws.String(c.ID),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#platform":  aws.String("plaform"),
			"#segment":   aws.String("segment"),
			"#adAccount": aws.String("ad_account"),
			"#campaign":  aws.String("id"),
			"#startTime": aws.String("start_time"),
			"#endTime":   aws.String("end_time"),
			"#budget":    aws.String("budget"),
			"#targeting": aws.String("targeting"),
			"#media":     aws.String("media"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":sort": {
				S: aws.String(userID),
			},
			":secondSort": {
				S: aws.String(c.EndTime),
			},
			":thirdSort": {
				S: aws.String(fmt.Sprintf("%s:%s", userID, segment)),
			},
			":fourthSort": {
				S: aws.String(platform),
			},
			":platform": {
				S: aws.String(platform),
			},
			":segment": {
				S: aws.String(segment),
			},
			":adAccount": {
				S: aws.String(adAccount),
			},
			":campaign": {
				S: aws.String(c.ID),
			},
			":startTime": {
				S: aws.String(c.StartTime),
			},
			":endTime": {
				S: aws.String(c.EndTime),
			},
			":budget": {
				S: aws.String(c.Budget),
			},
			":targeting": targeting,
			":media":     media,
		},
		UpdateExpression: aws.String("set sort=:sort, secondSort=:secondSort, thirdSort=:thirdSort, fourthSort=:fourthSort, #platform=:platform, #segment=:segment, #adAccount=:adAccount, #campaign=:campaign, #startTime=:startTime, #endTime=:endTime, #budget=:budget, #targeting=:targeting, #media=:media"),
	}
	_, err = d.svc.UpdateItem(in)
	if err != nil {
		return err
	}

	return nil
}

func (d *dynamo) GetCampaign(campaignID string) (*entities.Campaign, error) {
	if campaignID == "" {
		return nil, ErrorMissingCampaignID
	}
	c := &entities.Campaign{}

	in := &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"partition": {
				S: aws.String("campaigns"),
			},
			"key": {
				S: aws.String(campaignID),
			},
		},
	}
	out, err := d.svc.GetItem(in)
	if err != nil {
		return nil, err
	}
	err = dynamodbattribute.UnmarshalMap(out.Item, c)
	if err != nil {
		return nil, err
	}

	if c.ID == "" {
		return nil, ErrorUnableToFindCampaign
	}

	return c, nil
}

func (d *dynamo) GetUserCampaigns(userID string) (map[string][]string, error) {
	if userID == "" {
		return nil, ErrorMissingUserID
	}
	cIDs := []sortKeys{}

	in := &dynamodb.QueryInput{
		TableName: aws.String(TableName),
		ExpressionAttributeNames: map[string]*string{
			"#p": aws.String("partition"),
			"#s": aws.String("sort"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":partition": {
				S: aws.String("campaigns"),
			},
			":sort": {
				S: aws.String(userID),
			},
		},
		IndexName:              aws.String("partition-sort-index"),
		KeyConditionExpression: aws.String("#p = :partition AND #s = :sort"),
	}
	out, err := d.svc.Query(in)
	if err != nil {
		return nil, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(out.Items, &cIDs)
	if err != nil {
		return nil, err
	}

	c := make(map[string][]string)
	for _, keys := range cIDs {
		c[keys.FourthSort] = append(c[keys.FourthSort], keys.Key)
	}

	return c, nil
}

func (d *dynamo) GetActiveCampaigns(platform string) (map[string][]string, error) {
	if platform == "" {
		return nil, ErrorMissingPlatform
	}
	cIDs := []sortKeys{}

	in := &dynamodb.QueryInput{
		TableName: aws.String(TableName),
		ExpressionAttributeNames: map[string]*string{
			"#p":  aws.String("partition"),
			"#ss": aws.String("secondSort"),
			"#4s": aws.String("fourthSort"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":partition": {
				S: aws.String("campaigns"),
			},
			":secondSort": {
				S: aws.String(time.Now().String()),
			},
			":4s": {
				S: aws.String(platform),
			},
		},
		IndexName:              aws.String("partition-secondSort-index"),
		KeyConditionExpression: aws.String("#p = :partition AND #ss > :secondSort"),
		FilterExpression:       aws.String("#4s = :4s"),
	}
	out, err := d.svc.Query(in)
	if err != nil {
		return nil, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(out.Items, &cIDs)
	if err != nil {
		return nil, err
	}

	c := make(map[string][]string)
	for _, keys := range cIDs {
		c[keys.Sort] = append(c[keys.Sort], keys.Key)
	}

	return c, nil
}

func (d *dynamo) SetSegment(userID, segment string, initialPopulation []*genetic.Chromosome) error {
	if userID == "" {
		return ErrorMissingUserID
	}
	if segment == "" {
		return ErrorMissingSegment
	}

	population, err := dynamodbattribute.Marshal(initialPopulation)
	if err != nil {
		return err
	}

	in := &dynamodb.UpdateItemInput{
		TableName: aws.String(TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"partition": {
				S: aws.String(userID),
			},
			"key": {
				S: aws.String("segments"),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#segment": aws.String(segment),
			"#all":     aws.String("names"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":segment": population,
			":name": {
				S: aws.String(segment),
			},
			":empty": {
				L: []*dynamodb.AttributeValue{},
			},
		},
		UpdateExpression: aws.String("set #segment=:segment, #all=list_append(if_not_exists(#all, :empty), :name)"),
	}
	_, err = d.svc.UpdateItem(in)

	return err
}

func (d *dynamo) GetSegment(userID, segment string) ([]*genetic.Chromosome, error) {
	if userID == "" {
		return nil, ErrorMissingUserID
	}
	if segment == "" {
		return nil, ErrorMissingSegment
	}

	population := []*genetic.Chromosome{}

	in := &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"partition": {
				S: aws.String(userID),
			},
			"key": {
				S: aws.String("segments"),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#segment": aws.String(segment),
		},
		ProjectionExpression: aws.String("#segment"),
	}
	out, err := d.svc.GetItem(in)
	if err != nil {
		return nil, err
	}
	err = dynamodbattribute.Unmarshal(out.Item[segment], population)
	if err != nil {
		return nil, err
	}

	return population, nil
}

func (d *dynamo) GetSegments(userID string) ([]string, error) {
	if userID == "" {
		return nil, ErrorMissingUserID
	}
	in := &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"partition": {
				S: aws.String(userID),
			},
			"key": {
				S: aws.String("segments"),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#names": aws.String("names"),
		},
		ProjectionExpression: aws.String("#names"),
	}
	out, err := d.svc.GetItem(in)
	if err != nil {
		return nil, err
	}

	segments := []string{}
	err = dynamodbattribute.Unmarshal(out.Item["names"], &segments)
	if err != nil {
		return nil, err
	}

	return segments, nil
}

func (d *dynamo) GetSegmentCampaigns(userID, segment string) ([]string, error) {
	if userID == "" {
		return nil, ErrorMissingUserID
	}
	if segment == "" {
		return nil, ErrorMissingSegment
	}
	cIDs := []sortKeys{}

	in := &dynamodb.QueryInput{
		TableName: aws.String(TableName),
		ExpressionAttributeNames: map[string]*string{
			"#p": aws.String("partition"),
			"#s": aws.String("thirdSort"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":partition": {
				S: aws.String("campaigns"),
			},
			":thirdSort": {
				S: aws.String(fmt.Sprintf("%s:%s", userID, segment)),
			},
		},
		IndexName:              aws.String("partition-thirdSort-index"),
		KeyConditionExpression: aws.String("#p = :partition AND #s = :thirdSort"),
	}
	out, err := d.svc.Query(in)
	if err != nil {
		return nil, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(out.Items, &cIDs)
	if err != nil {
		return nil, err
	}

	sort.SliceStable(cIDs, func(i, j int) bool {
		return cIDs[i].SecondSort > cIDs[j].SecondSort
	})

	c := make([]string, len(cIDs))
	for i, keys := range cIDs {
		c[i] = keys.Key
	}

	return c, nil
}
