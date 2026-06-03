package nexustribunal

import (
	tribunaladapters "cgwm/battle/internal/nexus_tribunal/adapters"
	tribunalmodels "cgwm/battle/internal/nexus_tribunal/models"
	tribunalprompts "cgwm/battle/internal/nexus_tribunal/prompts"
	"cgwm/battle/internal/service"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	phaseInvestigation = "investigation"
	phaseTestimony     = "testimony"
	phaseLive          = "live_trial"
	phaseVerdict       = "verdict"
	statusCreated      = "created"
	statusOpen         = "open"
	statusClosed       = "closed"
)

type TribunalCase struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	OwnerID             uint      `gorm:"index" json:"ownerId"`
	Title               string    `gorm:"size:180;not null" json:"title"`
	CaseType            string    `gorm:"size:80" json:"caseType"`
	Description         string    `gorm:"type:text" json:"description"`
	AccusationPosition  string    `gorm:"type:text" json:"accusationPosition"`
	DefensePosition     string    `gorm:"type:text" json:"defensePosition"`
	PlayerRole          string    `gorm:"size:40" json:"playerRole"`
	Mode                string    `gorm:"size:40;index" json:"mode"`
	Tone                string    `gorm:"size:80" json:"tone"`
	Visibility          string    `gorm:"size:40" json:"visibility"`
	ProviderType        string    `gorm:"size:80" json:"providerType"`
	ProviderModel       string    `gorm:"size:160" json:"providerModel"`
	ProviderIsLocal     bool      `json:"providerIsLocal"`
	LocalEndpoint       string    `gorm:"size:255" json:"localEndpoint"`
	ProviderStream      bool      `json:"providerStream"`
	APIKeyMode          string    `gorm:"size:80" json:"apiKeyMode"`
	JuryCount           int       `json:"juryCount"`
	EnableInvestigation bool      `json:"enableInvestigation"`
	EnableObjections    bool      `json:"enableObjections"`
	Status              string    `gorm:"size:40;index" json:"status"`
	CurrentPhase        string    `gorm:"size:60;index" json:"currentPhase"`
	DefenseScore        int       `json:"defenseScore"`
	AccusationScore     int       `json:"accusationScore"`
	Pressure            int       `json:"pressure"`
	Verdict             string    `gorm:"size:80" json:"verdict"`
	VerdictSummary      string    `gorm:"type:text" json:"verdictSummary"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type TribunalEvidence struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CaseID       uint      `gorm:"index" json:"caseId"`
	OwnerID      uint      `gorm:"index" json:"ownerId"`
	Title        string    `gorm:"size:180;not null" json:"title"`
	Description  string    `gorm:"type:text" json:"description"`
	EvidenceType string    `gorm:"size:80" json:"evidenceType"`
	SourceType   string    `gorm:"size:80" json:"sourceType"`
	SourceID     string    `gorm:"size:120" json:"sourceId"`
	Strength     int       `json:"strength"`
	Reliability  int       `json:"reliability"`
	Tags         string    `gorm:"size:255" json:"tags"`
	SupportsSide string    `gorm:"size:40" json:"supportsSide"`
	AssetID      string    `gorm:"size:120" json:"assetId"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type TribunalWitness struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CaseID      uint      `gorm:"index" json:"caseId"`
	OwnerID     uint      `gorm:"index" json:"ownerId"`
	Name        string    `gorm:"size:160;not null" json:"name"`
	Role        string    `gorm:"size:120" json:"role"`
	Personality string    `gorm:"size:255" json:"personality"`
	Credibility int       `json:"credibility"`
	Bias        string    `gorm:"size:120" json:"bias"`
	Knowledge   string    `gorm:"type:text" json:"knowledge"`
	Secrets     string    `gorm:"type:text" json:"-"`
	AssetID     string    `gorm:"size:120" json:"assetId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type TribunalStatement struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	CaseID             uint      `gorm:"index" json:"caseId"`
	WitnessID          uint      `gorm:"index" json:"witnessId"`
	OwnerID            uint      `gorm:"index" json:"ownerId"`
	Content            string    `gorm:"type:text" json:"content"`
	StatementIndex     int       `json:"statementIndex"`
	Tags               string    `gorm:"size:255" json:"tags"`
	TruthLevel         string    `gorm:"size:60" json:"truthLevel"`
	ContradictionHints string    `gorm:"type:text" json:"contradictionHints"`
	IsAttackable       bool      `json:"isAttackable"`
	PressureCount      int       `json:"pressureCount"`
	Status             string    `gorm:"size:60" json:"status"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type createCaseRequest struct {
	Title               string         `json:"title"`
	CaseType            string         `json:"caseType"`
	Description         string         `json:"description"`
	AccusationPosition  string         `json:"accusationPosition"`
	DefensePosition     string         `json:"defensePosition"`
	PlayerRole          string         `json:"playerRole"`
	Mode                string         `json:"mode"`
	Tone                string         `json:"tone"`
	Visibility          string         `json:"visibility"`
	EnableInvestigation bool           `json:"enableInvestigation"`
	EnableObjections    bool           `json:"enableObjections"`
	Provider            providerConfig `json:"provider"`
	JuryCount           int            `json:"juryCount"`
	NexusContext        map[string]any `json:"nexusContext"`
}

type providerConfig struct {
	ProviderType  string `json:"providerType"`
	Model         string `json:"model"`
	IsLocal       bool   `json:"isLocal"`
	LocalEndpoint string `json:"localEndpoint"`
	Stream        bool   `json:"stream"`
	APIKeyMode    string `json:"apiKeyMode"`
	APIKey        string `json:"apiKey"`
}

type evidenceRequest struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	EvidenceType string   `json:"evidenceType"`
	SourceType   string   `json:"sourceType"`
	SourceID     string   `json:"sourceId"`
	Strength     int      `json:"strength"`
	Reliability  int      `json:"reliability"`
	Tags         []string `json:"tags"`
	SupportsSide string   `json:"supportsSide"`
	AssetID      string   `json:"assetId"`
}

type witnessRequest struct {
	Name        string `json:"name"`
	Role        string `json:"role"`
	Personality string `json:"personality"`
	Credibility int    `json:"credibility"`
	Bias        string `json:"bias"`
	Knowledge   string `json:"knowledge"`
	Secrets     string `json:"secrets"`
	AssetID     string `json:"assetId"`
}

type statementActionRequest struct {
	StatementID uint   `json:"statementId"`
	EvidenceID  uint   `json:"evidenceId"`
	Argument    string `json:"argument"`
}

type providerTestRequest struct {
	ProviderType  string `json:"providerType"`
	Model         string `json:"model"`
	LocalEndpoint string `json:"localEndpoint"`
	Stream        bool   `json:"stream"`
	APIKeyMode    string `json:"apiKeyMode"`
	APIKey        string `json:"apiKey"`
	Prompt        string `json:"prompt"`
}

type providerOverrideRequest struct {
	Provider providerConfig `json:"provider"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// RegisterRoutes wires the autonomous Tribunal module. The caller can mount it
// once from the main router without leaking Tribunal handlers into router files.
func RegisterRoutes(router *gin.Engine, db *gorm.DB, authMiddleware gin.HandlerFunc, adminMiddleware gin.HandlerFunc) {
	_ = db.AutoMigrate(&TribunalCase{}, &TribunalEvidence{}, &TribunalWitness{}, &TribunalStatement{}, &tribunalmodels.TribunalGeneratedCase{}, &tribunalmodels.TribunalCaseGenerationBatch{})
	module := newModule(db)
	for _, prefix := range []string{"/api/nexus-tribunal", "/api/v1/nexus-tribunal"} {
		group := router.Group(prefix)
		group.Use(authMiddleware)
		module.mount(group)
		admin := group.Group("/admin")
		admin.Use(adminMiddleware)
		admin.GET("/debug", module.debug)
		admin.GET("/generated-cases/batches", module.listGenerationBatches)
		admin.POST("/generated-cases/generate-now", module.triggerGenerateNow)
	}
}

type module struct {
	db *gorm.DB
}

func newModule(db *gorm.DB) *module {
	return &module{db: db}
}

func (m *module) mount(group *gin.RouterGroup) {
	group.GET("/index", m.index)
	group.GET("/providers", m.providers)
	group.POST("/providers/test", m.testProvider)
	group.GET("/assets/manifest", m.assetManifest)
	group.POST("/cases", m.createCase)
	group.GET("/cases/:caseId", m.getCase)
	group.GET("/cases/:caseId/investigation", m.getInvestigation)
	group.POST("/cases/:caseId/investigation/generate", m.generateInvestigation)
	group.POST("/cases/:caseId/investigation/ready", m.markInvestigationReady)
	group.GET("/cases/:caseId/evidence", m.listEvidence)
	group.POST("/cases/:caseId/evidence", m.addEvidence)
	group.GET("/cases/:caseId/witnesses", m.listWitnesses)
	group.POST("/cases/:caseId/witnesses", m.addWitness)
	group.GET("/cases/:caseId/testimony/current", m.currentTestimony)
	group.POST("/cases/:caseId/testimony/generate", m.generateTestimony)
	group.POST("/cases/:caseId/press", m.pressStatement)
	group.POST("/cases/:caseId/objection", m.objectStatement)
	group.POST("/cases/:caseId/present-evidence", m.presentEvidence)
	group.POST("/cases/:caseId/ai-analysis", m.aiAnalysis)
	group.POST("/cases/:caseId/jury-vote", m.juryVote)
	group.POST("/cases/:caseId/verdict", m.verdict)
	group.GET("/archives", m.archives)
	// Generated cases (from PROMPT_TRIBUNAL_CASE_GENERATOR)
	group.GET("/generated-cases", m.listGeneratedCases)
	group.GET("/generated-cases/filters", m.generatedFilters)
	group.GET("/generated-cases/:genId", m.getGeneratedCase)
	group.POST("/generated-cases/:genId/load", m.loadGeneratedCase)
	// Admin/debug for generated (protected by adminMiddleware on /admin sub)
}

func (m *module) index(c *gin.Context) {
	ownerID := currentOwnerID(c)
	var openCases int64
	var closedCases int64
	var archivedCases int64
	m.db.Model(&TribunalCase{}).Where("owner_id = ? AND status <> ?", ownerID, statusClosed).Count(&openCases)
	m.db.Model(&TribunalCase{}).Where("owner_id = ? AND status = ?", ownerID, statusClosed).Count(&closedCases)
	archivedCases = closedCases
	var recent []TribunalCase
	_ = m.db.Where("owner_id = ?", ownerID).Order("updated_at desc").Limit(6).Find(&recent).Error
	respondOK(c, gin.H{
		"activeProvider": activeProviderSummary(),
		"availableModes": []string{"quick_trial", "full_case", "debate_only", "auto_play"},
		"recentCases":    recent,
		"stats": gin.H{
			"openCases":     openCases,
			"closedCases":   closedCases,
			"archivedCases": archivedCases,
		},
	})
}

func (m *module) providers(c *gin.Context) {
	items := make([]gin.H, 0)
	for _, item := range service.SupportedAIProviders() {
		keyEnv, modelEnv := providerEnvNames(item.Name)
		items = append(items, gin.H{
			"providerType":              item.Name,
			"providerName":              item.DisplayName,
			"aliases":                   item.Aliases,
			"chatCompletionsCompatible": item.ChatCompletionsCompatible,
			"isLocal":                   false,
			"configuredFromEnv":         strings.TrimSpace(os.Getenv(keyEnv)) != "",
			"defaultModel":              defaultModel(item.Name, os.Getenv(modelEnv)),
			"apiKeyMode":                "local_client_key",
		})
	}
	items = append(items,
		gin.H{"providerType": "ollama", "providerName": "Ollama", "isLocal": true, "status": "endpoint_required", "defaultModel": "llama3.2", "defaultEndpoint": "http://localhost:11434", "apiKeyMode": "no_key_local"},
		gin.H{"providerType": "lmstudio", "providerName": "LM Studio", "isLocal": true, "status": "endpoint_required", "defaultModel": "local-model", "defaultEndpoint": "http://localhost:1234", "apiKeyMode": "no_key_local"},
		gin.H{"providerType": "local", "providerName": "Local OpenAI-compatible", "isLocal": true, "status": "endpoint_required", "defaultModel": "local-model", "defaultEndpoint": "http://localhost:1234", "apiKeyMode": "optional_key"},
		gin.H{"providerType": "custom", "providerName": "Custom OpenAI-compatible", "isLocal": true, "status": "endpoint_required", "defaultModel": "local-model", "defaultEndpoint": "http://localhost:1234", "apiKeyMode": "optional_key"},
	)
	respondOK(c, gin.H{"providers": items})
}

func (m *module) testProvider(c *gin.Context) {
	var req providerTestRequest
	if err := bindJSON(c, &req); err != nil {
		respondErr(c, http.StatusBadRequest, "INVALID_PAYLOAD", "Payload provider invalide.", nil)
		return
	}
	result, err := testAIProvider(c.Request.Context(), req)
	if err != nil {
		respondErr(c, http.StatusBadGateway, "PROVIDER_UNAVAILABLE", err.Error(), gin.H{"providerType": normalizeProvider(req.ProviderType)})
		return
	}
	respondOK(c, result)
}

func (m *module) createCase(c *gin.Context) {
	var req createCaseRequest
	if err := bindJSON(c, &req); err != nil {
		respondErr(c, http.StatusBadRequest, "INVALID_PAYLOAD", "Payload de creation d'affaire invalide.", nil)
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		respondErr(c, http.StatusBadRequest, "TITLE_REQUIRED", "Le titre de l'affaire est obligatoire.", nil)
		return
	}
	mode := defaultText(req.Mode, "quick_trial")
	nextPhase := phaseInvestigation
	if mode == "quick_trial" || !req.EnableInvestigation {
		nextPhase = phaseLive
	}
	item := TribunalCase{
		OwnerID:             currentOwnerID(c),
		Title:               title,
		CaseType:            defaultText(req.CaseType, "civil"),
		Description:         strings.TrimSpace(req.Description),
		AccusationPosition:  strings.TrimSpace(req.AccusationPosition),
		DefensePosition:     strings.TrimSpace(req.DefensePosition),
		PlayerRole:          defaultText(req.PlayerRole, "defense"),
		Mode:                mode,
		Tone:                defaultText(req.Tone, "cyberpunk_serious"),
		Visibility:          defaultText(req.Visibility, "private"),
		ProviderType:        normalizeProvider(defaultText(req.Provider.ProviderType, "openai")),
		ProviderModel:       defaultText(req.Provider.Model, defaultModel(req.Provider.ProviderType, "")),
		ProviderIsLocal:     req.Provider.IsLocal,
		LocalEndpoint:       strings.TrimSpace(req.Provider.LocalEndpoint),
		ProviderStream:      req.Provider.Stream,
		APIKeyMode:          defaultText(req.Provider.APIKeyMode, "local_client_key"),
		JuryCount:           clamp(req.JuryCount, 3, 9),
		EnableInvestigation: req.EnableInvestigation,
		EnableObjections:    req.EnableObjections,
		Status:              statusCreated,
		CurrentPhase:        nextPhase,
		DefenseScore:        50,
		AccusationScore:     50,
		Pressure:            20,
	}
	if err := m.db.Create(&item).Error; err != nil {
		respondErr(c, http.StatusInternalServerError, "CASE_CREATE_FAILED", "Impossible de creer l'affaire.", nil)
		return
	}
	if mode == "quick_trial" {
		_ = m.seedMinimumDossier(c.Request.Context(), item)
	}
	respondOK(c, gin.H{"caseId": item.ID, "status": item.Status, "currentPhase": item.CurrentPhase, "nextScreen": nextScreen(item.CurrentPhase)})
}

func (m *module) getCase(c *gin.Context) {
	item, err := m.caseByParam(c)
	if err != nil {
		return
	}
	respondOK(c, m.casePayload(item))
}

func (m *module) getInvestigation(c *gin.Context) {
	item, err := m.caseByParam(c)
	if err != nil {
		return
	}
	evidence, witnesses := m.caseEvidenceWitnesses(item.ID, item.OwnerID)
	respondOK(c, gin.H{
		"case":              item,
		"objectives":        investigationObjectives(item),
		"evidence":          evidence,
		"witnesses":         witnesses,
		"locations":         []string{"Archive reseau", "Salle d'audience holographique", "Noeud de logs biométriques"},
		"accusationTheory":  defaultText(item.AccusationPosition, "L'accusation doit etablir une responsabilite claire."),
		"defenseTheory":     defaultText(item.DefensePosition, "La defense doit isoler les contradictions utiles."),
		"readyForTrial":     len(evidence) > 0 && len(witnesses) > 0,
		"currentPhase":      item.CurrentPhase,
		"recommendedAction": "generate",
	})
}

func (m *module) generateInvestigation(c *gin.Context) {
	item, err := m.caseByParam(c)
	if err != nil {
		return
	}
	if err := m.seedMinimumDossier(c.Request.Context(), item); err != nil {
		respondErr(c, http.StatusInternalServerError, "INVESTIGATION_GENERATE_FAILED", "Impossible de generer le dossier.", nil)
		return
	}
	item.Status = statusOpen
	item.CurrentPhase = phaseInvestigation
	_ = m.db.Save(&item).Error
	evidence, witnesses := m.caseEvidenceWitnesses(item.ID, item.OwnerID)
	respondOK(c, gin.H{"case": item, "evidence": evidence, "witnesses": witnesses, "readyForTrial": true})
}

func (m *module) markInvestigationReady(c *gin.Context) {
	item, err := m.caseByParam(c)
	if err != nil {
		return
	}
	item.Status = statusOpen
	item.CurrentPhase = phaseLive
	if err := m.db.Save(&item).Error; err != nil {
		respondErr(c, http.StatusInternalServerError, "CASE_UPDATE_FAILED", "Impossible de changer de phase.", nil)
		return
	}
	respondOK(c, gin.H{"caseId": item.ID, "currentPhase": item.CurrentPhase, "nextScreen": "live"})
}

func (m *module) listEvidence(c *gin.Context) {
	item, err := m.caseByParam(c)
	if err != nil {
		return
	}
	var evidence []TribunalEvidence
	_ = m.db.Where("case_id = ? AND owner_id = ?", item.ID, item.OwnerID).Order("id asc").Find(&evidence).Error
	respondOK(c, gin.H{"evidence": evidence})
}

func (m *module) addEvidence(c *gin.Context) {
	item, err := m.caseByParam(c)
	if err != nil {
		return
	}
	var req evidenceRequest
	if err := bindJSON(c, &req); err != nil || strings.TrimSpace(req.Title) == "" {
		respondErr(c, http.StatusBadRequest, "INVALID_EVIDENCE", "La preuve doit avoir un titre valide.", nil)
		return
	}
	evidence := TribunalEvidence{
		CaseID:       item.ID,
		OwnerID:      item.OwnerID,
		Title:        strings.TrimSpace(req.Title),
		Description:  strings.TrimSpace(req.Description),
		EvidenceType: defaultText(req.EvidenceType, "document"),
		SourceType:   defaultText(req.SourceType, "manual"),
		SourceID:     strings.TrimSpace(req.SourceID),
		Strength:     clamp(req.Strength, 1, 100),
		Reliability:  clamp(req.Reliability, 1, 100),
		Tags:         strings.Join(req.Tags, ","),
		SupportsSide: defaultText(req.SupportsSide, "neutral"),
		AssetID:      defaultText(req.AssetID, "tribunal.evidence.document"),
	}
	if err := m.db.Create(&evidence).Error; err != nil {
		respondErr(c, http.StatusInternalServerError, "EVIDENCE_CREATE_FAILED", "Impossible d'ajouter la preuve.", nil)
		return
	}
	respondOK(c, evidence)
}

func (m *module) listWitnesses(c *gin.Context) {
	item, err := m.caseByParam(c)
	if err != nil {
		return
	}
	var witnesses []TribunalWitness
	_ = m.db.Where("case_id = ? AND owner_id = ?", item.ID, item.OwnerID).Order("id asc").Find(&witnesses).Error
	respondOK(c, gin.H{"witnesses": witnesses})
}

func (m *module) addWitness(c *gin.Context) {
	item, err := m.caseByParam(c)
	if err != nil {
		return
	}
	var req witnessRequest
	if err := bindJSON(c, &req); err != nil || strings.TrimSpace(req.Name) == "" {
		respondErr(c, http.StatusBadRequest, "INVALID_WITNESS", "Le temoin doit avoir un nom valide.", nil)
		return
	}
	witness := TribunalWitness{
		CaseID:      item.ID,
		OwnerID:     item.OwnerID,
		Name:        strings.TrimSpace(req.Name),
		Role:        defaultText(req.Role, "temoin"),
		Personality: defaultText(req.Personality, "precis, prudent"),
		Credibility: clamp(req.Credibility, 1, 100),
		Bias:        defaultText(req.Bias, "neutral"),
		Knowledge:   strings.TrimSpace(req.Knowledge),
		Secrets:     strings.TrimSpace(req.Secrets),
		AssetID:     defaultText(req.AssetID, "tribunal.character.witness_default"),
	}
	if err := m.db.Create(&witness).Error; err != nil {
		respondErr(c, http.StatusInternalServerError, "WITNESS_CREATE_FAILED", "Impossible d'ajouter le temoin.", nil)
		return
	}
	respondOK(c, witness)
}

func (m *module) currentTestimony(c *gin.Context) {
	item, err := m.caseByParam(c)
	if err != nil {
		return
	}
	var statements []TribunalStatement
	_ = m.db.Where("case_id = ? AND owner_id = ?", item.ID, item.OwnerID).Order("statement_index asc").Find(&statements).Error
	if len(statements) == 0 {
		respondOK(c, gin.H{"case": item, "statements": []TribunalStatement{}, "activeStatement": nil})
		return
	}
	respondOK(c, gin.H{"case": item, "statements": statements, "activeStatement": statements[0]})
}

func (m *module) generateTestimony(c *gin.Context) {
	item, err := m.caseByParam(c)
	if err != nil {
		return
	}
	override := bindProviderOverride(c)
	var witness TribunalWitness
	if err := m.db.Where("case_id = ? AND owner_id = ?", item.ID, item.OwnerID).Order("id asc").First(&witness).Error; err != nil {
		respondErr(c, http.StatusBadRequest, "WITNESS_REQUIRED", "Ajoutez au moins un temoin avant le temoignage.", nil)
		return
	}
	if err := m.seedStatements(item, witness, override); err != nil {
		respondErr(c, http.StatusInternalServerError, "TESTIMONY_GENERATE_FAILED", "Impossible de generer le temoignage.", nil)
		return
	}
	item.CurrentPhase = phaseTestimony
	item.Status = statusOpen
	_ = m.db.Save(&item).Error
	var statements []TribunalStatement
	_ = m.db.Where("case_id = ? AND owner_id = ?", item.ID, item.OwnerID).Order("statement_index asc").Find(&statements).Error
	respondOK(c, gin.H{"case": item, "witness": witness, "statements": statements})
}

func (m *module) pressStatement(c *gin.Context) {
	item, statement, ok := m.statementAction(c)
	if !ok {
		return
	}
	statement.PressureCount++
	statement.Status = "pressed"
	_ = m.db.Save(&statement).Error
	item.Pressure = clamp(item.Pressure+5, 0, 100)
	_ = m.db.Save(&item).Error
	respondOK(c, gin.H{"case": item, "statement": statement, "response": "Le temoin precise sa phrase, mais expose davantage sa logique interne."})
}

func (m *module) objectStatement(c *gin.Context) {
	item, statement, ok := m.statementAction(c)
	if !ok {
		return
	}
	var req statementActionRequest
	_ = bindJSON(c, &req)
	var evidence TribunalEvidence
	if req.EvidenceID == 0 || m.db.Where("id = ? AND case_id = ? AND owner_id = ?", req.EvidenceID, item.ID, item.OwnerID).First(&evidence).Error != nil {
		item.Pressure = clamp(item.Pressure+8, 0, 100)
		item.DefenseScore = clamp(item.DefenseScore-3, 0, 100)
		_ = m.db.Save(&item).Error
		respondOK(c, gin.H{"accepted": false, "case": item, "statement": statement, "message": "Objection rejetee : aucune preuve valide n'a ete presentee."})
		return
	}
	statement.Status = "contradicted"
	_ = m.db.Save(&statement).Error
	item.DefenseScore = clamp(item.DefenseScore+8, 0, 100)
	item.AccusationScore = clamp(item.AccusationScore-5, 0, 100)
	item.Pressure = clamp(item.Pressure+4, 0, 100)
	_ = m.db.Save(&item).Error
	respondOK(c, gin.H{"accepted": true, "case": item, "statement": statement, "evidence": evidence, "message": "Contradiction acceptee : la preuve remet en cause la declaration."})
}

func (m *module) presentEvidence(c *gin.Context) {
	item, _, ok := m.statementAction(c)
	if !ok {
		return
	}
	var req statementActionRequest
	_ = bindJSON(c, &req)
	var evidence TribunalEvidence
	if req.EvidenceID == 0 || m.db.Where("id = ? AND case_id = ? AND owner_id = ?", req.EvidenceID, item.ID, item.OwnerID).First(&evidence).Error != nil {
		respondErr(c, http.StatusBadRequest, "EVIDENCE_REQUIRED", "Une preuve valide est obligatoire.", nil)
		return
	}
	impact := clamp(evidence.Strength/10, 1, 10)
	if evidence.SupportsSide == "defense" {
		item.DefenseScore = clamp(item.DefenseScore+impact, 0, 100)
	} else if evidence.SupportsSide == "accusation" {
		item.AccusationScore = clamp(item.AccusationScore+impact, 0, 100)
	}
	_ = m.db.Save(&item).Error
	respondOK(c, gin.H{"case": item, "evidence": evidence, "officialEffect": gin.H{"scoreImpact": impact}})
}

func (m *module) aiAnalysis(c *gin.Context) {
	item, err := m.caseByParam(c)
	if err != nil {
		return
	}
	override := bindProviderOverride(c)
	evidence, witnesses := m.caseEvidenceWitnesses(item.ID, item.OwnerID)

	// Try real AI advisory via adapter for richer analysis
	adapter := tribunaladapters.NewAIProviderAdapter(providerEnvKey)
	aiText := ""
	if resp, aerr := adapter.Generate(c.Request.Context(), tribunaladapters.GenerateRequest{
		ProviderType:  providerOverrideText(override.ProviderType, item.ProviderType),
		Model:         providerOverrideText(override.Model, item.ProviderModel),
		APIKey:        override.APIKey,
		LocalEndpoint: providerOverrideText(override.LocalEndpoint, item.LocalEndpoint),
		SystemPrompt:  "Tu es un analyste de tribunal IA cyberpunk. Donne 2-3 suggestions courtes et precises pour le joueur en fonction du contexte de l'affaire, des scores et des preuves.",
		Prompt:        fmt.Sprintf("Affaire: %s. Defense:%d Accusation:%d Pression:%d. %d preuves, %d temoins. Sujets: %s vs %s. Reponds en 2 bullets max.", item.Title, item.DefenseScore, item.AccusationScore, item.Pressure, len(evidence), len(witnesses), item.DefensePosition, item.AccusationPosition),
	}); aerr == nil {
		aiText = strings.TrimSpace(resp.Text)
	}
	if aiText == "" {
		aiText = fmt.Sprintf("Analyse consultative : %d preuve(s), %d temoin(s), pression %d. La defense doit cibler les phrases attaquables et l'accusation doit proteger les preuves fortes.", len(evidence), len(witnesses), item.Pressure)
	}

	respondOK(c, gin.H{
		"caseId":       item.ID,
		"analysis":     aiText,
		"advisoryOnly": true,
	})
}

func (m *module) juryVote(c *gin.Context) {
	item, err := m.caseByParam(c)
	if err != nil {
		return
	}
	count := clamp(item.JuryCount, 3, 9)
	votes := make([]gin.H, 0, count)
	for i := 1; i <= count; i++ {
		lean := "neutral"
		if item.DefenseScore > item.AccusationScore+i {
			lean = "defense"
		} else if item.AccusationScore > item.DefenseScore+i {
			lean = "accusation"
		}
		votes = append(votes, gin.H{"jurorId": i, "stance": lean, "confidence": clamp(45+i*6+abs(item.DefenseScore-item.AccusationScore)/2, 1, 100), "reason": "Vote calcule depuis les scores officiels et les contradictions validees."})
	}
	respondOK(c, gin.H{"caseId": item.ID, "votes": votes})
}

func (m *module) verdict(c *gin.Context) {
	item, err := m.caseByParam(c)
	if err != nil {
		return
	}
	if item.DefenseScore > item.AccusationScore+10 {
		item.Verdict = "innocent"
	} else if item.AccusationScore > item.DefenseScore+10 {
		item.Verdict = "guilty"
	} else {
		item.Verdict = "neutral"
	}
	item.VerdictSummary = fmt.Sprintf("Verdict %s. Score defense %d, accusation %d, pression %d.", item.Verdict, item.DefenseScore, item.AccusationScore, item.Pressure)
	item.CurrentPhase = phaseVerdict
	item.Status = statusClosed
	if err := m.db.Save(&item).Error; err != nil {
		respondErr(c, http.StatusInternalServerError, "VERDICT_FAILED", "Impossible d'enregistrer le verdict.", nil)
		return
	}
	respondOK(c, gin.H{
		"case":    item,
		"verdict": item.Verdict,
		"summary": item.VerdictSummary,
		"proposedConsequences": []gin.H{
			{"type": "create_lore_entry", "visibility": "private", "reason": "Verdict Tribunal archive."},
		},
	})
}

func (m *module) archives(c *gin.Context) {
	var cases []TribunalCase
	_ = m.db.Where("owner_id = ? AND status = ?", currentOwnerID(c), statusClosed).Order("updated_at desc").Limit(50).Find(&cases).Error
	respondOK(c, gin.H{"archives": cases})
}

func (m *module) debug(c *gin.Context) {
	respondOK(c, gin.H{"module": "nexus_tribunal", "status": "mounted", "serverTime": time.Now().UTC()})
}

func (m *module) assetManifest(c *gin.Context) {
	respondOK(c, gin.H{
		"version":  1,
		"fallback": "assets/nexus_games/tribunal/ui/tribunal_ui_placeholder.png",
		"assets":   tribunalAssetManifest(),
	})
}

func (m *module) casePayload(item TribunalCase) gin.H {
	evidence, witnesses := m.caseEvidenceWitnesses(item.ID, item.OwnerID)
	var statements []TribunalStatement
	_ = m.db.Where("case_id = ? AND owner_id = ?", item.ID, item.OwnerID).Order("statement_index asc").Find(&statements).Error
	return gin.H{"case": item, "evidence": evidence, "witnesses": witnesses, "statements": statements}
}

func (m *module) caseByParam(c *gin.Context) (TribunalCase, error) {
	id, err := strconv.ParseUint(c.Param("caseId"), 10, 64)
	if err != nil || id == 0 {
		respondErr(c, http.StatusBadRequest, "INVALID_CASE_ID", "Identifiant d'affaire invalide.", nil)
		return TribunalCase{}, errors.New("invalid case id")
	}
	var item TribunalCase
	if err := m.db.Where("id = ? AND owner_id = ?", uint(id), currentOwnerID(c)).First(&item).Error; err != nil {
		respondErr(c, http.StatusNotFound, "CASE_NOT_FOUND", "Affaire introuvable.", gin.H{"caseId": id})
		return TribunalCase{}, err
	}
	return item, nil
}

func (m *module) statementAction(c *gin.Context) (TribunalCase, TribunalStatement, bool) {
	item, err := m.caseByParam(c)
	if err != nil {
		return TribunalCase{}, TribunalStatement{}, false
	}
	var req statementActionRequest
	_ = bindJSON(c, &req)
	var statement TribunalStatement
	query := m.db.Where("case_id = ? AND owner_id = ?", item.ID, item.OwnerID).Order("statement_index asc")
	if req.StatementID != 0 {
		query = m.db.Where("id = ? AND case_id = ? AND owner_id = ?", req.StatementID, item.ID, item.OwnerID)
	}
	if err := query.First(&statement).Error; err != nil {
		respondErr(c, http.StatusBadRequest, "STATEMENT_REQUIRED", "Aucune phrase attaquable disponible.", nil)
		return item, TribunalStatement{}, false
	}
	return item, statement, true
}

func (m *module) caseEvidenceWitnesses(caseID uint, ownerID uint) ([]TribunalEvidence, []TribunalWitness) {
	var evidence []TribunalEvidence
	var witnesses []TribunalWitness
	_ = m.db.Where("case_id = ? AND owner_id = ?", caseID, ownerID).Order("id asc").Find(&evidence).Error
	_ = m.db.Where("case_id = ? AND owner_id = ?", caseID, ownerID).Order("id asc").Find(&witnesses).Error
	return evidence, witnesses
}

func (m *module) seedMinimumDossier(ctx context.Context, item TribunalCase) error {
	var count int64
	m.db.Model(&TribunalEvidence{}).Where("case_id = ? AND owner_id = ?", item.ID, item.OwnerID).Count(&count)
	if count == 0 {
		if err := m.db.WithContext(ctx).Create(&TribunalEvidence{
			CaseID: item.ID, OwnerID: item.OwnerID, Title: "Journal de coherence Nexus",
			Description:  "Trace serveur horodatee permettant de verifier une contradiction temporelle.",
			EvidenceType: "surveillance_log", SourceType: "system", Strength: 72, Reliability: 86,
			Tags: "time,log,contradiction", SupportsSide: "defense", AssetID: "tribunal.evidence.surveillance_log",
		}).Error; err != nil {
			return err
		}
	}
	m.db.Model(&TribunalWitness{}).Where("case_id = ? AND owner_id = ?", item.ID, item.OwnerID).Count(&count)
	if count == 0 {
		if err := m.db.WithContext(ctx).Create(&TribunalWitness{
			CaseID: item.ID, OwnerID: item.OwnerID, Name: "Temoin protocol-7", Role: "operateur reseau",
			Personality: "methodique, evasif sous pression", Credibility: 68, Bias: "institutional",
			Knowledge: "A observe les logs et les decisions automatisees liees a l'affaire.",
			AssetID:   "tribunal.character.witness_default",
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (m *module) seedStatements(item TribunalCase, witness TribunalWitness, override providerConfig) error {
	var count int64
	m.db.Model(&TribunalStatement{}).Where("case_id = ? AND owner_id = ?", item.ID, item.OwnerID).Count(&count)
	if count > 0 {
		return nil
	}

	// Attempt real AI generation via adapter for dynamic, case-specific testimony
	adapter := tribunaladapters.NewAIProviderAdapter(providerEnvKey)
	aiPrompt := fmt.Sprintf("Genere 4 phrases de temoignage courtes (1 ligne chacune), attackables, pour un temoin '%s' (%s) dans l'affaire '%s'. Contexte: accusation '%s', defense '%s'. Retourne uniquement les 4 lignes separees par | sans numerotation ni explication.", witness.Name, witness.Role, item.Title, item.AccusationPosition, item.DefensePosition)
	lines := []string{
		"J'ai consulte les journaux avant que le protocole ne verrouille le dossier.",
		"L'horodatage principal indiquait une sequence stable, sans anomalie apparente.",
		"Je n'ai vu aucun acces externe pendant la fenetre critique.",
		"Si contradiction il y a, elle vient probablement d'un journal secondaire.",
	}
	if resp, aerr := adapter.Generate(context.Background(), tribunaladapters.GenerateRequest{
		ProviderType:  providerOverrideText(override.ProviderType, item.ProviderType),
		Model:         providerOverrideText(override.Model, item.ProviderModel),
		APIKey:        override.APIKey,
		LocalEndpoint: providerOverrideText(override.LocalEndpoint, item.LocalEndpoint),
		SystemPrompt:  "Tu es un greffier IA qui produit des declarations de temoin coherentes et contradictoires potentielles pour un tribunal cyberpunk. Reponds strictement au format demande.",
		Prompt:        aiPrompt,
	}); aerr == nil && strings.TrimSpace(resp.Text) != "" {
		parts := strings.Split(strings.TrimSpace(resp.Text), "|")
		if len(parts) >= 3 {
			lines = []string{}
			for _, p := range parts {
				if t := strings.TrimSpace(p); t != "" {
					lines = append(lines, t)
				}
			}
		}
	}

	for index, line := range lines {
		statement := TribunalStatement{
			CaseID: item.ID, OwnerID: item.OwnerID, WitnessID: witness.ID, Content: line,
			StatementIndex: index + 1, Tags: "log,time", TruthLevel: "partial",
			ContradictionHints: "Comparer avec les preuves de type surveillance_log ou biometric_log.",
			IsAttackable:       index >= 1, Status: "active",
		}
		if err := m.db.Create(&statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func bindProviderOverride(c *gin.Context) providerConfig {
	var req providerOverrideRequest
	_ = bindJSON(c, &req)
	req.Provider.ProviderType = normalizeProvider(req.Provider.ProviderType)
	req.Provider.Model = strings.TrimSpace(req.Provider.Model)
	req.Provider.LocalEndpoint = strings.TrimSpace(req.Provider.LocalEndpoint)
	req.Provider.APIKey = strings.TrimSpace(req.Provider.APIKey)
	return req.Provider
}

func providerOverrideText(override string, fallback string) string {
	if strings.TrimSpace(override) != "" {
		return strings.TrimSpace(override)
	}
	return fallback
}

func testAIProvider(ctx context.Context, req providerTestRequest) (gin.H, error) {
	providerType := normalizeProvider(req.ProviderType)
	model := defaultText(req.Model, tribunaladapters.DefaultModelForProvider(providerType))
	adapter := tribunaladapters.NewAIProviderAdapter(providerEnvKey)
	response, err := adapter.Generate(ctx, tribunaladapters.GenerateRequest{
		ProviderType:  providerType,
		Model:         model,
		APIKey:        req.APIKey,
		LocalEndpoint: req.LocalEndpoint,
		SystemPrompt:  "Tu verifies une configuration technique. Reponds tres court.",
		Prompt:        defaultText(req.Prompt, "Reponds uniquement OK"),
	})
	if err != nil {
		return nil, err
	}
	return gin.H{"status": "connected", "latencyMs": response.LatencyMs, "modelAvailable": true, "message": truncate(response.Text, 180)}, nil
}

func bindJSON(c *gin.Context, target any) error {
	if c.Request.Body == nil {
		return nil
	}
	return c.ShouldBindJSON(target)
}

func respondOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data, "meta": gin.H{"requestId": requestID(c), "serverTime": time.Now().UTC().Format(time.RFC3339)}})
}

func respondErr(c *gin.Context, status int, code string, message string, details any) {
	c.JSON(status, gin.H{"success": false, "error": gin.H{"code": code, "message": message, "details": details, "requestId": requestID(c)}})
}

func requestID(c *gin.Context) string {
	if value := strings.TrimSpace(c.GetHeader("X-Request-Id")); value != "" {
		return value
	}
	return fmt.Sprintf("tribunal-%d", time.Now().UnixNano())
}

func currentOwnerID(c *gin.Context) uint {
	if value, ok := c.Get("auth.user_id"); ok {
		if id, ok := value.(uint); ok {
			return id
		}
	}
	return 0
}

func activeProviderSummary() gin.H {
	providerType := defaultText(os.Getenv("TRIBUNAL_AI_PROVIDER"), defaultText(os.Getenv("WORLD_AI_PRIMARY_PROVIDER"), "openai"))
	return gin.H{
		"providerType": providerType,
		"providerName": displayProvider(providerType),
		"model":        defaultModel(providerType, os.Getenv(providerModelEnv(providerType))),
		"isLocal":      false,
		"status":       configuredStatus(providerType),
		"apiKeyMode":   "local_client_key",
	}
}

func providerEnvKey(providerType string) string {
	keyEnv, _ := providerEnvNames(providerType)
	return strings.TrimSpace(os.Getenv(keyEnv))
}

func providerModelEnv(providerType string) string {
	_, modelEnv := providerEnvNames(providerType)
	return modelEnv
}

func providerEnvNames(providerType string) (string, string) {
	switch normalizeProvider(providerType) {
	case "mistral":
		return "MISTRAL_AI_KEY", "MISTRAL_AI_MODEL"
	case "claude", "anthropic":
		return "ANTHROPIC_AI_KEY", "ANTHROPIC_AI_MODEL"
	case "gemini", "google", "google_ai", "google-ai":
		return "GEMINI_AI_KEY", "GEMINI_AI_MODEL"
	case "xia", "xai", "x-ai":
		return "XAI_AI_KEY", "XAI_AI_MODEL"
	case "openrouter", "open_router":
		return "OPENROUTER_AI_KEY", "OPENROUTER_AI_MODEL"
	default:
		return "OPEN_AI_KEY", "OPEN_AI_MODEL"
	}
}

func configuredStatus(providerType string) string {
	if providerEnvKey(providerType) != "" {
		return "configured"
	}
	return "client_key_required"
}

func displayProvider(providerType string) string {
	for _, item := range service.SupportedAIProviders() {
		if normalizeProvider(item.Name) == normalizeProvider(providerType) {
			return item.DisplayName
		}
	}
	return providerType
}

func defaultModel(providerType string, configured string) string {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimSpace(configured)
	}
	switch normalizeProvider(providerType) {
	case "mistral":
		return "mistral-large-latest"
	case "openrouter", "open_router":
		return "openai/gpt-4o-mini"
	case "xia", "xai", "x-ai":
		return "grok-3-mini"
	case "claude", "anthropic":
		return "claude-sonnet-4-20250514"
	case "gemini", "google", "google_ai", "google-ai":
		return "gemini-3.5-flash"
	default:
		return "gpt-4o-mini"
	}
}

func normalizeProvider(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func defaultText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func clamp(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func nextScreen(phase string) string {
	switch phase {
	case phaseInvestigation:
		return "investigation"
	case phaseVerdict:
		return "verdict"
	default:
		return "live"
	}
}

func investigationObjectives(item TribunalCase) []string {
	return []string{
		"Identifier au moins une preuve exploitable.",
		"Ajouter ou verifier un temoin.",
		fmt.Sprintf("Preparer une ligne %s avant le proces live.", defaultText(item.PlayerRole, "defense")),
	}
}

func tribunalAssetManifest() []gin.H {
	return []gin.H{
		assetRecord("tribunal.background.main", "background", "Fond principal Tribunal", "assets/nexus_games/tribunal/backgrounds/tribunal_background_main.webp"),
		assetRecord("tribunal.background.courtroom", "background", "Salle d'audience", "assets/nexus_games/tribunal/backgrounds/tribunal_background_courtroom.webp"),
		assetRecord("tribunal.background.investigation", "background", "Enquete", "assets/nexus_games/tribunal/backgrounds/tribunal_background_investigation.webp"),
		assetRecord("tribunal.character.judge_ai", "character", "Juge IA", "assets/nexus_games/tribunal/characters/tribunal_character_judge_ai.png"),
		assetRecord("tribunal.character.prosecutor_ai", "character", "Accusation IA", "assets/nexus_games/tribunal/characters/tribunal_character_prosecutor_ai.png"),
		assetRecord("tribunal.character.defense_ai", "character", "Defense IA", "assets/nexus_games/tribunal/characters/tribunal_character_defense_ai.png"),
		assetRecord("tribunal.character.witness_default", "character", "Temoin par defaut", "assets/nexus_games/tribunal/characters/tribunal_character_witness_default.png"),
		assetRecord("tribunal.character.clerk_ai", "character", "Greffier IA", "assets/nexus_games/tribunal/characters/tribunal_character_clerk_ai.png"),
		assetRecord("tribunal.action.objection", "action", "Objection", "assets/nexus_games/tribunal/actions/tribunal_action_objection.png"),
		assetRecord("tribunal.action.press", "action", "Appuyer", "assets/nexus_games/tribunal/actions/tribunal_action_press.png"),
		assetRecord("tribunal.action.present_evidence", "action", "Presenter une preuve", "assets/nexus_games/tribunal/actions/tribunal_action_present_evidence.png"),
		assetRecord("tribunal.action.ai_analysis", "action", "Analyse IA", "assets/nexus_games/tribunal/actions/tribunal_action_ai_analysis.png"),
		assetRecord("tribunal.evidence.document", "evidence", "Document", "assets/nexus_games/tribunal/evidence/tribunal_evidence_document.png"),
		assetRecord("tribunal.evidence.surveillance_log", "evidence", "Log surveillance", "assets/nexus_games/tribunal/evidence/tribunal_evidence_surveillance_log.png"),
		assetRecord("tribunal.evidence.biometric_log", "evidence", "Log biometrique", "assets/nexus_games/tribunal/evidence/tribunal_evidence_biometric_log.png"),
		assetRecord("tribunal.ui.dialog_panel", "ui", "Panneau dialogue", "assets/nexus_games/tribunal/ui/tribunal_ui_dialog_panel.png"),
		assetRecord("tribunal.ui.evidence_panel", "ui", "Panneau preuves", "assets/nexus_games/tribunal/ui/tribunal_ui_evidence_panel.png"),
		assetRecord("tribunal.ui.placeholder", "ui", "Placeholder", "assets/nexus_games/tribunal/ui/tribunal_ui_placeholder.png"),
		assetRecord("tribunal.gauge.pressure", "gauge", "Pression tribunal", "assets/nexus_games/tribunal/gauges/tribunal_gauge_pressure.png"),
		assetRecord("tribunal.verdict.guilty", "verdict", "Coupable", "assets/nexus_games/tribunal/verdicts/tribunal_verdict_guilty.png"),
		assetRecord("tribunal.verdict.innocent", "verdict", "Innocent", "assets/nexus_games/tribunal/verdicts/tribunal_verdict_innocent.png"),
		assetRecord("tribunal.verdict.neutral", "verdict", "Neutre", "assets/nexus_games/tribunal/verdicts/tribunal_verdict_neutral.png"),
	}
}

func assetRecord(id string, category string, name string, path string) gin.H {
	return gin.H{
		"id":        id,
		"type":      "tribunal",
		"category":  category,
		"name":      name,
		"path":      path,
		"fallback":  "assets/nexus_games/tribunal/ui/tribunal_ui_placeholder.png",
		"tags":      []string{"tribunal", category},
		"isPremium": false,
		"isRemote":  false,
	}
}

// =============================================================================
// GENERATED CASES HANDLERS (PROMPT_TRIBUNAL_CASE_GENERATOR_FLUTTER_GO)
// =============================================================================

func (m *module) listGeneratedCases(c *gin.Context) {
	ownerID := currentOwnerID(c) // not strictly owner for generated (public templates), but keep for future
	_ = ownerID
	q := m.db.Model(&tribunalmodels.TribunalGeneratedCase{}).Where("is_published = ? AND status IN ?", true, []string{"ready", "published"})
	if lvl := c.Query("level"); lvl != "" {
		if n, _ := strconv.Atoi(lvl); n > 0 {
			q = q.Where("level = ?", n)
		}
	}
	if diff := c.Query("difficulty"); diff != "" {
		q = q.Where("difficulty = ?", diff)
	}
	if ct := c.Query("type"); ct != "" {
		q = q.Where("case_type = ?", ct)
	}
	if mode := c.Query("mode"); mode != "" {
		q = q.Where("mode = ?", mode)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if tag := c.Query("tag"); tag != "" {
		q = q.Where("JSON_CONTAINS(tags_json, ?)", `"`+tag+`"`)
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		q = q.Where("title LIKE ? OR summary LIKE ?", like, like)
	}
	page := 1
	if p, _ := strconv.Atoi(c.Query("page")); p > 0 {
		page = p
	}
	limit := 20
	if l, _ := strconv.Atoi(c.Query("limit")); l > 0 && l <= 100 {
		limit = l
	}
	offset := (page - 1) * limit

	var total int64
	q.Count(&total)

	var items []tribunalmodels.TribunalGeneratedCase
	if err := q.Order("level asc, created_at desc").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		respondErr(c, http.StatusInternalServerError, "DB_ERROR", "Erreur lecture affaires generees.", nil)
		return
	}

	// light projection for list
	data := make([]gin.H, 0, len(items))
	for _, it := range items {
		var tags []string
		_ = json.Unmarshal(it.TagsJSON, &tags)
		data = append(data, gin.H{
			"id":                       it.ID,
			"title":                    it.Title,
			"summary":                  it.Summary,
			"caseType":                 it.CaseType,
			"level":                    it.Level,
			"difficulty":               it.Difficulty,
			"estimatedDurationMinutes": it.EstimatedDurationMinutes,
			"mode":                     it.Mode,
			"tone":                     it.Tone,
			"playerRoleSuggestion":     it.PlayerRoleSuggestion,
			"tags":                     tags,
			"isPlayable":               it.IsPlayable,
			"isPublished":              it.IsPublished,
			"createdAt":                it.CreatedAt,
			"providerType":             it.ProviderType,
			"model":                    it.Model,
		})
	}
	respondOK(c, gin.H{
		"success": true,
		"data":    data,
		"pagination": gin.H{
			"page":    page,
			"limit":   limit,
			"total":   total,
			"hasNext": int64(offset+limit) < total,
		},
	})
}

func (m *module) generatedFilters(c *gin.Context) {
	respondOK(c, gin.H{
		"success": true,
		"data": gin.H{
			"levels":       []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			"difficulties": []string{"initiation", "easy", "standard", "intermediate", "confirmed", "hard", "expert", "master", "legendary", "nexus"},
			"types":        []string{"moral", "political", "guild_conflict", "player_conflict", "world_event", "quest_consequence", "roleplay", "absurd", "custom"},
			"modes":        []string{"quick_trial", "full_case", "debate_only", "auto_play", "nexus_integrated"},
			"statuses":     []string{"published", "ready", "archived"},
			"tags":         []string{"ia", "ville", "guilde", "preuve", "trahison", "liberte", "nexus"},
		},
	})
}

func (m *module) getGeneratedCase(c *gin.Context) {
	genID, _ := strconv.Atoi(c.Param("genId"))
	var g tribunalmodels.TribunalGeneratedCase
	if err := m.db.First(&g, genID).Error; err != nil {
		respondErr(c, http.StatusNotFound, "NOT_FOUND", "Affaire generee introuvable.", nil)
		return
	}
	var tags, witnesses, evidence, testimony, contradictions []any
	_ = json.Unmarshal(g.TagsJSON, &tags)
	_ = json.Unmarshal(g.WitnessesJSON, &witnesses)
	_ = json.Unmarshal(g.EvidenceJSON, &evidence)
	_ = json.Unmarshal(g.TestimonyJSON, &testimony)
	_ = json.Unmarshal(g.ExpectedContradictionsJSON, &contradictions)
	respondOK(c, gin.H{
		"success": true,
		"data": gin.H{
			"id":                       g.ID,
			"title":                    g.Title,
			"summary":                  g.Summary,
			"caseType":                 g.CaseType,
			"level":                    g.Level,
			"difficulty":               g.Difficulty,
			"estimatedDurationMinutes": g.EstimatedDurationMinutes,
			"mode":                     g.Mode,
			"tone":                     g.Tone,
			"playerRoleSuggestion":     g.PlayerRoleSuggestion,
			"accusationPosition":       g.AccusationPosition,
			"defensePosition":          g.DefensePosition,
			"tags":                     tags,
			"witnesses":                witnesses,
			"evidence":                 evidence,
			"testimonyStatements":      testimony,
			"expectedContradictions":   contradictions,
			"status":                   g.Status,
			"isPlayable":               g.IsPlayable,
			"isPublished":              g.IsPublished,
			"generatedByCron":          g.GeneratedByCron,
			"providerType":             g.ProviderType,
			"model":                    g.Model,
			"createdAt":                g.CreatedAt,
		},
	})
}

func (m *module) loadGeneratedCase(c *gin.Context) {
	genID, _ := strconv.Atoi(c.Param("genId"))
	var g tribunalmodels.TribunalGeneratedCase
	if err := m.db.First(&g, genID).Error; err != nil {
		respondErr(c, http.StatusNotFound, "NOT_FOUND", "Affaire generee introuvable.", nil)
		return
	}
	if !g.IsPlayable || g.Status == "archived" || g.Status == "rejected" {
		respondErr(c, http.StatusBadRequest, "NOT_PLAYABLE", "Cette affaire n'est plus chargeable.", nil)
		return
	}

	ownerID := currentOwnerID(c)
	var req struct {
		PlayerRole string         `json:"playerRole"`
		Provider   providerConfig `json:"provider"`
	}
	_ = bindJSON(c, &req)
	if req.PlayerRole == "" {
		req.PlayerRole = g.PlayerRoleSuggestion
	}

	// Create real playable TribunalCase
	tc := TribunalCase{
		OwnerID:            ownerID,
		Title:              g.Title,
		CaseType:           g.CaseType,
		Description:        g.Summary,
		AccusationPosition: g.AccusationPosition,
		DefensePosition:    g.DefensePosition,
		PlayerRole:         defaultText(req.PlayerRole, "neutral"),
		Mode:               defaultText(g.Mode, "full_case"),
		Tone:               defaultText(g.Tone, "cyberpunk_serious"),
		ProviderType:       defaultText(req.Provider.ProviderType, g.ProviderType),
		ProviderModel:      defaultText(req.Provider.Model, g.ProviderModel),
		Status:             statusOpen,
		CurrentPhase:       phaseInvestigation,
		DefenseScore:       50,
		AccusationScore:    50,
		Pressure:           10,
		JuryCount:          5,
		EnableInvestigation: true,
		EnableObjections:    true,
	}
	if err := m.db.Create(&tc).Error; err != nil {
		respondErr(c, http.StatusInternalServerError, "CASE_CREATE_FAILED", "Impossible de charger l'affaire.", nil)
		return
	}

	// Copy evidences
	var evs []tribunalmodels.TribunalGeneratedCase // reuse for unmarshal only
	_ = json.Unmarshal(g.EvidenceJSON, &evs) // actually array of maps
	var evList []map[string]any
	_ = json.Unmarshal(g.EvidenceJSON, &evList)
	for _, e := range evList {
		ev := TribunalEvidence{
			CaseID:       tc.ID,
			OwnerID:      ownerID,
			Title:        fmt.Sprint(e["title"]),
			Description:  fmt.Sprint(e["description"]),
			EvidenceType: defaultText(fmt.Sprint(e["evidenceType"]), "document"),
			Strength:     intFromAny(e["strength"], 60),
			Reliability:  intFromAny(e["reliability"], 70),
			Tags:         strings.Join(stringSliceFromAny(e["tags"]), ","),
			SupportsSide: defaultText(fmt.Sprint(e["supportsSide"]), "neutral"),
			AssetID:      "tribunal.evidence.document",
		}
		_ = m.db.Create(&ev).Error
	}

	// Copy witnesses + their statements
	var witList []map[string]any
	_ = json.Unmarshal(g.WitnessesJSON, &witList)
	var stmtList []map[string]any
	_ = json.Unmarshal(g.TestimonyJSON, &stmtList)
	for _, w := range witList {
		wit := TribunalWitness{
			CaseID:      tc.ID,
			OwnerID:     ownerID,
			Name:        fmt.Sprint(w["name"]),
			Role:        defaultText(fmt.Sprint(w["role"]), "temoin"),
			Credibility: intFromAny(w["credibility"], 65),
			Bias:        defaultText(fmt.Sprint(w["bias"]), "neutral"),
			Personality: defaultText(fmt.Sprint(w["personality"]), "precis"),
			Knowledge:   defaultText(fmt.Sprint(w["knowledge"]), ""),
			AssetID:     "tribunal.character.witness_default",
		}
		if err := m.db.Create(&wit).Error; err != nil {
			continue
		}
		// attach matching statements
		for _, s := range stmtList {
			if fmt.Sprint(s["witnessName"]) == wit.Name || strings.Contains(wit.Name, fmt.Sprint(s["witnessName"])) {
				st := TribunalStatement{
					CaseID:         tc.ID,
					WitnessID:      wit.ID,
					OwnerID:        ownerID,
					Content:        fmt.Sprint(s["content"]),
					StatementIndex: len(stmtList), // rough
					Tags:           strings.Join(stringSliceFromAny(s["tags"]), ","),
					IsAttackable:   boolFromAny(s["isAttackable"], true),
					TruthLevel:     "partial",
					Status:         "active",
				}
				_ = m.db.Create(&st).Error
			}
		}
	}

	// link back
	g.CaseID = &tc.ID
	_ = m.db.Save(&g).Error

	respondOK(c, gin.H{
		"success":       true,
		"data":          gin.H{"caseId": tc.ID, "generatedCaseId": g.ID, "status": "created", "currentPhase": tc.CurrentPhase, "nextScreen": "investigation"},
		"message":       "Affaire chargee avec succes.",
	})
}

func (m *module) listGenerationBatches(c *gin.Context) {
	var batches []tribunalmodels.TribunalCaseGenerationBatch
	_ = m.db.Order("created_at desc").Limit(20).Find(&batches).Error
	respondOK(c, gin.H{"success": true, "data": batches})
}

func (m *module) triggerGenerateNow(c *gin.Context) {
	var req struct {
		Provider string `form:"provider" json:"provider"`
		Model    string `form:"model" json:"model"`
		APIKey   string `form:"api_key" json:"apiKey"`
		Count    int    `form:"count" json:"count"`
	}
	ct := c.ContentType()
	if strings.Contains(ct, "json") {
		_ = c.ShouldBindJSON(&req)
	} else {
		_ = c.ShouldBind(&req)
	}
	count := req.Count
	if count <= 0 || count > 20 {
		count = 10
	}
	providerType := normalizeProvider(req.Provider)
	model := defaultText(req.Model, tribunaladapters.DefaultModelForProvider(providerType))
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		respondErr(c, http.StatusBadRequest, "API_KEY_REQUIRED", "Clé API temporaire requise pour la génération manuelle.", nil)
		return
	}

	batchID, generated, err := ManualGenerateTribunalCases(m.db, providerType, model, apiKey, count)
	if err != nil {
		respondErr(c, http.StatusBadGateway, "GENERATE_FAILED", err.Error(), nil)
		return
	}

	respondOK(c, gin.H{
		"success":   true,
		"batchId":   batchID,
		"generated": generated,
		"message":   fmt.Sprintf("%d affaires Tribunal générées manuellement.", generated),
	})
}

// ManualGenerateTribunalCases is the shared implementation for manual generation
// (used by the Tribunal admin page via /admin/generate/tribunal and by the internal trigger).
func ManualGenerateTribunalCases(db *gorm.DB, providerType, model, apiKey string, count int) (batchID uint, generated int, err error) {
	if count <= 0 || count > 20 {
		count = 10
	}

	batch := tribunalmodels.TribunalCaseGenerationBatch{
		StartedAt:      time.Now(),
		Source:         "admin_manual",
		TriggerType:    "manual",
		Status:         "running",
		RequestedCount: count,
		ProviderType:   providerType,
		ProviderModel:  model,
	}
	if err := db.Create(&batch).Error; err != nil {
		return 0, 0, fmt.Errorf("batch create: %w", err)
	}

	sys, userPrompt := tribunalprompts.BuildGeneratedCasesPrompt()
	adapter := tribunaladapters.NewAIProviderAdapter(func(pt string) string { return "" }) // we pass explicit key
	resp, aerr := adapter.Generate(context.Background(), tribunaladapters.GenerateRequest{
		ProviderType: providerType,
		Model:        model,
		APIKey:       apiKey,
		SystemPrompt: sys,
		Prompt:       userPrompt,
	})
	if aerr != nil {
		batch.Status = "failed"
		batch.ErrorMessage = aerr.Error()
		db.Save(&batch)
		return batch.ID, 0, aerr
	}

	cleaned := cleanJSONLocal(resp.Text)
	var wrapper struct {
		Cases []map[string]any `json:"cases"`
	}
	if uerr := json.Unmarshal([]byte(cleaned), &wrapper); uerr != nil {
		var direct []map[string]any
		if uerr2 := json.Unmarshal([]byte(cleaned), &direct); uerr2 == nil {
			wrapper.Cases = direct
		}
	}

	generated = 0
	for i, raw := range wrapper.Cases {
		if i >= count {
			break
		}
		title := strings.TrimSpace(fmt.Sprint(raw["title"]))
		if title == "" {
			continue
		}
		tagsB, _ := json.Marshal(raw["tags"])
		witB, _ := json.Marshal(raw["witnesses"])
		evB, _ := json.Marshal(raw["evidence"])
		testB, _ := json.Marshal(raw["testimonyStatements"])
		contrB, _ := json.Marshal(raw["expectedContradictions"])

		rec := tribunalmodels.TribunalGeneratedCase{
			GenerationBatchID:          batch.ID,
			Title:                      title,
			Summary:                    fmt.Sprint(raw["summary"]),
			CaseType:                   defaultText(fmt.Sprint(raw["caseType"]), "custom"),
			Level:                      clampInt(intFromAny(raw["level"], 1), 1, 10),
			Difficulty:                 defaultText(fmt.Sprint(raw["difficulty"]), "standard"),
			EstimatedDurationMinutes:   intFromAny(raw["estimatedDurationMinutes"], 5+clampInt(intFromAny(raw["level"], 5),1,10)*5),
			Mode:                       defaultText(fmt.Sprint(raw["mode"]), "full_case"),
			Tone:                       defaultText(fmt.Sprint(raw["tone"]), "cyberpunk_serious"),
			PlayerRoleSuggestion:       defaultText(fmt.Sprint(raw["playerRoleSuggestion"]), "neutral"),
			AccusationPosition:         fmt.Sprint(raw["accusationPosition"]),
			DefensePosition:            fmt.Sprint(raw["defensePosition"]),
			TagsJSON:                   datatypes.JSON(tagsB),
			WitnessesJSON:              datatypes.JSON(witB),
			EvidenceJSON:               datatypes.JSON(evB),
			TestimonyJSON:              datatypes.JSON(testB),
			ExpectedContradictionsJSON: datatypes.JSON(contrB),
			Status:                     "ready",
			IsPlayable:                 true,
			IsPublished:                true,
			GeneratedByCron:            false,
			ProviderType:               providerType,
			ProviderModel:              model,
		}
		if cerr := db.Create(&rec).Error; cerr == nil {
			generated++
		}
	}

	now := time.Now()
	batch.FinishedAt = &now
	batch.GeneratedCount = generated
	batch.PublishedCount = generated
	batch.DurationMs = time.Since(batch.StartedAt).Milliseconds()
	if generated == 0 {
		batch.Status = "failed"
		batch.ErrorMessage = "aucune affaire valide générée par l'IA"
	} else {
		batch.Status = "success"
	}
	db.Save(&batch)

	return batch.ID, generated, nil
}

// cleanJSONLocal is a local copy of the scheduler's cleanJSON for parsing LLM JSON output.
func cleanJSONLocal(value string) string {
	clean := strings.TrimSpace(value)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)
	start := strings.IndexAny(clean, "[{")
	endArray := strings.LastIndex(clean, "]")
	endObject := strings.LastIndex(clean, "}")
	end := endArray
	if endObject > end {
		end = endObject
	}
	if start >= 0 && end >= start {
		return clean[start : end+1]
	}
	return clean
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func intFromAny(v any, fb int) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int32:
		return int(x)
	case int64:
		return int(x)
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(x)); err == nil {
			return n
		}
	}
	return fb
}

// small helpers for load (intFromAny already defined above for the trigger)
func boolFromAny(v any, fb bool) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return fb
}
func stringSliceFromAny(v any) []string {
	if arr, ok := v.([]any); ok {
		out := make([]string, 0, len(arr))
		for _, a := range arr {
			out = append(out, fmt.Sprint(a))
		}
		return out
	}
	return nil
}
