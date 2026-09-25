package models

import "time"

const (
	CustomerOccupationFarmer         = "farmer"
	CustomerOccupationFisher         = "fisher"
	CustomerOccupationRetailCustomer = "retail_customer"
	CustomerOccupationUMKMOwner      = "umkm_owner"
	CustomerOccupationOther          = "other"
	CustomerGenderMale               = "male"
	CustomerGenderFemale             = "female"
)

// CustomerAnalysis merepresentasikan catatan wawancara pelanggan milik Manager KDKMP.
type CustomerAnalysis struct {
	ID               int64     `gorm:"primaryKey" json:"id"`
	UserID           int64     `gorm:"column:user_id" json:"user_id"`
	FullName         string    `gorm:"column:full_name" json:"full_name"`
	OccupationRole   string    `gorm:"column:occupation_role" json:"occupation_role"`
	OccupationOther  *string   `gorm:"column:occupation_other" json:"occupation_other"`
	Age              int       `gorm:"column:age" json:"age"`
	Gender           string    `gorm:"column:gender" json:"gender"`
	InterviewPurpose string    `gorm:"column:interview_purpose" json:"interview_purpose"`
	Summary          string    `gorm:"column:summary" json:"summary"`
	Sentiment        int       `gorm:"column:sentiment" json:"sentiment"`
	CreatedAt        time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (CustomerAnalysis) TableName() string { return "customer_analyses" }

func CustomerOccupationLabels() map[string]string {
	return map[string]string{
		CustomerOccupationFarmer:         "Petani",
		CustomerOccupationFisher:         "Nelayan",
		CustomerOccupationRetailCustomer: "Customer Retail",
		CustomerOccupationUMKMOwner:      "Pemilik Produk UMKM",
		CustomerOccupationOther:          "Lainnya",
	}
}

func CustomerGenderLabels() map[string]string {
	return map[string]string{CustomerGenderMale: "Laki-laki", CustomerGenderFemale: "Perempuan"}
}

func CustomerSentimentLabels() map[int]string {
	return map[int]string{1: "Negatif", 2: "Cenderung Negatif", 3: "Netral", 4: "Cenderung Positif", 5: "Positif"}
}
