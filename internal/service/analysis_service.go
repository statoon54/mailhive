package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/statoon54/mailhive/internal/analysis"
	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/port"
	"github.com/statoon54/mailhive/internal/templates"
)

// AnalysisService fournit les fonctionnalités d'analyse de templates.
type AnalysisService struct {
	templateRepo port.TemplateRepository
	spamChecker  *analysis.SpamChecker
	htmlChecker  *analysis.HTMLCompatChecker
	linkChecker  *analysis.LinkChecker
}

// NewAnalysisService crée un nouveau service d'analyse.
func NewAnalysisService(templateRepo port.TemplateRepository) *AnalysisService {
	return &AnalysisService{
		templateRepo: templateRepo,
		spamChecker:  analysis.NewSpamChecker(),
		htmlChecker:  analysis.NewHTMLCompatChecker(),
		linkChecker:  analysis.NewLinkChecker(),
	}
}

// renderTemplate charge et rend un template avec les données fournies.
func (s *AnalysisService) renderTemplate(
	ctx context.Context,
	tenantID, templateID uuid.UUID,
	data map[string]string,
) (subject, textBody, htmlBody string, err error) {
	tmpl, err := s.templateRepo.GetByID(ctx, tenantID, templateID)
	if err != nil {
		return "", "", "", err
	}

	if data == nil {
		data = make(map[string]string)
	}

	if err := templates.ValidateData(tmpl.Variables, data); err != nil {
		return "", "", "", fmt.Errorf("%w : %s", domain.ErrValidation, err.Error())
	}

	compiled, err := templates.Compile(tmpl.SubjectTmpl, tmpl.TextBody, tmpl.HTMLBody)
	if err != nil {
		return "", "", "", fmt.Errorf("erreur de compilation du template : %w", err)
	}

	subject, err = compiled.RenderSubject(data)
	if err != nil {
		return "", "", "", err
	}
	textBody, err = compiled.RenderText(data)
	if err != nil {
		return "", "", "", err
	}
	htmlBody, err = compiled.RenderHTML(data)
	if err != nil {
		return "", "", "", err
	}

	return subject, textBody, htmlBody, nil
}

// SpamCheck analyse un template rendu pour les indicateurs de spam.
func (s *AnalysisService) SpamCheck(
	ctx context.Context,
	tenantID, templateID uuid.UUID,
	data map[string]string,
) (*domain.SpamCheckResult, error) {
	subject, textBody, htmlBody, err := s.renderTemplate(ctx, tenantID, templateID, data)
	if err != nil {
		return nil, err
	}
	return s.spamChecker.Check(subject, textBody, htmlBody), nil
}

// HTMLCheck analyse un template rendu pour la compatibilité clients email.
func (s *AnalysisService) HTMLCheck(
	ctx context.Context,
	tenantID, templateID uuid.UUID,
	data map[string]string,
) (*domain.HTMLCheckResult, error) {
	_, _, htmlBody, err := s.renderTemplate(ctx, tenantID, templateID, data)
	if err != nil {
		return nil, err
	}
	return s.htmlChecker.Check(htmlBody), nil
}

// LinkCheck analyse un template rendu pour vérifier les liens.
func (s *AnalysisService) LinkCheck(
	ctx context.Context,
	tenantID, templateID uuid.UUID,
	data map[string]string,
) (*domain.LinkCheckResult, error) {
	_, _, htmlBody, err := s.renderTemplate(ctx, tenantID, templateID, data)
	if err != nil {
		return nil, err
	}
	return s.linkChecker.Check(ctx, htmlBody), nil
}

// ComputeSpamScore calcule le score spam pour un contenu brut.
func (s *AnalysisService) ComputeSpamScore(subject, textBody, htmlBody string) float32 {
	result := s.spamChecker.Check(subject, textBody, htmlBody)
	return result.Score
}
