package models

// Nilai & label enum task (mirror app/Enums pada aplikasi lama).

// TaskBmcStatusOptions berisi opsi poin BMC.
var TaskBmcStatusOptions = []map[string]string{
	{"value": "belum_dipetakan", "label": "Belum Dipetakan"},
	{"value": "key_partnerships", "label": "Key Partnerships"},
	{"value": "key_activities", "label": "Key Activities"},
	{"value": "key_resources", "label": "Key Resources"},
	{"value": "value_propositions", "label": "Value Propositions"},
	{"value": "customer_relationships", "label": "Customer Relationships"},
	{"value": "channels", "label": "Channels"},
	{"value": "customer_segments", "label": "Customer Segments"},
	{"value": "cost_structure", "label": "Cost Structure"},
	{"value": "revenue_streams", "label": "Revenue Streams"},
}

// TaskPeriodOptions berisi opsi periode task.
var TaskPeriodOptions = []map[string]string{
	{"value": "once", "label": "Sekali"},
	{"value": "daily", "label": "Harian"},
	{"value": "weekly", "label": "Mingguan"},
	{"value": "monthly", "label": "Bulanan"},
}

// TaskInputTypeOptions berisi opsi tipe input field tambahan.
var TaskInputTypeOptions = []map[string]string{
	{"value": "text", "label": "Text"},
	{"value": "textarea", "label": "Textarea"},
	{"value": "integer", "label": "Integer"},
	{"value": "decimal", "label": "Decimal"},
	{"value": "number", "label": "Number"},
	{"value": "date", "label": "Date"},
	{"value": "datetime", "label": "Datetime"},
	{"value": "time", "label": "Time"},
	{"value": "boolean", "label": "Boolean"},
	{"value": "select", "label": "Select"},
	{"value": "radio", "label": "Radio"},
	{"value": "checkbox", "label": "Checkbox"},
	{"value": "file", "label": "File Upload"},
}

// TaskShowWhenOptions berisi opsi kapan field tambahan ditampilkan.
var TaskShowWhenOptions = []map[string]string{
	{"value": "start", "label": "Mulai Task"},
	{"value": "finish", "label": "Selesaikan Task"},
}

// OptionValueValid memastikan nilai ada di daftar opsi.
func OptionValueValid(options []map[string]string, value string) bool {
	for _, option := range options {
		if option["value"] == value {
			return true
		}
	}
	return false
}

// OptionLabel mengembalikan label dari nilai opsi.
func OptionLabel(options []map[string]string, value string) string {
	for _, option := range options {
		if option["value"] == value {
			return option["label"]
		}
	}
	return value
}
