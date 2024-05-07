package types

type Place struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Address  string   `json:"address"`
	Phone    string   `json:"phone"`
	Location GeoPoint `json:"location"`
}

type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type PlacesResponse struct {
	Name        string  `json:"name"`
	Total       int     `json:"total"`
	Places      []Place `json:"places"`
	PrevPage    int     `json:"prev_page"`
	CurrentPage int     `json:"current_page"`
	NextPage    int     `json:"next_page"`
	LastPage    int     `json:"last_page"`
}
