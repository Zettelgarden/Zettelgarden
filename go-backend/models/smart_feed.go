package models

// SmartFeedScore represents the scoring breakdown for an article in the smart feed
type SmartFeedScore struct {
	ArticleID        int     `json:"article_id"`
	Score            float64 `json:"score"`
	VolumeScore      float64 `json:"volume_score"`
	InteractionBonus float64 `json:"interaction_bonus"`
	IsPriority       bool    `json:"is_priority"`
	Reason           string  `json:"reason"`
}

// RSSArticleWithScore extends RSSArticle with smart scoring
type RSSArticleWithScore struct {
	RSSArticle
	SmartScore *SmartFeedScore `json:"smart_score,omitempty"`
}
