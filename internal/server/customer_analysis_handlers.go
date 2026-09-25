package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
)

type customerAnalysisPayload struct {
	FullName         string  `json:"full_name"`
	OccupationRole   string  `json:"occupation_role"`
	OccupationOther  *string `json:"occupation_other"`
	Age              int     `json:"age"`
	Gender           string  `json:"gender"`
	InterviewPurpose string  `json:"interview_purpose"`
	Summary          string  `json:"summary"`
	Sentiment        int     `json:"sentiment"`
}

// ListCustomerAnalysesHandler godoc
//
//	@Summary	Daftar Customer Analysis milik Manager KDKMP
//	@Tags		Customer Analysis
//	@Produce	json
//	@Security	CookieAuth
//	@Success	200	{object}	map[string]any
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Router		/api/v1/customer-analyses [get]
func ListCustomerAnalysesHandler(c *gin.Context) {
	user, ok := customerAnalysisUser(c)
	if !ok {
		return
	}

	var analyses []models.CustomerAnalysis
	if err := AppDeps.DB.WithContext(c.Request.Context()).
		Where("user_id = ?", user.ID).
		Order("created_at DESC").Order("id DESC").
		Find(&analyses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat Customer Analysis"})
		return
	}

	data := make([]gin.H, 0, len(analyses))
	for index := range analyses {
		data = append(data, customerAnalysisResponse(&analyses[index]))
	}
	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"options": gin.H{
			"occupations": customerAnalysisOptions(models.CustomerOccupationLabels()),
			"sentiments":  customerAnalysisSentimentOptions(),
		},
	})
}

// CreateCustomerAnalysisHandler godoc
//
//	@Summary	Buat Customer Analysis milik Manager KDKMP
//	@Tags		Customer Analysis
//	@Accept		json
//	@Produce	json
//	@Security	CookieAuth
//	@Param		payload	body		customerAnalysisPayload	true	"Data narasumber"
//	@Success	201		{object}	map[string]any
//	@Failure	401		{object}	map[string]string
//	@Failure	403		{object}	map[string]string
//	@Failure	422		{object}	map[string]string
//	@Router		/api/v1/customer-analyses [post]
func CreateCustomerAnalysisHandler(c *gin.Context) {
	user, ok := customerAnalysisUser(c)
	if !ok {
		return
	}

	var payload customerAnalysisPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data Customer Analysis tidak valid"})
		return
	}
	analysis, err := parseCustomerAnalysisPayload(payload)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": err.Error()})
		return
	}
	analysis.UserID = user.ID

	if err := AppDeps.DB.WithContext(c.Request.Context()).Create(analysis).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan Customer Analysis"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": customerAnalysisResponse(analysis)})
}

// UpdateCustomerAnalysisHandler godoc
//
//	@Summary	Perbarui Customer Analysis milik Manager KDKMP
//	@Tags		Customer Analysis
//	@Accept		json
//	@Produce	json
//	@Security	CookieAuth
//	@Param		id		path		int				true	"ID Customer Analysis"
//	@Param		payload	body		customerAnalysisPayload	true	"Data narasumber"
//	@Success	200		{object}	map[string]any
//	@Failure	401		{object}	map[string]string
//	@Failure	403		{object}	map[string]string
//	@Failure	404		{object}	map[string]string
//	@Failure	422		{object}	map[string]string
//	@Router		/api/v1/customer-analyses/{id} [put]
func UpdateCustomerAnalysisHandler(c *gin.Context) {
	user, ok := customerAnalysisUser(c)
	if !ok {
		return
	}
	analysis, ok := ownedCustomerAnalysis(c, user.ID)
	if !ok {
		return
	}

	var payload customerAnalysisPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data Customer Analysis tidak valid"})
		return
	}
	updated, err := parseCustomerAnalysisPayload(payload)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": err.Error()})
		return
	}
	analysis.FullName = updated.FullName
	analysis.OccupationRole = updated.OccupationRole
	analysis.OccupationOther = updated.OccupationOther
	analysis.Age = updated.Age
	analysis.Gender = updated.Gender
	analysis.InterviewPurpose = updated.InterviewPurpose
	analysis.Summary = updated.Summary
	analysis.Sentiment = updated.Sentiment

	if err := AppDeps.DB.WithContext(c.Request.Context()).Save(analysis).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memperbarui Customer Analysis"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": customerAnalysisResponse(analysis)})
}

func customerAnalysisUser(c *gin.Context) (*models.User, bool) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return nil, false
	}
	return user, true
}

func ownedCustomerAnalysis(c *gin.Context, userID int64) (*models.CustomerAnalysis, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer Analysis tidak ditemukan"})
		return nil, false
	}
	var analysis models.CustomerAnalysis
	err = AppDeps.DB.WithContext(c.Request.Context()).Where("id = ? AND user_id = ?", id, userID).First(&analysis).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer Analysis tidak ditemukan"})
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat Customer Analysis"})
		return nil, false
	}
	return &analysis, true
}

func parseCustomerAnalysisPayload(payload customerAnalysisPayload) (*models.CustomerAnalysis, error) {
	fullName, err := requiredCustomerAnalysisText(payload.FullName, "Nama narasumber wajib diisi", 255)
	if err != nil {
		return nil, err
	}
	occupationRole := strings.TrimSpace(payload.OccupationRole)
	if _, ok := models.CustomerOccupationLabels()[occupationRole]; !ok {
		return nil, errors.New("Pekerjaan atau peran tidak valid")
	}
	var occupationOther *string
	if payload.OccupationOther != nil {
		value := strings.TrimSpace(*payload.OccupationOther)
		if value != "" {
			if utf8.RuneCountInString(value) > 255 {
				return nil, errors.New("Pekerjaan atau peran lainnya maksimal 255 karakter")
			}
			occupationOther = &value
		}
	}
	if occupationRole == models.CustomerOccupationOther && occupationOther == nil {
		return nil, errors.New("Pekerjaan atau peran lainnya wajib diisi")
	}
	if payload.Age < 1 || payload.Age > 120 {
		return nil, errors.New("Umur harus berada antara 1 hingga 120 tahun")
	}
	gender := strings.TrimSpace(payload.Gender)
	if _, ok := models.CustomerGenderLabels()[gender]; !ok {
		return nil, errors.New("Jenis kelamin tidak valid")
	}
	interviewPurpose, err := requiredCustomerAnalysisText(payload.InterviewPurpose, "Tujuan wawancara wajib diisi", 1000)
	if err != nil {
		return nil, err
	}
	summary, err := requiredCustomerAnalysisText(payload.Summary, "Ringkasan wajib diisi", 5000)
	if err != nil {
		return nil, err
	}
	if payload.Sentiment < 1 || payload.Sentiment > 5 {
		return nil, errors.New("Nilai sentimen harus berada antara 1 hingga 5")
	}

	return &models.CustomerAnalysis{FullName: fullName, OccupationRole: occupationRole, OccupationOther: occupationOther, Age: payload.Age, Gender: gender, InterviewPurpose: interviewPurpose, Summary: summary, Sentiment: payload.Sentiment}, nil
}

func requiredCustomerAnalysisText(value, requiredMessage string, maximum int) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errors.New(requiredMessage)
	}
	if utf8.RuneCountInString(trimmed) > maximum {
		return "", errors.New("Teks melebihi batas karakter")
	}
	return trimmed, nil
}

func customerAnalysisResponse(analysis *models.CustomerAnalysis) gin.H {
	occupationLabel := models.CustomerOccupationLabels()[analysis.OccupationRole]
	if analysis.OccupationRole == models.CustomerOccupationOther && analysis.OccupationOther != nil {
		occupationLabel = *analysis.OccupationOther
	}
	return gin.H{
		"id": analysis.ID, "full_name": analysis.FullName, "occupation_role": analysis.OccupationRole,
		"occupation_other": analysis.OccupationOther, "occupation_label": occupationLabel, "age": analysis.Age,
		"gender": analysis.Gender, "gender_label": models.CustomerGenderLabels()[analysis.Gender],
		"interview_purpose": analysis.InterviewPurpose, "summary": analysis.Summary, "sentiment": analysis.Sentiment,
		"sentiment_label": models.CustomerSentimentLabels()[analysis.Sentiment], "created_at": analysis.CreatedAt, "updated_at": analysis.UpdatedAt,
	}
}

func customerAnalysisOptions(labels map[string]string) []gin.H {
	keys := []string{models.CustomerOccupationFarmer, models.CustomerOccupationFisher, models.CustomerOccupationRetailCustomer, models.CustomerOccupationUMKMOwner, models.CustomerOccupationOther}
	result := make([]gin.H, 0, len(keys))
	for _, key := range keys {
		result = append(result, gin.H{"value": key, "label": labels[key]})
	}
	return result
}

func customerAnalysisSentimentOptions() []gin.H {
	labels := models.CustomerSentimentLabels()
	result := make([]gin.H, 0, len(labels))
	for value := 1; value <= 5; value++ {
		result = append(result, gin.H{"value": value, "label": labels[value]})
	}
	return result
}
