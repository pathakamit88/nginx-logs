package ossearch

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	opensearch "github.com/opensearch-project/opensearch-go"
	opensearchapi "github.com/opensearch-project/opensearch-go/opensearchapi"
	"io"
	"net/http"
	"nginx-log/pkg/config"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const IndexName = "mpop-lke-logs*"

type searchResponse struct {
	Hits struct {
		Hits []struct {
			Source struct {
				Message   string `json:"message"`
				Timestamp string `json:"@timestamp"`
			} `json:"_source"`
			Sort []int64 `json:"sort"`
		} `json:"hits"`
	} `json:"hits"`
}

type message struct {
	Host                 string    `json:"host"`
	Status               string    `json:"status"`
	ProxyProtocolAddr    string    `json:"proxy_protocol_addr"`
	TrueClientIp         string    `json:"true_client_ip"`
	TenantId             string    `json:"tenant_id"`
	UserId               string    `json:"user_id"`
	Time                 time.Time `json:"time"`
	SslProtocol          string    `json:"ssl_protocol"`
	SslClientFingerprint string    `json:"ssl_client_fingerprint"`
	RequestTime          string    `json:"request_time"`
	UpstreamResponseTime string    `json:"upstream_response_time"`
	Request              string    `json:"request"`
	Referer              string    `json:"referer"`
	UserAgent            string    `json:"user_agent"`
}

type requestStat struct {
	Request string
	Count   int
	Min     float64
	Avg     float64
	Max     float64
}

func parseInterval(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		fmt.Println("Invalid interval format. Use 1m, 15m, 30m, 1h etc.")
		os.Exit(1)
	}
	return d
}

func parseResult(body []byte) ([]string, int64, error) {
	var resp searchResponse
	err := json.Unmarshal(body, &resp)
	if err != nil {
		return nil, 0, err
	}
	messages := make([]string, 0, len(resp.Hits.Hits))
	var lastSort int64
	for i, hit := range resp.Hits.Hits {
		messages = append(messages, hit.Source.Message)
		if i == len(resp.Hits.Hits)-1 {
			lastSort = hit.Sort[0]
		}
	}
	return messages, lastSort, nil
}

func executeSearch(ctx context.Context, client *opensearch.Client, gteTime string, sort int64) ([]string, int64, error) {
	var query string
	if sort != 0 {
		query = fmt.Sprintf(`{
        "query": {
            "bool": {
                "filter": [
                    { "term": { "dissect.namespace": "pulsar" } },
                    { "term": { "dissect.container_name": "nginx" } },
                    { "range": { "@timestamp": { "gte": "%s" } } }
                ]
            }
        },
        "search_after": [%d],
        "sort": [ { "@timestamp": "asc" } ],
        "_source": [ "message", "@timestamp" ],
        "size": 100
    }`, gteTime, sort)
	} else {
		query = fmt.Sprintf(`{
        "query": {
            "bool": {
                "filter": [
                    { "term": { "dissect.namespace": "pulsar" } },
                    { "term": { "dissect.container_name": "nginx" } },
                    { "range": { "@timestamp": { "gte": "%s" } } }
                ]
            }
        },
        "sort": [ { "@timestamp": "asc" } ],
        "_source": [ "message", "@timestamp" ],
        "size": 100
    }`, gteTime)
	}

	//fmt.Println(query)
	content := strings.NewReader(query)

	search := opensearchapi.SearchRequest{
		Index: []string{IndexName},
		Body:  content,
	}

	searchResponse, err := search.Do(ctx, client)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search document: %w", err)
	}
	if searchResponse.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("search request failed with status code %d", searchResponse.StatusCode)
	}
	defer searchResponse.Body.Close()

	body, err := io.ReadAll(searchResponse.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response body: %w", err)
	}

	return parseResult(body)
}

func printRequestTimes(requestTimes map[string][]float64) {
	//fmt.Println("Total number unique requests:", len(requestTimes))
	stats := make([]requestStat, 0)
	for request, times := range requestTimes {
		count := len(times)
		if count == 0 {
			continue
		}
		var sum, minTime, maxTime float64
		minTime = times[0]
		maxTime = times[0]
		for _, t := range times {
			sum += t
			if t < minTime {
				minTime = t
			}
			if t > maxTime {
				maxTime = t
			}
		}
		avgTime := sum / float64(len(times))
		if avgTime < 1.0 && count == 1 {
			continue // Discard records with avgTime 0.00
		}
		stats = append(stats, requestStat{
			Request: request,
			Count:   count,
			Min:     minTime,
			Avg:     avgTime,
			Max:     maxTime,
		})
	}

	// Sort by Avg descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Avg > stats[j].Avg
	})

	fmt.Printf("%-10s %-10s %-10s %-10s %-s\n", "min", "avg", "max", "Count", "Path")
	for _, stat := range stats {
		fmt.Printf("%-10.2f %-10.2f %-10.2f %-10d %-s\n", stat.Min, stat.Avg, stat.Max, stat.Count, stat.Request)
	}
}

func GetResponse(ctx context.Context, cfg *config.Config, intervalStr string) {

	client, err := opensearch.NewClient(opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
	})
	if err != nil {
		fmt.Println("cannot initialize", err)
		os.Exit(1)
	}

	interval := parseInterval(intervalStr)
	gteTime := time.Now().UTC().Add(-interval).Format("2006-01-02T15:04:05")

	fullMessages := make([]string, 0)
	messages, sort, err := executeSearch(ctx, client, gteTime, 0)
	if err != nil {
		fmt.Println("failed to execute initial executeSearch:", err)
		os.Exit(1)
	}
	for len(messages) > 0 {
		fullMessages = append(fullMessages, messages...)
		messages, sort, err = executeSearch(ctx, client, gteTime, sort)
		if err != nil {
			fmt.Println("failed to execute executeSearch:", err)
			break
		}
	}
	fmt.Println("Total number of requests:", len(fullMessages))
	requestTimes := parseMessages(fullMessages)
	printRequestTimes(requestTimes)
}

func parseRequest(request string) string {
	// Remove everything after '?'
	result := strings.SplitN(request, "?", 2)[0]
	parts := strings.Split(result, "/")
	re := regexp.MustCompile(`^[A-Za-z0-9_-]{22}$`)
	for i, part := range parts {
		if re.MatchString(part) {
			parts[i] = "XXXXXXXXXXXXXXX" // Mask the 22-character string
		}
	}
	return strings.Join(parts, "/")
}

func parseMessages(messages []string) map[string][]float64 {
	requestTimes := make(map[string][]float64)

	for _, line := range messages {
		var msg message
		err := json.Unmarshal([]byte(line), &msg)
		if err != nil {
			//fmt.Println("failed to parse message:", err, line)
			continue
		}
		requestTimeFloat, err := strconv.ParseFloat(msg.UpstreamResponseTime, 64)
		if err != nil {
			//fmt.Println("failed to convert request time:", err, msg)
			continue
		}
		request := parseRequest(msg.Request)
		requestTimes[request] = append(requestTimes[request], requestTimeFloat)
	}
	return requestTimes
}
