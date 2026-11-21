// Package model berisi struktur data yang dipakai bersama oleh semua lapisan.
package model

import (
	"encoding/json"
	"strings"
)

// OnOff menerima true/false, 1/0, atau "ON"/"OFF" dari perangkat,
// supaya format payload firmware tidak harus persis sama.
type OnOff bool

func (o *OnOff) UnmarshalJSON(b []byte) error {
	s := strings.ToUpper(strings.Trim(strings.TrimSpace(string(b)), `"`))
	switch s {
	case "TRUE", "1", "ON", "YES", "HIGH":
		*o = true
	default:
		*o = false
	}
	return nil
}

// Telemetry adalah payload MQTT periodik dari perangkat (JSON).
// Suhu, kelembaban, dan gas wajib ada (pointer agar bisa dibedakan dari nol).
type Telemetry struct {
	DeviceID       string   `json:"device_id"`
	Temperature    *float64 `json:"temperature"`
	Humidity       *float64 `json:"humidity"`
	GasLevel       *float64 `json:"gas_level"`
	RiskLevel      string   `json:"scm_risk_level"`
	ExhaustFan     OnOff    `json:"exhaust_fan"`
	VentOpen       OnOff    `json:"vent_open"` // flap ventilasi terbuka
	BuzzerAlarm    OnOff    `json:"buzzer_alarm"`
	ControlMode    string   `json:"control_mode"`
	ConveyorStatus string   `json:"conveyor_status"` // RUNNING | STOPPED | SCANNING | ERROR
	DistanceCM     float64  `json:"distance_cm"`     // pembacaan HC-SR04
}

// ScanPayload adalah payload MQTT saat barcode batch berhasil dibaca.
// Mendukung dua bentuk: format lama (batch_id) dan format barcode lengkap (id + data box).
type ScanPayload struct {
	DeviceID string `json:"device_id"`
	BatchID  string `json:"batch_id"`
	Material string `json:"material"`
	OK       *bool  `json:"ok"` // jika false, scan diabaikan

	// Format barcode batch dari perangkat
	ID   string  `json:"id"`
	Lot  string  `json:"lot"`
	Frm  string  `json:"frm"`
	Geo  string  `json:"geo"`
	Var  string  `json:"var"`
	Prc  string  `json:"prc"`
	Dt   string  `json:"dt"`
	MC   float64 `json:"mc"`
	Grd  string  `json:"grd"`
	WT   float64 `json:"wt"`
	Silo string  `json:"silo"`

	Raw string `json:"-"` // JSON asli dari perangkat, disimpan apa adanya
}

// Reading adalah satu pembacaan yang tersimpan di database.
type Reading struct {
	ID         int64   `json:"id"`
	TS         int64   `json:"ts"` // unix detik (UTC)
	DeviceID   string  `json:"device_id"`
	Suhu       float64 `json:"suhu"`
	Kelembaban float64 `json:"kelembaban"`
	Gas        float64 `json:"gas"`
	Risiko     string  `json:"risiko"`
	Kipas      bool    `json:"kipas"`
	Ventilasi  bool    `json:"ventilasi"`
	Buzzer     bool    `json:"buzzer"`
	Mode       string  `json:"mode"`
	Konveyor   string  `json:"konveyor"`
	JarakCM    float64 `json:"jarak_cm"`
}

// Scan adalah satu barang (batch) yang masuk ke silo.
type Scan struct {
	ID       int64  `json:"id"`
	TS       int64  `json:"ts"`
	DeviceID string `json:"device_id"`
	BatchID  string `json:"batch_id"`
	Material string `json:"material"`
	Payload  string `json:"-"` // JSON barcode lengkap (kosong untuk scan format lama)
}

// TrackingItem adalah batch tracking untuk API: data barcode dikirim sebagai objek JSON.
type TrackingItem struct {
	Scan
	Data json.RawMessage `json:"data"`
	Foto []string        `json:"foto"` // nama file foto dari kamera perangkat, terbaru dulu
}

// Event adalah catatan kejadian penting (perubahan level, kipas, konveyor, barang masuk).
type Event struct {
	ID       int64  `json:"id"`
	TS       int64  `json:"ts"`
	DeviceID string `json:"device_id"`
	Jenis    string `json:"jenis"` // risiko | kipas | konveyor | barang
	Level    string `json:"level"` // NORMAL | WARNING | CRITICAL | INFO
	Pesan    string `json:"pesan"`
}

// User adalah akun yang boleh masuk ke dashboard.
type User struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	Nama         string `json:"nama"`
	PasswordHash string `json:"-"`
}
