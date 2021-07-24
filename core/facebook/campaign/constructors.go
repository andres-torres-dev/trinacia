package campaign

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
	"sync"

	"bitbucket.org/backend/core/facebook/internal"
	"bitbucket.org/backend/core/genetic"
	"bitbucket.org/backend/core/logger"
	"bitbucket.org/backend/core/server"
)

// Constructor used to create a chromosome
// given an ad set
type Constructor interface {
	GenerateChromosome(adSetID, accessToken string) (*genetic.Chromosome, error)
}

type constructor struct {
	client server.Client
	test   bool
}

// NewConstructor creates a new constructor interface
func NewConstructor(config ...func(*constructor)) Constructor {
	c := &constructor{
		client: server.New(),
	}

	for _, fn := range config {
		fn(c)
	}

	return c
}

// GenerateChromosome creates a chromosome from an adset ID and an access token
func (constructor *constructor) GenerateChromosome(adSetID, accessToken string) (*genetic.Chromosome, error) {
	c, idx, err := constructor.loadDefaultChromosome()
	if err != nil {
		return c, err
	}

	c.ID = adSetID

	t, err := constructor.getAdSetTargeting(adSetID, accessToken)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup

	go func(wg *sync.WaitGroup) {
		wg.Add(1)
		populateChromosome(c, idx, t.Behaviors)
		wg.Done()
	}(&wg)
	go func(wg *sync.WaitGroup) {
		wg.Add(1)
		populateChromosome(c, idx, t.FamilyStatuses)
		wg.Done()
	}(&wg)
	go func(wg *sync.WaitGroup) {
		wg.Add(1)
		populateChromosome(c, idx, t.Industries)
		wg.Done()
	}(&wg)
	go func(wg *sync.WaitGroup) {
		wg.Add(1)
		populateChromosome(c, idx, t.Interests)
		wg.Done()
	}(&wg)
	go func(wg *sync.WaitGroup) {
		wg.Add(1)
		populateChromosome(c, idx, t.LifeEvents)
		wg.Done()
	}(&wg)

	wg.Wait()

	return c, nil
}

func (constructor *constructor) getAdSetTargeting(adSetID, accessToken string) (*geneticTargeting, error) {
	var (
		result = struct {
			Targeting *geneticTargeting       `json:"targeting"`
			Error     *internal.FacebookError `json:"error"`
		}{}
	)
	uV := url.Values{}
	uV.Add("access_token", accessToken)
	uV.Add("fields", "targeting")
	u := internal.SetURL(adSetID, uV)
	resp, err := constructor.client.Get(u)
	if err != nil {
		return nil, &logger.Error{
			Level:   "Panic",
			Message: "Unable to perform get request for an adset's targeting.",
			Err:     err,
		}
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, &logger.Error{
			Level:   "Panic",
			Message: "Unable to read response from an adset's targeting.",
			Err:     err,
		}
	}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, &logger.Error{
			Level:   "Panic",
			Message: "Unable to unmarshal response from an adset's targeting.",
			Err:     err,
		}
	}
	if result.Error != nil {
		return nil, &logger.Error{
			Level:   "Error",
			Message: "Response from request for an adset's targeting contains an error.",
			Err:     result.Error,
		}
	}

	return result.Targeting, nil
}

func populateChromosome(c *genetic.Chromosome, idx map[string]*genetic.Gene, target []targetingByType) {
	for _, t := range target {
		if idx[t.ID] != nil {
			idx[t.ID].Value = 1
		}
	}
}

// loadDefaultChromosome loads the value of an empty tree for a facebook targeting and
// creates a Chromosome
func (constructor *constructor) loadDefaultChromosome() (*genetic.Chromosome, map[string]*genetic.Gene, error) {
	var (
		defaultTree = struct {
			Data []targetingByType `json:"data"`
		}{}
		b   []byte
		err error
	)

	if !constructor.test {
		b, err = ioutil.ReadFile("targetingTree.json")
		if err != nil {
			return nil, nil, &logger.Error{
				Level:   "Panic",
				Message: "Unable to load data from default targeting tree.",
				Err:     err,
			}
		}
	} else {
		b, err = ioutil.ReadFile("tests-fixtures/targetingTree.json")
		if err != nil {
			return nil, nil, &logger.Error{
				Level:   "Panic",
				Message: "Unable to load data from default targeting tree.",
				Err:     err,
			}
		}
	}

	err = json.Unmarshal(b, &defaultTree)
	if err != nil {
		return nil, nil, &logger.Error{
			Level:   "Panic",
			Message: "Unable to unmarshal data from default targeting tree.",
			Err:     err,
		}
	}

	path := make(map[string]*genetic.Gene)
	root := &genetic.Gene{
		Name: defaultTree.Data[0].Name,
	}
	path[root.Name] = root
	c := &genetic.Chromosome{
		Root: root,
	}
	idx := make(map[string]*genetic.Gene)

	for i := 1; i < len(defaultTree.Data); i++ {
		g := &genetic.Gene{
			ID:   defaultTree.Data[i].ID,
			Name: defaultTree.Data[i].Name,
			Type: defaultTree.Data[i].Type,
		}
		if len(defaultTree.Data[i].Path) == 0 {
			root.Children = append(root.Children, g)
			path[defaultTree.Data[i].Name] = g
		} else if len(defaultTree.Data[i].Path) >= 1 && defaultTree.Data[i].ID == "" {
			path[defaultTree.Data[i].Name] = g
			parent := path[defaultTree.Data[i].Path[len(defaultTree.Data[i].Path)-1]]
			parent.Children = append(parent.Children, g)
		} else {
			idx[defaultTree.Data[i].ID] = g
			parent := path[defaultTree.Data[i].Path[len(defaultTree.Data[i].Path)-1]]
			parent.Children = append(parent.Children, g)
		}
	}

	return c, idx, nil
}
