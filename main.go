package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"github.com/gorilla/mux"
	"fmt"
)


type Retur struct {
	ID     int    `json:"id"`
	Barang string `json:"barang"`
	Alasan string `json:"alasan"`
}

//ini nyoba dummy data
var returs = []Retur{
	{ID: 1, Barang: "Laptop", Alasan: "Rusak layar"},
	{ID: 2, Barang: "Headset", Alasan: "Tidak berfungsi"},
}

// Handler untuk menampilkan semua data retur
func getReturs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json") 
	json.NewEncoder(w).Encode(returs)                 
}

// Handler untuk menambah retur baru
func createRetur(w http.ResponseWriter, r *http.Request) {
	var newRetur Retur
	json.NewDecoder(r.Body).Decode(&newRetur) 
	newRetur.ID = len(returs) + 1             
	returs = append(returs, newRetur)        
	w.WriteHeader(http.StatusCreated)         
	json.NewEncoder(w).Encode(newRetur)       
}

func deleteReturHandler(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan parameter ID dari URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "ID harus berupa angka", http.StatusBadRequest)
		return
	}

	// Cari dan hapus data dengan ID tersebut
	for i, retur := range returs {
		if retur.ID == id {
			// Hapus data dari slice
			returs = append(returs[:i], returs[i+1:]...)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Retur dengan ID %d berhasil dihapus", id)
			return
		}
	}

	// Jika tidak ditemukan
	http.Error(w, "Retur tidak ditemukan", http.StatusNotFound)
}

func main() {
	http.HandleFunc("/retur", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			getReturs(w, r)
		} else if r.Method == "POST" {
			createRetur(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	http.ListenAndServe(":8080", nil)
}
