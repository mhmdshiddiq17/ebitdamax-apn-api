package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// FixedCostConfiguredKey menandai fixed cost task sudah dikonfigurasi eksplisit.
const FixedCostConfiguredKey = "_configured"

// CostBreakdown adalah map komponen biaya yang disimpan sebagai JSONB.
type CostBreakdown map[string]int

// GormDataType memberi tahu GORM tipe kolom JSONB.
func (CostBreakdown) GormDataType() string {
	return "jsonb"
}

// Value mengubah map menjadi JSON untuk kolom jsonb. Nilai `_configured`
// ditulis sebagai boolean agar kompatibel dengan data aplikasi lama.
func (c CostBreakdown) Value() (driver.Value, error) {
	if c == nil {
		return "{}", nil
	}

	payload := make(map[string]any, len(c))
	for key, value := range c {
		if key == FixedCostConfiguredKey {
			payload[key] = value != 0
			continue
		}
		payload[key] = value
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return string(encoded), nil
}

// Scan membaca kolom jsonb menjadi map; nilai boolean/angka/string numerik
// dinormalisasi menjadi integer (kompatibel dengan data aplikasi lama).
func (c *CostBreakdown) Scan(value any) error {
	if value == nil {
		*c = CostBreakdown{}
		return nil
	}

	var raw []byte
	switch typed := value.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return fmt.Errorf("cost breakdown: tipe tidak didukung %T", value)
	}

	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return err
	}

	breakdown := make(CostBreakdown, len(parsed))
	for key, item := range parsed {
		breakdown[key] = coerceCostValue(item)
	}
	*c = breakdown

	return nil
}

func coerceCostValue(value any) int {
	switch typed := value.(type) {
	case nil:
		return 0
	case bool:
		if typed {
			return 1
		}
		return 0
	case float64:
		if typed < 0 {
			return 0
		}
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil || parsed < 0 {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

// NormalizeCostBreakdown menyeragamkan komponen biaya menjadi integer >= 0.
func NormalizeCostBreakdown(raw CostBreakdown) CostBreakdown {
	return CostBreakdown{
		"man":      normalizeCostValue(raw["man"]),
		"machine":  normalizeCostValue(raw["machine"]),
		"method":   normalizeCostValue(raw["method"]),
		"material": normalizeCostValue(raw["material"]),
	}
}

// ConfiguredFixedCostBreakdown menandai fixed cost sebagai terkonfigurasi.
func ConfiguredFixedCostBreakdown(raw CostBreakdown) CostBreakdown {
	normalized := NormalizeCostBreakdown(raw)
	normalized[FixedCostConfiguredKey] = 1
	return normalized
}

// CostTotal menjumlahkan komponen biaya.
func CostTotal(raw CostBreakdown) int {
	total := 0
	for _, value := range NormalizeCostBreakdown(raw) {
		total += value
	}
	return total
}

func normalizeCostValue(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

// StringList adalah daftar string yang disimpan sebagai JSONB.
type StringList []string

// GormDataType memberi tahu GORM tipe kolom JSONB.
func (StringList) GormDataType() string {
	return "jsonb"
}

// Value mengubah daftar menjadi JSON untuk kolom jsonb.
func (s StringList) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	payload, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(payload), nil
}

// Scan membaca kolom jsonb menjadi daftar string.
func (s *StringList) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}

	var raw []byte
	switch typed := value.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return fmt.Errorf("string list: tipe tidak didukung %T", value)
	}

	return json.Unmarshal(raw, s)
}

// ClockTime mewakili kolom `time` (jam:menit) tanpa tanggal.
type ClockTime struct {
	Hour   int
	Minute int
	Valid  bool
}

// Value mengubah jam menjadi string waktu untuk kolom time.
func (c ClockTime) Value() (driver.Value, error) {
	if !c.Valid {
		return nil, nil
	}
	return fmt.Sprintf("%02d:%02d:00", c.Hour, c.Minute), nil
}

// Scan membaca kolom time (mendukung pgtype.Time maupun time.Time).
func (c *ClockTime) Scan(value any) error {
	if value == nil {
		c.Valid = false
		return nil
	}

	switch typed := value.(type) {
	case time.Time:
		c.Hour, c.Minute, c.Valid = typed.Hour(), typed.Minute(), true
		return nil
	case pgtype.Time:
		if !typed.Valid {
			c.Valid = false
			return nil
		}
		total := typed.Microseconds
		c.Hour = int(total / 3_600_000_000)
		c.Minute = int((total % 3_600_000_000) / 60_000_000)
		c.Valid = true
		return nil
	}

	var text string
	switch typed := value.(type) {
	case []byte:
		text = string(typed)
	case string:
		text = typed
	default:
		text = fmt.Sprint(value)
	}

	parsed, err := time.Parse("15:04:05", text)
	if err != nil {
		parsed, err = time.Parse("15:04", text)
	}
	if err != nil {
		return fmt.Errorf("clock time: format tidak dikenal %q", text)
	}

	c.Hour, c.Minute, c.Valid = parsed.Hour(), parsed.Minute(), true
	return nil
}

// String mengembalikan jam dalam format "15:04" (kosong jika nil/tidak valid).
func (c *ClockTime) String() string {
	if c == nil || !c.Valid {
		return ""
	}
	return fmt.Sprintf("%02d:%02d", c.Hour, c.Minute)
}

// Task merepresentasikan tabel `tasks` (skema existing, tanpa perubahan).
type Task struct {
	ID             int64         `gorm:"primaryKey" json:"id"`
	UUID           string        `gorm:"column:uuid" json:"uuid"`
	TaskCategoryID int64         `gorm:"column:task_category_id" json:"task_category_id"`
	BMCStatus      string        `gorm:"column:bmc_status" json:"bmc_status"`
	SortOrder      *int          `gorm:"column:sort_order" json:"sort_order"`
	Name           string        `json:"name"`
	Description    *string       `json:"description"`
	ExecutionTime  *ClockTime    `gorm:"column:execution_time" json:"-"`
	TimeRequire    int           `gorm:"column:time_require" json:"time_require"`
	LowerThreshold *int          `gorm:"column:lower_time_threshold_minutes" json:"lower_time_threshold_minutes"`
	UpperThreshold *int          `gorm:"column:upper_time_threshold_minutes" json:"upper_time_threshold_minutes"`
	Period         string        `json:"period"`
	IsActive       bool          `gorm:"column:is_active" json:"is_active"`
	IsMandatory    bool          `gorm:"column:is_mandatory" json:"is_mandatory"`
	FixedCost      CostBreakdown `gorm:"column:fixed_cost;type:jsonb" json:"-"`
	VariableCost   CostBreakdown `gorm:"column:variable_cost;type:jsonb" json:"-"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`

	TaskCategory     *TaskCategory         `gorm:"foreignKey:TaskCategoryID" json:"task_category,omitempty"`
	Roles            []Role                `gorm:"many2many:task_roles" json:"roles,omitempty"`
	AdditionalFields []TaskAdditionalField `gorm:"foreignKey:TaskID" json:"additional_fields,omitempty"`
}

// TableName memetakan struct ke tabel existing.
func (Task) TableName() string {
	return "tasks"
}

// FixedCostBreakdown mengembalikan komponen fixed cost yang sudah dinormalisasi.
func (t *Task) FixedCostBreakdown() CostBreakdown {
	return NormalizeCostBreakdown(t.FixedCost)
}

// VariableCostBreakdown mengembalikan komponen variable cost yang sudah dinormalisasi.
func (t *Task) VariableCostBreakdown() CostBreakdown {
	return NormalizeCostBreakdown(t.VariableCost)
}

// TaskAdditionalField merepresentasikan tabel `task_additional_fields`.
type TaskAdditionalField struct {
	ID         int64      `gorm:"primaryKey" json:"id"`
	UUID       string     `gorm:"column:uuid" json:"uuid"`
	TaskID     int64      `gorm:"column:task_id" json:"task_id"`
	Label      string     `json:"label"`
	FieldName  string     `gorm:"column:field_name" json:"field_name"`
	InputType  string     `gorm:"column:input_type" json:"input_type"`
	ShowWhen   string     `gorm:"column:show_when" json:"show_when"`
	IsRequired bool       `gorm:"column:is_required" json:"is_required"`
	SortOrder  int        `gorm:"column:sort_order" json:"sort_order"`
	Options    StringList `gorm:"column:options;type:jsonb" json:"options"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// TableName memetakan struct ke tabel existing.
func (TaskAdditionalField) TableName() string {
	return "task_additional_fields"
}
