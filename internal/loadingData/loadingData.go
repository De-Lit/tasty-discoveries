package loadingData

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"tastydiscoveries/pkg/types"
	"time"
)

var (
	indexName  string
	filePath   string
	numWorkers int
	flushBytes int
	numItems   int
)

const elasticURL = "http://elasticsearch:9200"

func init() {
	flag.StringVar(&indexName, "index", "places", "Index name")
	flag.StringVar(&filePath, "f", "", "File path")
	flag.IntVar(&numWorkers, "workers", runtime.NumCPU(), "Number of indexer workers")
	flag.IntVar(&flushBytes, "flush", 5e+6, "Flush threshold in bytes")
	flag.IntVar(&numItems, "count", 10000, "Number of documents to generate")
	flag.Parse()
}

func Run() {

	var (
		countSuccessful uint64
		placeList       *[]types.Place
		csvData         [][]string
		es              *elasticsearch.Client
		err             error
	)

	var mapping = `{
		"settings": {
			"index": {
			  "max_result_window" : 20000
			}
		},
		"mappings": {
			"properties": {
				"name": {
					"type":"text"
				},
				"address": {
					"type":"text"
				},
				"phone": {
					"type":"text"
				},
				"location": {
					"type":"geo_point"
				}
			}
		}
	}`

	// if es, err = elasticsearch.NewDefaultClient(); err != nil {
	// 	log.Fatalf("Error creating the client: %s", err)
	// }

	cfg := elasticsearch.Config{
		Addresses: []string{
			elasticURL,
		},
	}
	if es, err = elasticsearch.NewClient(cfg); err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	exist, err := es.Indices.Exists([]string{indexName})
	if err != nil {
		log.Fatalln(err)
	}
	defer exist.Body.Close()
	if exist.StatusCode != 200 {
		es.Indices.Create(indexName, es.Indices.Create.WithBody(strings.NewReader(mapping)))
	}
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	csvReader := csv.NewReader(file)
	csvReader.Comma = '\t'
	if csvData, err = csvReader.ReadAll(); err != nil {
		log.Fatalln(err)
	}

	if placeList, err = createPlaceList(&csvData); err != nil {
		log.Fatalln(err)
	}

	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         indexName,        // The default index name
		Client:        es,               // The Elasticsearch client
		NumWorkers:    numWorkers,       // The number of worker goroutines
		FlushBytes:    int(flushBytes),  // The flush threshold in bytes
		FlushInterval: 30 * time.Second, // The periodic flush interval
	})
	if err != nil {
		log.Fatalf("Error creating the indexer: %s", err)
	}

	for _, a := range *placeList {
		// Prepare the data payload: encode article to JSON
		data, err := json.Marshal(a)
		if err != nil {
			log.Fatalf("Cannot encode article %d: %s", a.ID, err)
		}

		err = bi.Add(
			context.Background(),
			esutil.BulkIndexerItem{
				// Action field configures the operation to perform (index, create, delete, update)
				Action: "index",

				// DocumentID is the (optional) document ID
				DocumentID: strconv.Itoa(a.ID),

				// Body is an `io.Reader` with the payload
				Body: bytes.NewReader(data),

				// OnSuccess is called for each successful operation
				OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
					atomic.AddUint64(&countSuccessful, 1)
				},

				// OnFailure is called for each failed operation
				OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
					if err != nil {
						log.Printf("ERROR: %s", err)
					} else {
						log.Printf("ERROR: %s: %s", res.Error.Type, res.Error.Reason)
					}
				},
			},
		)
		if err != nil {
			log.Fatalf("Unexpected error: %s", err)
		}
	}

	if err = bi.Close(context.Background()); err != nil {
		log.Fatalf("Unexpected error: %s", err)
	}

}

func createPlaceList(data *[][]string) (*[]types.Place, error) {
	var (
		placeList []types.Place
		err       error
	)

	for i, line := range *data {
		if i > 0 {
			var place types.Place
			for j, field := range line {
				if j == 0 {
					place.ID = i
				} else if j == 1 {
					place.Name = field
				} else if j == 2 {
					place.Address = field
				} else if j == 3 {
					place.Phone = field
				} else if j == 4 {
					if place.Location.Lon, err = strconv.ParseFloat(strings.TrimSpace(field), 64); err != nil {
						return nil, err
					}
				} else if j == 5 {
					if place.Location.Lat, err = strconv.ParseFloat(strings.TrimSpace(field), 64); err != nil {
						return nil, err
					}
				} else {
					break
				}
			}
			placeList = append(placeList, place)
		}
	}
	return &placeList, nil
}
