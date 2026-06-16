package analysis

import "github.com/statoon54/mailhive/internal/domain"

const spamMaxScore float32 = 10.0

// SpamChecker analyse le contenu d'un email pour détecter des indicateurs de spam.
type SpamChecker struct {
	rules []SpamRule
}

// SpamRule est une règle de détection de spam.
type SpamRule interface {
	Name() string
	Description() string
	Check(subject, textBody, htmlBody string) (score float32, details string)
}

// NewSpamChecker crée un SpamChecker avec toutes les règles intégrées.
func NewSpamChecker() *SpamChecker {
	return &SpamChecker{
		rules: []SpamRule{
			&textHTMLRatioRule{},
			&spamKeywordsRule{},
			&excessiveCapsRule{},
			&suspiciousLinksRule{},
			&missingUnsubscribeRule{},
			&excessivePunctuationRule{},
			&allCapsSubjectRule{},
			&hiddenTextRule{},
		},
	}
}

// Check analyse le contenu email et retourne un SpamCheckResult.
func (sc *SpamChecker) Check(subject, textBody, htmlBody string) *domain.SpamCheckResult {
	result := &domain.SpamCheckResult{
		MaxScore: spamMaxScore,
	}

	var total float32
	for _, rule := range sc.rules {
		score, details := rule.Check(subject, textBody, htmlBody)
		if score > 0 {
			result.Rules = append(result.Rules, domain.SpamRuleResult{
				Name:        rule.Name(),
				Description: rule.Description(),
				Score:       score,
				Details:     details,
			})
			total += score
		}
	}

	if total > spamMaxScore {
		total = spamMaxScore
	}
	result.Score = total
	result.Pass = true

	return result
}

// CheckWithThreshold analyse et vérifie le seuil du tenant.
func (sc *SpamChecker) CheckWithThreshold(subject, textBody, htmlBody string, threshold *float32) *domain.SpamCheckResult {
	result := sc.Check(subject, textBody, htmlBody)
	if threshold != nil && result.Score > *threshold {
		result.Pass = false
	}
	return result
}
