package simplestInterface

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"tastydiscoveries/pkg/db"
	"tastydiscoveries/pkg/jwt"
	"tastydiscoveries/pkg/types"
	"text/template"

	"github.com/elastic/go-elasticsearch/v8"
)

const (
	index      = "places"
	indexPage  = "web/index.html"
	elasticURL = "http://elasticsearch:9200"
)

func getPlaces(index string, r *http.Request) (*types.PlacesResponse, error) {
	var es *elasticsearch.Client
	var err error

	cfg := elasticsearch.Config{
		Addresses: []string{
			elasticURL,
		},
	}
	if es, err = elasticsearch.NewClient(cfg); err != nil {
		return nil, err
	}
	page := r.URL.Query().Get("page")
	if page == "" {
		page = "1"
	}
	realPage, err := strconv.Atoi(page)
	if err != nil || realPage == 0 {
		return nil, errors.New("Invalid 'page' value: " + page)
	}

	s := db.NewESStore(es, index)
	pageSize := 10
	offset := (realPage - 1) * pageSize
	places, totalPlaces, err := s.GetPlaces(pageSize, offset)
	if err != nil {
		return nil, err
	}

	totalPages := totalPlaces / pageSize
	if totalPlaces%pageSize != 0 {
		totalPages++
	}

	if realPage > totalPages {
		return nil, errors.New("Invalid 'page' value: " + page)
	}

	response := types.PlacesResponse{
		Name:        "Places",
		Total:       totalPlaces,
		Places:      places,
		CurrentPage: realPage,
		PrevPage:    realPage - 1,
		NextPage:    realPage + 1,
		LastPage:    totalPages,
	}
	return &response, nil
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	response, err := getPlaces(index, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tpl := template.Must(template.ParseFiles(indexPage))
	err = tpl.Execute(w, response)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	response, err := getPlaces(index, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	responseJSON, err := json.MarshalIndent(response, "", "    ")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(responseJSON)
}

func recommendHandler(w http.ResponseWriter, r *http.Request) {
	if !isValidToken(w, r) {
		return
	}

	cfg := elasticsearch.Config{
		Addresses: []string{
			elasticURL,
		},
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		http.Error(w, "Error creating Elasticsearch client: "+err.Error(), http.StatusInternalServerError)
		return
	}

	lat := r.URL.Query().Get("lat")
	if _, err := strconv.ParseFloat(lat, 64); err != nil {
		http.Error(w, "Invalid 'latitude' value: "+lat, http.StatusBadRequest)
		return
	}

	lon := r.URL.Query().Get("lon")
	if _, err := strconv.ParseFloat(lon, 64); err != nil {
		http.Error(w, "Invalid 'longitude' value: "+lon, http.StatusBadRequest)
		return
	}

	query := map[string]interface{}{
		"sort": []map[string]interface{}{
			{
				"_geo_distance": map[string]interface{}{
					"location": map[string]interface{}{
						"lat": lat,
						"lon": lon,
					},
					"order":           "asc",
					"unit":            "km",
					"mode":            "min",
					"distance_type":   "arc",
					"ignore_unmapped": true,
				},
			},
		},
		"size": 3,
	}

	s := db.NewESStore(es, index)

	places, _, err := s.GetResponse(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	recommendation := map[string]interface{}{
		"name":   "Recommendation",
		"places": places,
	}

	jsonData, err := json.MarshalIndent(recommendation, "", "    ")
	if err != nil {
		http.Error(w, "Error encoding JSON response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func getTokenHandler(w http.ResponseWriter, r *http.Request) {
	token, err := jwt.CreateToken("admin")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	jsonData, err := json.MarshalIndent(map[string]interface{}{"token": token}, "", "    ")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(jsonData)
}

func isValidToken(w http.ResponseWriter, r *http.Request) bool {
	// Getting a JWT token from the Authorization header
	token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")

	// JWT Token Validation
	_, err := jwt.ValidateToken(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return false
	}

	// If the token is valid, continuing executing the protected endpoint
	return true
}

func Run() {
	port := "8888"

	mux := http.NewServeMux()

	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/api/places/", apiHandler)
	mux.HandleFunc("/api/recommend/", recommendHandler)
	mux.HandleFunc("/api/get_token/", getTokenHandler)
	http.ListenAndServe(":"+port, mux)
}
