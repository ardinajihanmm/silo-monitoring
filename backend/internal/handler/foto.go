package handler

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
)

// namaValid: kode box dan nama file harus diawali huruf/angka, tanpa "/" (cegah path traversal).
var namaValid = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func fotoDir() string {
	if d := os.Getenv("FOTO_DIR"); d != "" {
		return d
	}
	return "data/foto"
}

// fotoList mengembalikan nama file foto satu batch, terbaru dulu.
func fotoList(batch string) []string {
	out := []string{}
	if !namaValid.MatchString(batch) {
		return out
	}
	entries, err := os.ReadDir(filepath.Join(fotoDir(), batch))
	if err != nil {
		return out
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if !entries[i].IsDir() {
			out = append(out, entries[i].Name())
		}
	}
	return out
}

// DeviceFoto menerima foto dari kamera perangkat (multipart: batch_id, foto) saat barcode terbaca.
func (h *Handler) DeviceFoto(c fiber.Ctx) error {
	key := os.Getenv("DEVICE_KEY")
	if key == "" {
		return fail(c, fiber.StatusServiceUnavailable, "DEVICE_KEY belum diset di server")
	}
	if c.Get("X-Device-Key") != key {
		return fail(c, fiber.StatusUnauthorized, "Device key salah")
	}
	batch := c.FormValue("batch_id")
	if !namaValid.MatchString(batch) {
		return fail(c, fiber.StatusBadRequest, "batch_id tidak valid")
	}
	file, err := c.FormFile("foto")
	if err != nil {
		return fail(c, fiber.StatusBadRequest, "file foto wajib ada")
	}
	if file.Size > 5<<20 {
		return fail(c, fiber.StatusRequestEntityTooLarge, "Foto maksimal 5 MB")
	}
	dir := filepath.Join(fotoDir(), batch)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	nama := strconv.FormatInt(time.Now().UnixNano(), 10) + ".jpg"
	if err := c.SaveFile(file, filepath.Join(dir, nama)); err != nil {
		return err
	}
	return ok(c, fiber.Map{"nama": nama, "batch_id": batch})
}

// FotoFile mengirim satu foto batch.
func (h *Handler) FotoFile(c fiber.Ctx) error {
	batch, nama := c.Params("kode"), c.Params("file")
	if !namaValid.MatchString(batch) || !namaValid.MatchString(nama) {
		return fail(c, fiber.StatusBadRequest, "nama tidak valid")
	}
	path := filepath.Join(fotoDir(), batch, nama)
	if _, err := os.Stat(path); err != nil {
		return fail(c, fiber.StatusNotFound, "Foto tidak ditemukan")
	}
	return c.SendFile(path)
}
