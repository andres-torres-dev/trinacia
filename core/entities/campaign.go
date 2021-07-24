package entities

import "bitbucket.org/backend/core/genetic"

// Campaign object represent a Trinacia campaign
type Campaign struct {
	ID        string                `json:"id"`
	Budget    string                `json:"budget"`
	StartTime string                `json:"start_time"`
	EndTime   string                `json:"end_time"`
	Targeting []*genetic.Chromosome `json:"targeting"`
	Media     []Media               `json:"media"`
}

// Media used in the campaign
type Media struct {
	Title     string `json:"title,omitempty"`
	Body      string `json:"body,omitempty"`
	VideoID   string `json:"video_id,omitempty"`
	URL       string `json:"url,omitempty"`
	ImageHash string `json:"image_hash,omitempty"`
}
