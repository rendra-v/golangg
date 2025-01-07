package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Struct Retur untuk merepresentasikan data retur yang ada di database
type Retur struct {
	ID          int    `json:"id"`
	Barang      string `json:"barang"`
	Alasan      string `json:"alasan"`
	Status      string `json:"status"`
	Pengembalian string `json:"pengembalian"`
}

// Stack adalah implementasi stack generik dengan tipe data generik (T)
type Stack[T any] struct {
	items []T // Slice yang menyimpan item dalam stack
}

// Fungsi Push digunakan untuk menambah item ke dalam stack
func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item) // Menambahkan item ke stack (tumpukan)
}

// Fungsi Pop digunakan untuk mengeluarkan item dari stack dan mengembalikannya
func (s *Stack[T]) Pop() (T, bool) {
	if len(s.items) == 0 { // Jika stack kosong, kembalikan zero value dari T
		var zero T
		return zero, false
	}
	// Mengambil item terakhir yang ada di stack
	item := s.items[len(s.items)-1]
	// Menghapus item terakhir dari stack
	s.items = s.items[:len(s.items)-1]
	return item, true // Mengembalikan item yang dikeluarkan dan status sukses
}

// Fungsi IsEmpty untuk mengecek apakah stack kosong
func (s *Stack[T]) IsEmpty() bool {
	return len(s.items) == 0 // Mengembalikan true jika stack kosong
}

var (
	db           *gorm.DB        // Variabel untuk koneksi database
	deletedStack Stack[Retur]    // Stack untuk menyimpan data retur yang dihapus (untuk undo)
)

// Fungsi initDB digunakan untuk menginisialisasi koneksi database menggunakan GORM
func initDB() {
	var err error
	dsn := "root:@tcp(127.0.0.1:3306)/retur_db?charset=utf8mb4&parseTime=True&loc=Local"
	// Membuka koneksi database menggunakan GORM
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		// Jika koneksi gagal, aplikasi akan berhenti dan menampilkan error
		panic("Failed to connect to database: " + err.Error())
	}
	// Melakukan auto-migrasi, yaitu memastikan tabel untuk struct Retur ada di database
	db.AutoMigrate(&Retur{})
}

// Fungsi respondJSON digunakan untuk mengirimkan response dalam format JSON
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json") // Menyatakan bahwa konten yang dikirimkan adalah JSON
	w.WriteHeader(status)                             // Menentukan status HTTP yang dikirimkan
	json.NewEncoder(w).Encode(payload)                // Mengubah payload ke format JSON dan mengirimkannya ke response
}

// Fungsi handleError digunakan untuk mengirimkan error dalam format JSON
func handleError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message}) // Mengirimkan pesan error dalam format JSON
}

// Fungsi getReturs digunakan untuk mengambil semua data retur dari database
func getReturs(w http.ResponseWriter, r *http.Request) {
	var returs []Retur
	// Mengambil semua data retur dari database
	if err := db.Find(&returs).Error; err != nil {
		// Jika ada error, kirimkan error response
		handleError(w, http.StatusInternalServerError, "Failed to retrieve returns")
		return
	}
	// Kirimkan response dengan data retur dalam format JSON
	respondJSON(w, http.StatusOK, returs)
}

// Fungsi createRetur digunakan untuk membuat data retur baru dan menyimpannya ke database
func createRetur(w http.ResponseWriter, r *http.Request) {
	var newRetur Retur
	// Mendecode data JSON dari request body ke dalam struct Retur
	if err := json.NewDecoder(r.Body).Decode(&newRetur); err != nil {
		// Jika input tidak valid, kirimkan pesan error
		handleError(w, http.StatusBadRequest, "Invalid input")
		return
	}
	newRetur.Status = "Dalam Proses" // Set status awal retur menjadi "Dalam Proses"
	// Menyimpan data retur baru ke database
	if err := db.Create(&newRetur).Error; err != nil {
		// Jika gagal menyimpan data, kirimkan pesan error
		handleError(w, http.StatusInternalServerError, "Failed to create return")
		return
	}
	// Kirimkan data retur yang baru dibuat dalam format JSON
	respondJSON(w, http.StatusCreated, newRetur)
}

// Fungsi approveReturHandler digunakan untuk mengubah status retur menjadi "Disetujui"
func approveReturHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)              // Mendapatkan parameter id dari URL
	id, err := strconv.Atoi(vars["id"]) // Mengonversi id menjadi integer
	if err != nil {
		handleError(w, http.StatusBadRequest, "Invalid ID format") // Jika id invalid, kirimkan error
		return
	}

	var input struct {
		Pengembalian string `json:"pengembalian"` // Input untuk pengembalian (barang atau uang)
	}
	// Mendecode input dari request body
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		handleError(w, http.StatusBadRequest, "Invalid input") // Jika input tidak valid, kirimkan error
		return
	}

	// Pastikan nilai pengembalian hanya bisa "barang" atau "uang"
	if input.Pengembalian != "barang" && input.Pengembalian != "uang" {
		handleError(w, http.StatusBadRequest, "Pengembalian must be 'barang' or 'uang'") // Jika input pengembalian tidak valid, kirimkan error
		return
	}

	var retur Retur
	// Mencari retur berdasarkan id
	if err := db.First(&retur, id).Error; err != nil {
		handleError(w, http.StatusNotFound, "Return not found") // Jika retur tidak ditemukan, kirimkan error
		return
	}

	// Update status retur dan jenis pengembalian
	retur.Pengembalian = input.Pengembalian
	retur.Status = "Disetujui"
	// Menyimpan perubahan data retur ke database
	if err := db.Save(&retur).Error; err != nil {
		handleError(w, http.StatusInternalServerError, "Failed to update return") // Jika gagal menyimpan perubahan, kirimkan error
		return
	}
	// Kirimkan data retur yang sudah diperbarui dalam format JSON
	respondJSON(w, http.StatusOK, retur)
}

// Fungsi deleteReturHandler digunakan untuk menghapus data retur
func deleteReturHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"]) // Mengambil id dari URL dan mengonversinya menjadi integer
	if err != nil {
		handleError(w, http.StatusBadRequest, "Invalid ID format") // Jika id invalid, kirimkan error
		return
	}

	var retur Retur
	// Mencari retur berdasarkan id
	if err := db.First(&retur, id).Error; err != nil {
		handleError(w, http.StatusNotFound, "Return not found") // Jika retur tidak ditemukan, kirimkan error
		return
	}

	// Memasukkan retur yang akan dihapus ke dalam stack agar bisa di-undo
	deletedStack.Push(retur)
	// Menghapus data retur dari database
	if err := db.Delete(&retur).Error; err != nil {
		handleError(w, http.StatusInternalServerError, "Failed to delete return") // Jika gagal menghapus, kirimkan error
		return
	}
	// Kirimkan pesan sukses jika data berhasil dihapus
	respondJSON(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("Return with ID %d deleted", id)})
}

// Fungsi undoDeleteReturHandler digunakan untuk mengembalikan data retur yang sudah dihapus
func undoDeleteReturHandler(w http.ResponseWriter, r *http.Request) {
	// Jika tidak ada data yang dihapus (stack kosong), kirimkan error
	if deletedStack.IsEmpty() {
		handleError(w, http.StatusBadRequest, "No returns to undo")
		return
	}

	// Mengambil data retur terakhir yang dihapus dari stack
	item, _ := deletedStack.Pop()
	// Mengembalikan data retur ke database
	if err := db.Create(&item).Error; err != nil {
		handleError(w, http.StatusInternalServerError, "Failed to restore return") // Jika gagal mengembalikan, kirimkan error
		return
	}
	// Kirimkan data retur yang dikembalikan dalam format JSON
	respondJSON(w, http.StatusOK, item)
}

func main() {
	initDB() // Inisialisasi koneksi database

	r := mux.NewRouter() // Membuat router baru untuk routing request
	r.HandleFunc("/retur", getReturs).Methods("GET")
	r.HandleFunc("/retur", createRetur).Methods("POST")
	r.HandleFunc("/retur/{id}/approve", approveReturHandler).Methods("POST")
	r.HandleFunc("/retur/{id}/delete", deleteReturHandler).Methods("DELETE")
	r.HandleFunc("/retur/undo", undoDeleteReturHandler).Methods("POST")

	// Menjalankan server di port 8080
	http.ListenAndServe(":8080", r)
}