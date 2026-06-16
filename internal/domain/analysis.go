package domain

// SpamCheckResult contient le résultat d'une analyse de score spam.
type SpamCheckResult struct {
	Rules    []SpamRuleResult `json:"rules"`
	MaxScore float32          `json:"max_score"`
	Score    float32          `json:"score"`
	Pass     bool             `json:"pass"`
}

// SpamRuleResult représente une règle spam déclenchée.
type SpamRuleResult struct {
	Description string  `json:"description"`
	Details     string  `json:"details,omitempty"`
	Name        string  `json:"name"`
	Score       float32 `json:"score"`
}

// SpamScoreAction représente l'action à effectuer quand le score spam dépasse le seuil.
type SpamScoreAction string

const (
	SpamScoreActionWarn  SpamScoreAction = "warn"
	SpamScoreActionBlock SpamScoreAction = "block"
)

// HTMLCheckResult contient le résultat d'une vérification de compatibilité HTML.
type HTMLCheckResult struct {
	Issues     []HTMLCompatIssue `json:"issues"`
	TotalCount int               `json:"total_count"`
}

// HTMLCompatIssue représente un problème de compatibilité avec un client mail.
type HTMLCompatIssue struct {
	Selector    string   `json:"selector"`
	Property    string   `json:"property"`
	Description string   `json:"description"`
	Severity    string   `json:"severity"` // "error", "warning"
	Clients     []string `json:"clients"`
}

// LinkCheckResult contient le résultat d'une vérification de liens.
type LinkCheckResult struct {
	Links       []LinkStatus `json:"links"`
	TotalCount  int          `json:"total_count"`
	BrokenCount int          `json:"broken_count"`
}

// LinkStatus représente le statut d'un lien vérifié.
type LinkStatus struct {
	Details    string `json:"details,omitempty"`
	Source     string `json:"source"` // "href" ou "src"
	Status     string `json:"status"` // "ok", "broken", "redirect", "insecure", "timeout", "invalid"
	URL        string `json:"url"`
	StatusCode int    `json:"status_code,omitempty"`
}
