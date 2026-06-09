package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"cgwm/battle/internal/nexus_game/models"
	"cgwm/battle/internal/nexus_game/services"

	"github.com/gin-gonic/gin"
)

// ContentHandler provides REST CRUD + asset upload for major content items (buildings first).
// Used by admin Next.js for tables + forms, and by Flutter for catalog.
// All images uploaded here are stored in the persistent Nexus assets volume.
// The system is kept open: add similar methods for units/research.

type ContentHandler struct {
	contentSvc *services.ContentService
}

func NewContentHandler(contentSvc *services.ContentService) *ContentHandler {
	return &ContentHandler{contentSvc: contentSvc}
}

// === Buildings (table + CRUD for admin) ===

func (h *ContentHandler) ListBuildings(c *gin.Context) {
	published := c.Query("published") == "true"
	list, err := h.contentSvc.ListBuildings(published)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"buildings": list, "count": len(list)})
}

func (h *ContentHandler) Catalog(c *gin.Context) {
	published := c.Query("published") == "true"
	catalog, err := h.contentSvc.Catalog(published)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, catalog)
}

func (h *ContentHandler) GetBuilding(c *gin.Context) {
	id := c.Param("contentId")
	b, err := h.contentSvc.GetBuilding(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "building not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"building": b})
}

func (h *ContentHandler) CreateOrUpdateBuilding(c *gin.Context) {
	var def models.BuildingDefinition
	if err := c.ShouldBindJSON(&def); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if pathID := c.Param("contentId"); pathID != "" {
		if def.ContentID == "" {
			def.ContentID = pathID
		}
		if def.ContentID != pathID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "contentId path/body mismatch"})
			return
		}
	}
	if def.ContentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contentId required"})
		return
	}
	if err := h.contentSvc.CreateOrUpdateBuilding(&def); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "contentId": def.ContentID})
}

func (h *ContentHandler) DeleteBuilding(c *gin.Context) {
	id := c.Param("contentId")
	if err := h.contentSvc.DeleteBuilding(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *ContentHandler) DeleteBuildingByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.contentSvc.DeleteBuildingByID(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *ContentHandler) TranslationStatus(c *gin.Context) {
	localesParam := strings.TrimSpace(c.DefaultQuery("locales", "fr,en,de"))
	locales := []string{}
	for _, locale := range strings.Split(localesParam, ",") {
		locale = strings.TrimSpace(locale)
		if locale != "" {
			locales = append(locales, locale)
		}
	}
	rows, err := h.contentSvc.TranslationStatus(locales)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	missing := 0
	for _, row := range rows {
		if row.Key == "" || !row.Exists || len(row.MissingLocales) > 0 {
			missing++
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"rows":         rows,
		"count":        len(rows),
		"missingCount": missing,
		"locales":      locales,
	})
}

func (h *ContentHandler) AssetStatus(c *gin.Context) {
	publicContentBaseURL := requestBaseURL(c, path.Join(assetsBaseURL(), "content"))
	rows, err := h.contentSvc.AssetStatus(publicContentBaseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	missing := 0
	for _, row := range rows {
		if !row.Exists {
			missing++
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"rows":         rows,
		"count":        len(rows),
		"missingCount": missing,
	})
}

func requestBaseURL(c *gin.Context, publicPath string) string {
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		scheme = c.GetHeader("X-Scheme")
	}
	if scheme == "" {
		scheme = "https"
		if c.Request.TLS == nil {
			scheme = "http"
		}
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return fmt.Sprintf("%s://%s%s", strings.Split(scheme, ",")[0], strings.Split(host, ",")[0], publicPath)
}

func contentAssetFolder(domain string) string {
	switch domain {
	case "research":
		return "research"
	case "building":
		return "buildings"
	case "unit":
		return "units"
	default:
		return domain + "s"
	}
}

// UploadAsset for a building (or other domain).
// Form: file (multipart), contentId, domain="building", tier="1"|"2"|"3"|"4" (optional)
func (h *ContentHandler) UploadAsset(c *gin.Context) {
	domain := c.PostForm("domain")
	contentID := c.PostForm("contentId")
	tier := c.PostForm("tier")
	if domain == "" || contentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain and contentId required"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read file failed"})
		return
	}

	folder := contentAssetFolder(domain)
	publicContentBaseURL := requestBaseURL(c, path.Join(assetsBaseURL(), "content"))
	savedName, publicURL, err := h.contentSvc.UploadAsset(domain, contentID, tier, header.Filename, data, publicContentBaseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":         true,
		"savedAs":    savedName,
		"url":        publicURL,
		"urlHint":    publicURL,
		"publicPath": "/nexus-assets/content/" + folder + "/" + savedName,
	})
}

// === Player constructions (depends on buildings) ===

func (h *ContentHandler) ListPlayerBuildings(c *gin.Context) {
	pidStr := c.Param("id")
	pid, _ := strconv.ParseUint(pidStr, 10, 64)
	list, err := h.contentSvc.ListPlayerBuildings(uint(pid))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"buildings": list})
}

func (h *ContentHandler) StartConstruction(c *gin.Context) {
	// body: {profileId, contentId, targetLevel}
	var body struct {
		ProfileID   uint   `json:"profileId"`
		ContentID   string `json:"contentId"`
		TargetLevel int    `json:"targetLevel"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pb, err := h.contentSvc.StartConstruction(body.ProfileID, body.ContentID, body.TargetLevel)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"playerBuilding": pb, "note": "Construction started. Complete on tick or poll /complete."})
}

// Poll or call on load to complete ready constructions (server truth).
func (h *ContentHandler) CompleteReadyConstructions(c *gin.Context) {
	pidStr := c.Param("id")
	pid, _ := strconv.ParseUint(pidStr, 10, 64)

	list, _ := h.contentSvc.ListPlayerBuildings(uint(pid))
	completed := []uint{}
	for i := range list {
		pb := &list[i]
		done, _ := h.contentSvc.CompleteConstructionIfReady(pb)
		if done {
			completed = append(completed, pb.ID)
		}
	}
	c.JSON(http.StatusOK, gin.H{"completed": completed, "remaining": len(list) - len(completed)})
}

// === Simple admin HTML "pages" (table + basic CRUD UI) for backend dev ===
// These are quick browser-accessible pages with tables. Full power is the REST JSON + upload.
// Images uploaded appear in /nexus-assets after static mount.

func (h *ContentHandler) AdminBuildingsPage(c *gin.Context) {
	list, _ := h.contentSvc.ListBuildings(true)

	html := `<html><head><title>Nexus Admin - Buildings</title>
<style>body{font-family:sans-serif;margin:20px} table{border-collapse:collapse;width:100%} th,td{border:1px solid #ccc;padding:8px} form{margin:10px 0}</style>
</head><body>
<h1>Buildings CRUD (Backend Admin Page)</h1>
<p>Upload images via the form or the /admin/content/upload-asset endpoint. Served from the persistent nexus-assets volume.</p>

<h2>Create / Update</h2>
<form method="POST" action="/api/nexus-game/admin/content/buildings">
  contentId: <input name="contentId"><br>
  nameKey: <input name="nameKey"><br>
  rarity: <input name="rarity" value="common"><br>
  <button type="submit">Save (JSON body better for full fields - use curl/Next for now)</button>
</form>

<h2>Asset Upload</h2>
<form method="POST" action="/api/nexus-game/admin/content/upload-asset" enctype="multipart/form-data">
  domain: <input name="domain" value="building"><br>
  contentId: <input name="contentId"><br>
  tier: <input name="tier" value="1"><br>
  file: <input type="file" name="file"><br>
  <button type="submit">Upload Asset</button>
</form>

<h2>Existing Buildings</h2>
<table><tr><th>contentId</th><th>nameKey</th><th>rarity</th><th>maxLevel</th><th>Actions</th></tr>`
	for _, b := range list {
		html += `<tr><td>` + b.ContentID + `</td><td>` + b.NameKey + `</td><td>` + b.Rarity + `</td><td>` + strconv.Itoa(b.MaxLevel) + `</td><td>
		<a href="/api/nexus-game/admin/content/buildings/` + b.ContentID + `">View</a> | 
		<form style="display:inline" method="POST" action="/api/nexus-game/admin/content/buildings/` + b.ContentID + `/delete"><button>Delete</button></form>
		</td></tr>`
	}
	html += `</table>
<p>Note: Full CRUD via JSON API. This is a simple dev page. Use Next.js admin for production tables.</p>
</body></html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func (h *ContentHandler) AdminUnitsPage(c *gin.Context) {
	list, _ := h.contentSvc.ListUnits(true)
	html := `<html><head><title>Nexus Admin - Units</title>
<style>body{font-family:sans-serif;margin:20px} table{border-collapse:collapse;width:100%} th,td{border:1px solid #ccc;padding:8px} form{margin:10px 0}</style>
</head><body>
<h1>Units CRUD (Backend Admin Page)</h1>
<p>See NEXUS REFERENCE §5 for the 15 units + per-level stats 1-30 + counters table. Upload assets via /admin/content/upload-asset (domain=unit).</p>

<h2>JSON API</h2>
<p>Full CRUD at /admin/content/units (GET list, POST create, etc.). Use curl or Next.js for now.</p>

<h2>Existing Units</h2>
<table><tr><th>contentId</th><th>nameKey</th><th>rarity</th><th>Actions</th></tr>`
	for _, u := range list {
		html += `<tr><td>` + u.ContentID + `</td><td>` + u.NameKey + `</td><td>` + u.Rarity + `</td><td>
		<a href="/api/nexus-game/admin/content/units/` + u.ContentID + `">View</a> | 
		<form style="display:inline" method="POST" action="/api/nexus-game/admin/content/units/` + u.ContentID + `/delete"><button>Delete</button></form>
		</td></tr>`
	}
	html += `</table>
<p>Note: Seed more from reference. This is dev page.</p>
</body></html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func (h *ContentHandler) AdminResearchPage(c *gin.Context) {
	list, _ := h.contentSvc.ListResearch(true)
	html := `<html><head><title>Nexus Admin - Research</title>
<style>body{font-family:sans-serif;margin:20px} table{border-collapse:collapse;width:100%} th,td{border:1px solid #ccc;padding:8px}</style>
</head><body>
<h1>Research CRUD (Backend Admin Page)</h1>
<p>11 branches, 7 tiers each per NEXUS REFERENCE §6. Dependencies, effects, per-level costs.</p>

<h2>JSON API</h2>
<p>Full at /admin/content/research</p>

<h2>Existing Research Nodes</h2>
<table><tr><th>contentId</th><th>nameKey</th><th>branch</th><th>tier</th><th>Actions</th></tr>`
	for _, r := range list {
		html += `<tr><td>` + r.ContentID + `</td><td>` + r.NameKey + `</td><td>` + r.Branch + `</td><td>` + strconv.Itoa(r.Tier) + `</td><td>
		<a href="/api/nexus-game/admin/content/research/` + r.ContentID + `">View</a> | 
		<form style="display:inline" method="POST" action="/api/nexus-game/admin/content/research/` + r.ContentID + `/delete"><button>Delete</button></form>
		</td></tr>`
	}
	html += `</table>
</body></html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func (h *ContentHandler) ListUnits(c *gin.Context) {
	list, err := h.contentSvc.ListUnits(true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"units": list, "count": len(list)})
}

func (h *ContentHandler) ListResearch(c *gin.Context) {
	list, err := h.contentSvc.ListResearch(true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"research": list, "count": len(list)})
}

func (h *ContentHandler) GetUnit(c *gin.Context) {
	id := c.Param("contentId")
	unit, err := h.contentSvc.GetUnit(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"unit": unit})
}

func (h *ContentHandler) GetResearch(c *gin.Context) {
	id := c.Param("contentId")
	research, err := h.contentSvc.GetResearch(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "research not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"research": research})
}

func (h *ContentHandler) CreateOrUpdateUnit(c *gin.Context) {
	var def models.UnitDefinition
	if err := c.ShouldBindJSON(&def); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if pathID := c.Param("contentId"); pathID != "" {
		if def.ContentID == "" {
			def.ContentID = pathID
		}
		if def.ContentID != pathID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "contentId path/body mismatch"})
			return
		}
	}
	if def.ContentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contentId required"})
		return
	}
	if err := h.contentSvc.CreateOrUpdateUnit(&def); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "contentId": def.ContentID})
}

func (h *ContentHandler) CreateOrUpdateResearch(c *gin.Context) {
	var def models.ResearchDefinition
	if err := c.ShouldBindJSON(&def); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if pathID := c.Param("contentId"); pathID != "" {
		if def.ContentID == "" {
			def.ContentID = pathID
		}
		if def.ContentID != pathID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "contentId path/body mismatch"})
			return
		}
	}
	if def.ContentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contentId required"})
		return
	}
	if err := h.contentSvc.CreateOrUpdateResearch(&def); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "contentId": def.ContentID})
}

func (h *ContentHandler) DeleteUnit(c *gin.Context) {
	id := c.Param("contentId")
	if err := h.contentSvc.DeleteUnit(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *ContentHandler) DeleteUnitByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.contentSvc.DeleteUnitByID(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *ContentHandler) DeleteResearchByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.contentSvc.DeleteResearchByID(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *ContentHandler) DeleteResearch(c *gin.Context) {
	id := c.Param("contentId")
	if err := h.contentSvc.DeleteResearch(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Note: Admin pages now render real tables from DB for all three (buildings full, units/research basic).
// Seed the full catalogs from the reference to populate. Full CRUD (POST/PUT/DELETE) available for units and research too.
