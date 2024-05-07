package db

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"tastydiscoveries/pkg/types"

	"github.com/elastic/go-elasticsearch/v8"
)

type Store interface {
	// returns a list of items, a total number of hits and (or) an error in case of one
	GetPlaces(limit int, offset int) ([]types.Place, int, error)
}

type ESStore struct {
	client *elasticsearch.Client
	index  string
}

func NewESStore(client *elasticsearch.Client, index string) *ESStore {
	return &ESStore{client: client, index: index}
}

func (s *ESStore) GetResponse(query map[string]interface{}) ([]types.Place, int, error) {
	body, err := json.Marshal(query)
	if err != nil {
		return nil, 0, err
	}

	resp, err := s.client.Search(
		s.client.Search.WithContext(context.Background()),
		s.client.Search.WithIndex(s.index),
		s.client.Search.WithBody(bytes.NewReader(body)),
		s.client.Search.WithTrackTotalHits(true),
		s.client.Search.WithSort("id"),
	)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return nil, 0, fmt.Errorf("error getting places: %s", resp.Status())
	}

	var searchResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return nil, 0, err
	}

	hits, ok := searchResult["hits"].(map[string]interface{})["hits"].([]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("no hits found response")
	}

	var places []types.Place
	for _, hit := range hits {
		source := hit.(map[string]interface{})["_source"]
		place := types.Place{
			ID:      int(source.(map[string]interface{})["id"].(float64)),
			Name:    source.(map[string]interface{})["name"].(string),
			Address: source.(map[string]interface{})["address"].(string),
			Phone:   source.(map[string]interface{})["phone"].(string),
			Location: types.GeoPoint{
				Lat: source.(map[string]interface{})["location"].(map[string]interface{})["lat"].(float64),
				Lon: source.(map[string]interface{})["location"].(map[string]interface{})["lon"].(float64),
			},
		}
		places = append(places, place)
	}
	totalHits := int(searchResult["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))

	return places, totalHits, nil
}

func (s *ESStore) GetPlaces(limit int, offset int) ([]types.Place, int, error) {
	query := map[string]interface{}{
		"size": limit,
		"from": offset,
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}

	return s.GetResponse(query)
}
