package campaign

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"
	"time"

	"bitbucket.org/backend/core/facebook/internal"
	"bitbucket.org/backend/core/genetic"
	"bitbucket.org/backend/core/logger"
	"bitbucket.org/backend/core/server"
)

var (
	errMissingAccessToken = errors.New("The access token hasn't been set to the quality struct")
)

type q struct {
	client      server.Client
	accessToken string
}

func quality(config ...func(*q)) *q {
	q := &q{
		client: server.New(),
	}

	for _, fn := range config {
		fn(q)
	}

	return q
}

func (qu *q) compute(c *genetic.Chromosome) (float64, error) {
	if qu.accessToken == "" {
		return 0.0, errMissingAccessToken
	}

	const timeFormat = "2006-01-02"
	var (
		result = struct {
			Data []struct {
				Reach     string `json:"reach"`
				Cpm       string `json:"cpm"`
				UniqueCTR string `json:"unique_ctr"`
				DateStart string `json:"date_start"`
				DateStop  string `json:"date_stop"`
			} `json:"data"`
			Error *internal.FacebookError `json:"error"`
		}{}
	)

	uV := url.Values{}
	uV.Add("access_token", qu.accessToken)
	uV.Add("date_preset", "lifetime")
	uV.Add("fields", "reach,unique_ctr,cpm")
	u := internal.SetURL(fmt.Sprintf("%s/insights", c.ID), uV)

	resp, err := qu.client.Get(u)
	if err != nil {
		return 0.0, &logger.Error{
			Level:   "Panic",
			Message: "Unable to perform request to get quality of an adset.",
			Err:     err,
		}
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0.0, &logger.Error{
			Level:   "Panic",
			Message: "Unable to read response to get quality of an adset.",
			Err:     err,
		}
	}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return 0.0, &logger.Error{
			Level:   "Panic",
			Message: "Unable to unamarshal response to get a quality of an adset.",
			Err:     err,
		}
	}

	if result.Error != nil {
		return 0.0, &logger.Error{
			Level:   "Error",
			Message: "Response of an adset quality contained an error.",
			Err:     result.Error,
		}
	}

	if len(result.Data) == 0 {
		return 0.0, nil
	}

	var q float64

	for _, d := range result.Data {
		startTime, err := time.Parse(timeFormat, d.DateStart)
		if err != nil {
			return 0.0, &logger.Error{
				Level:   "Error",
				Message: "Unable to parse time of adset quality",
				Err:     err,
			}
		}
		endTime, err := time.Parse(timeFormat, d.DateStop)
		if err != nil {
			return 0.0, &logger.Error{
				Level:   "Error",
				Message: "Unable to parse time of adset quality",
				Err:     err,
			}
		}
		reach, err := strconv.ParseFloat(d.Reach, 64)
		if err != nil {
			return 0.0, &logger.Error{
				Level:   "Error",
				Message: "Unable to parse quality response data as a float.",
				Err:     err,
			}
		}
		uniqueCTR, err := strconv.ParseFloat(d.UniqueCTR, 64)
		if err != nil {
			return 0.0, &logger.Error{
				Level:   "Error",
				Message: "Unable to parse quality response data as a float.",
				Err:     err,
			}
		}
		cpm, err := strconv.ParseFloat(d.Cpm, 64)
		if err != nil {
			return 0.0, &logger.Error{
				Level:   "Error",
				Message: "Unable to parse quality response data as a float.",
				Err:     err,
			}
		}

		q += (reach * uniqueCTR / cpm) / (endTime.Sub(startTime).Hours() / 24)
	}

	q /= float64(len(result.Data))

	return q, nil
}
