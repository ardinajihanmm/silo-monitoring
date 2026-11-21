// Package handler berisi endpoint REST API (Fiber v3), autentikasi JWT, dan CORS.
//
// Format respons: sukses {"data": ...}, gagal {"pesan": "..."}.
package handler

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"silo/internal/config"
	"silo/internal/model"
	"silo/internal/risk"
	"silo/internal/stats"
	"silo/internal/store"
)

// Controller mengirim perintah ke perangkat (MQTT) atau ke simulator.
type Controller interface {
	Publish(payload []byte) error
}

type Handler struct {
	Store *store.Store
	Cfg   config.Config
	Ctl   Controller
	Loc   *time.Location
}

// ErrorHandler membuat semua galat berbentuk JSON {"pesan": ...}.
func ErrorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "Terjadi kesalahan pada server"
	var fe *fiber.Error
	if errors.As(err, &fe) {
		code, msg = fe.Code, fe.Message
	}
	return c.Status(code).JSON(fiber.Map{"pesan": msg})
}

func fail(c fiber.Ctx, code int, msg string) error {
	return c.Status(code).JSON(fiber.Map{"pesan": msg})
}

func ok(c fiber.Ctx, data any) error {
	return c.JSON(fiber.Map{"data": data})
}

// Register memasang semua rute. Middleware CORS dipasang di main.go SEBELUM Register,
// sehingga permintaan preflight OPTIONS dijawab sebelum sampai ke pengecekan token.
func (h *Handler) Register(app *fiber.App) {
	api := app.Group("/api")
	api.Get("/health", h.Health)
	api.Post("/auth/login", h.Login)

	api.Get("/auth/profil", h.Auth, h.Profil)
	api.Post("/auth/logout", h.Auth, h.Logout)
	api.Get("/telemetri/terbaru", h.Auth, h.Terbaru)
	api.Get("/telemetri/riwayat", h.Auth, h.Riwayat)
	api.Get("/statistik", h.Auth, h.Statistik)
	api.Get("/barang", h.Auth, h.Barang)
	api.Get("/tracking", h.Auth, h.Tracking)
	api.Get("/tracking/:kode", h.Auth, h.TrackingDetail)
	api.Post("/device/foto", h.DeviceFoto)
	api.Get("/tracking/:kode/foto/:file", h.Auth, h.FotoFile)
	api.Get("/kejadian", h.Auth, h.Kejadian)
	api.Post("/kontrol", h.Auth, h.Kontrol)
}

// ---------- auth ----------

func (h *Handler) Auth(c fiber.Ctx) error {
	hdr := c.Get("Authorization")
	if !strings.HasPrefix(hdr, "Bearer ") {
		return fail(c, fiber.StatusUnauthorized, "Token tidak ditemukan")
	}
	raw := strings.TrimPrefix(hdr, "Bearer ")
	claims := &jwt.RegisteredClaims{}
	tok, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) {
		return []byte(h.Cfg.JWTSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !tok.Valid {
		return fail(c, fiber.StatusUnauthorized, "Token tidak valid atau kedaluwarsa")
	}
	c.Locals("email", claims.Subject)
	return c.Next()
}

func (h *Handler) Login(c fiber.Ctx) error {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(c.Body(), &in); err != nil || in.Email == "" || in.Password == "" {
		return fail(c, fiber.StatusUnprocessableEntity, "Email dan password wajib diisi")
	}
	u, err := h.Store.UserByEmail(strings.ToLower(strings.TrimSpace(in.Email)))
	if err != nil {
		return err
	}
	if u == nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		return fail(c, fiber.StatusUnauthorized, "Email atau password salah")
	}
	exp := time.Now().Add(time.Duration(h.Cfg.JWTHours) * time.Hour)
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   u.Email,
		ExpiresAt: jwt.NewNumericDate(exp),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}).SignedString([]byte(h.Cfg.JWTSecret))
	if err != nil {
		return err
	}
	return ok(c, fiber.Map{"token": token, "kedaluwarsa": exp.Unix(), "user": fiber.Map{"email": u.Email, "nama": u.Nama}})
}

func (h *Handler) Profil(c fiber.Ctx) error {
	email, _ := c.Locals("email").(string)
	u, err := h.Store.UserByEmail(email)
	if err != nil {
		return err
	}
	if u == nil {
		return fail(c, fiber.StatusUnauthorized, "Pengguna tidak ditemukan")
	}
	return ok(c, fiber.Map{"email": u.Email, "nama": u.Nama})
}

// Logout bersifat stateless: token dibuang di sisi klien.
func (h *Handler) Logout(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"pesan": "Berhasil keluar"})
}

func (h *Handler) Health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok", "waktu": time.Now().Unix()})
}

// ---------- helper ----------

func intQuery(c fiber.Ctx, key string, def, lo, hi int) int {
	n, err := strconv.Atoi(c.Query(key, strconv.Itoa(def)))
	if err != nil {
		return def
	}
	return max(lo, min(hi, n))
}

func toPoints(rs []model.Reading) []stats.Point {
	pts := make([]stats.Point, len(rs))
	for i, r := range rs {
		pts[i] = stats.Point{
			TS: r.TS, Temp: r.Suhu, Hum: r.Kelembaban, Gas: r.Gas,
			Risk: levelOf(r.Risiko), Fan: r.Kipas, Buzzer: r.Buzzer, Conveyor: r.Konveyor,
		}
	}
	return pts
}

func levelOf(s string) risk.Level {
	l, _ := risk.Parse(s)
	return l
}

func startOfDay(now time.Time, loc *time.Location) time.Time {
	n := now.In(loc)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
}

// ---------- data ----------

// Terbaru: pembacaan terakhir + status perangkat + ringkasan barang masuk.
func (h *Handler) Terbaru(c fiber.Ctx) error {
	now := time.Now()
	rd, err := h.Store.LatestReading()
	if err != nil {
		return err
	}
	total, err := h.Store.CountScans()
	if err != nil {
		return err
	}
	today, err := h.Store.CountScansSince(startOfDay(now, h.Loc).Unix())
	if err != nil {
		return err
	}
	recent, err := h.Store.RecentScans(1)
	if err != nil {
		return err
	}
	var lastScan *model.Scan
	if len(recent) > 0 {
		lastScan = &recent[0]
	}

	online, age := false, int64(-1)
	if rd != nil {
		age = now.Unix() - rd.TS
		online = age <= int64(h.Cfg.DeviceTimeoutSec)
	}
	return ok(c, fiber.Map{
		"pembacaan":  rd,
		"online":     online,
		"umur_detik": age,
		"batas":      h.Cfg.Thresholds,
		"ventilasi":  h.Cfg.Vent,
		"barang":     fiber.Map{"total": total, "hari_ini": today, "terakhir": lastScan},
	})
}

// Riwayat: data grafik, dirata-ratakan per interval (maksimal ±120 titik).
func (h *Handler) Riwayat(c fiber.Ctx) error {
	hours := intQuery(c, "jam", 6, 1, 72)
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	rs, err := h.Store.ReadingsSince(since.Unix())
	if err != nil {
		return err
	}
	bucket := int64(hours * 3600 / 120)
	if bucket < 5 {
		bucket = 5
	}
	return ok(c, fiber.Map{
		"jam":          hours,
		"bucket_detik": bucket,
		"titik":        stats.Bucketize(toPoints(rs), bucket),
		"batas":        h.Cfg.Thresholds,
	})
}

// Statistik: seluruh perhitungan data pada jendela waktu `jam` terakhir.
func (h *Handler) Statistik(c fiber.Ctx) error {
	hours := intQuery(c, "jam", 24, 1, 72)
	now := time.Now()
	since := now.Add(-time.Duration(hours) * time.Hour)

	rs, err := h.Store.ReadingsSince(since.Unix())
	if err != nil {
		return err
	}
	summary := stats.Compute(toPoints(rs), h.Cfg.Thresholds)

	scans, err := h.Store.ScansSince(since.Unix())
	if err != nil {
		return err
	}
	times := make([]stats.ScanTime, len(scans))
	for i, s := range scans {
		times[i] = stats.ScanTime{TS: s.TS}
	}
	total, err := h.Store.CountScans()
	if err != nil {
		return err
	}
	today, err := h.Store.CountScansSince(startOfDay(now, h.Loc).Unix())
	if err != nil {
		return err
	}

	chartHours := max(6, min(48, hours))
	return ok(c, fiber.Map{
		"jam":       hours,
		"ringkasan": summary,
		"barang": fiber.Map{
			"total":         total,
			"hari_ini":      today,
			"dalam_jendela": len(scans),
			"laju_per_jam":  float64(int(float64(len(scans))/float64(hours)*100+0.5)) / 100,
			"per_jam":       stats.HourlyCounts(times, chartHours, now.Unix()),
		},
		"batas": h.Cfg.Thresholds,
	})
}

func (h *Handler) Barang(c fiber.Ctx) error {
	list, err := h.Store.RecentScans(intQuery(c, "limit", 20, 1, 200))
	if err != nil {
		return err
	}
	return ok(c, list)
}

func (h *Handler) Kejadian(c fiber.Ctx) error {
	list, err := h.Store.RecentEvents(intQuery(c, "limit", 20, 1, 200))
	if err != nil {
		return err
	}
	return ok(c, list)
}

// ---------- kontrol ----------

var allowedCommands = map[string][]string{
	"fan":      {"ON", "OFF"},
	"vent":     {"OPEN", "CLOSE"},
	"mode":     {"AUTO", "MANUAL"},
	"conveyor": {"START", "STOP"},
}

// Kontrol memvalidasi perintah dari dashboard lalu meneruskannya ke perangkat.
// Hanya kunci dan nilai pada allowedCommands yang diizinkan.
func (h *Handler) Kontrol(c fiber.Ctx) error {
	var body map[string]string
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return fail(c, fiber.StatusBadRequest, "Body harus berupa JSON, contoh: {\"fan\":\"ON\"}")
	}
	cmd := map[string]string{}
	for k, v := range body {
		allowed, known := allowedCommands[k]
		if !known {
			return fail(c, fiber.StatusUnprocessableEntity, "Perintah tidak dikenal: "+k)
		}
		v = strings.ToUpper(strings.TrimSpace(v))
		valid := false
		for _, a := range allowed {
			if a == v {
				valid = true
			}
		}
		if !valid {
			return fail(c, fiber.StatusUnprocessableEntity, "Nilai tidak valid untuk "+k+": "+v)
		}
		cmd[k] = v
	}
	if len(cmd) == 0 {
		return fail(c, fiber.StatusUnprocessableEntity, "Tidak ada perintah")
	}
	payload, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	if err := h.Ctl.Publish(payload); err != nil {
		return fail(c, fiber.StatusBadGateway, "Gagal mengirim perintah: "+err.Error())
	}
	return c.JSON(fiber.Map{"pesan": "Perintah terkirim", "data": cmd})
}

// ---------- tracking ----------

// toTracking mengubah scan menjadi item tracking; data barcode dikirim sebagai objek JSON.
func toTracking(list []model.Scan) []model.TrackingItem {
	out := make([]model.TrackingItem, 0, len(list))
	for _, sc := range list {
		var data json.RawMessage
		if json.Valid([]byte(sc.Payload)) {
			data = json.RawMessage(sc.Payload)
		}
		out = append(out, model.TrackingItem{Scan: sc, Data: data, Foto: fotoList(sc.BatchID)})
	}
	return out
}

// Tracking: daftar batch dari barcode perangkat, terbaru dulu.
func (h *Handler) Tracking(c fiber.Ctx) error {
	list, err := h.Store.RecentTracking(intQuery(c, "limit", 50, 1, 500))
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Gagal membaca riwayat tracking: "+err.Error())
	}
	return ok(c, toTracking(list))
}

// TrackingDetail: satu batch lengkap berdasarkan kode box.
func (h *Handler) TrackingDetail(c fiber.Ctx) error {
	sc, err := h.Store.TrackingByBatch(c.Params("kode"))
	if err != nil {
		return err
	}
	if sc == nil {
		return fail(c, fiber.StatusNotFound, "Batch tidak ditemukan")
	}
	return ok(c, toTracking([]model.Scan{*sc})[0])
}
