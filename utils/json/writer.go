package utils_json

import (
	"encoding/json"
	"log"
	"net/http"
)

// Write the averages array in the HTTP ResponseWriter. The response is
// JSON-formatted.
func WriteAverages(w http.ResponseWriter, avgCollection *AvgCollection) {
	js, err := json.Marshal(avgCollection)

	if err != nil {
		log.Println("Cannot marshalize aggregated averages.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
