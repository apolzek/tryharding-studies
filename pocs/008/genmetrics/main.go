package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
)

// Structures for YAML file
type Config struct {
	Metrics []MetricConfig `yaml:"metrics"`
}

type MetricConfig struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Labels      []LabelConfig `yaml:"labels"`
}

type LabelConfig struct {
	Name            string      `yaml:"name"`
	Values          interface{} `yaml:"values"`
	HighCardinality bool        `yaml:"high_cardinality"`
	Count           int         `yaml:"count"`
	Pattern         string      `yaml:"pattern,omitempty"`
	Length          int         `yaml:"length,omitempty"`
}

type MetricsCollector struct {
	config  Config
	metrics map[string]*prometheus.GaugeVec
}

func main() {

	rand.Seed(time.Now().UnixNano())

	port := getEnvWithDefault("PORT", "8090")

	config, err := loadConfig("metrics.yaml")
	if err != nil {
		log.Fatal("Error loading configuration:", err)
	}

	collector := NewMetricsCollector(config)

	go collector.updateMetrics()

	http.Handle("/metrics", promhttp.Handler())

	// Add health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Add endpoint to show metrics information
	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		info := collector.getMetricsInfo()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(info))
	})

	// Add debug endpoint to view metrics in text format
	http.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		totalSeries := 0
		for _, metricConfig := range collector.config.Metrics {
			combinations := len(collector.generateLabelCombinations(metricConfig.Labels))
			totalSeries += combinations
			fmt.Fprintf(w, "Metric: %s\n", metricConfig.Name)
			fmt.Fprintf(w, "  Description: %s\n", metricConfig.Description)
			fmt.Fprintf(w, "  Generated series: %d\n", combinations)
			fmt.Fprintf(w, "  Labels: ")
			for i, label := range metricConfig.Labels {
				if i > 0 {
					fmt.Fprintf(w, ", ")
				}
				fmt.Fprintf(w, "%s", label.Name)
				if label.HighCardinality {
					if label.Pattern != "" {
						fmt.Fprintf(w, " (high cardinality: %d values, pattern: %s, length: %d)",
							label.Count, label.Pattern, label.Length)
					} else {
						fmt.Fprintf(w, " (high cardinality: %d values)", label.Count)
					}
				} else {
					fmt.Fprintf(w, " (low cardinality)")
				}
			}
			fmt.Fprintf(w, "\n\n")
		}
		fmt.Fprintf(w, "TOTAL SERIES: %d\n", totalSeries)
	})

	log.Printf("Server started on port %s", port)
	log.Printf("Metrics available at: http://localhost:%s/metrics", port)
	log.Printf("Health check at: http://localhost:%s/health", port)
	log.Printf("Metrics info at: http://localhost:%s/info", port)
	log.Printf("Debug metrics at: http://localhost:%s/debug", port)

	// Show summary of metrics that will be exposed
	totalSeries := 0
	for _, metricConfig := range config.Metrics {
		combinations := 1
		for _, label := range metricConfig.Labels {
			if label.HighCardinality {
				combinations *= label.Count
			} else {
				switch v := label.Values.(type) {
				case []interface{}:
					combinations *= len(v)
				case []string:
					combinations *= len(v)
				}
			}
		}
		totalSeries += combinations
		log.Printf("Metric '%s' will generate %d time series", metricConfig.Name, combinations)
	}
	log.Printf("TOTAL: %d time series will be exposed at /metrics", totalSeries)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func loadConfig(filename string) (Config, error) {
	var config Config

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, fmt.Errorf("error reading file: %v", err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return config, fmt.Errorf("error parsing YAML: %v", err)
	}

	return config, nil
}

func NewMetricsCollector(config Config) *MetricsCollector {
	collector := &MetricsCollector{
		config:  config,
		metrics: make(map[string]*prometheus.GaugeVec),
	}

	// Create metrics based on configuration
	for _, metricConfig := range config.Metrics {
		labelNames := make([]string, len(metricConfig.Labels))
		for i, label := range metricConfig.Labels {
			labelNames[i] = label.Name
		}

		gauge := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: metricConfig.Name,
				Help: metricConfig.Description,
			},
			labelNames,
		)

		collector.metrics[metricConfig.Name] = gauge
		// Register each metric individually in the default registry
		prometheus.MustRegister(gauge)
	}

	// Generate initial metric values
	collector.generateMetricValues()

	return collector
}

// Implements prometheus.Collector interface
func (c *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range c.metrics {
		metric.Describe(ch)
	}
}

func (c *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	for _, metric := range c.metrics {
		metric.Collect(ch)
	}
}

func (c *MetricsCollector) updateMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.generateMetricValues()
		}
	}
}

func (c *MetricsCollector) generateMetricValues() {
	for _, metricConfig := range c.config.Metrics {
		gauge := c.metrics[metricConfig.Name]

		// Generate all label combinations
		labelCombinations := c.generateLabelCombinations(metricConfig.Labels)

		for _, labels := range labelCombinations {
			// Generate random value for metric based on type
			var value float64

			switch metricConfig.Name {
			case "device_status":
				// 0 or 1 for status
				value = float64(rand.Intn(2))
			case "request_latency":
				// Latency between 10ms and 2000ms
				value = float64(rand.Intn(1990) + 10)
			case "error_count":
				// Error count between 0 and 50
				value = float64(rand.Intn(51))
			case "memory_usage":
				// Memory usage between 100MB and 8GB (in bytes)
				value = float64(rand.Intn(8000000000-100000000) + 100000000)
			case "network_bytes_total":
				// Bytes transferred (large cumulative values)
				value = float64(rand.Intn(1000000000) + 1000000)
			default:
				// Default random value
				value = rand.Float64() * 100
			}

			gauge.WithLabelValues(labels...).Set(value)
		}

		log.Printf("Metric %s updated with %d label combinations",
			metricConfig.Name, len(c.generateLabelCombinations(metricConfig.Labels)))
	}
}

func (c *MetricsCollector) generateLabelCombinations(labels []LabelConfig) [][]string {
	if len(labels) == 0 {
		return [][]string{{}}
	}

	// Generate values for each label
	labelValues := make([][]string, len(labels))

	for i, label := range labels {
		labelValues[i] = c.generateLabelValues(label)
	}

	// Generate cartesian product of combinations
	return c.cartesianProduct(labelValues)
}

func (c *MetricsCollector) generateLabelValues(label LabelConfig) []string {
	var values []string

	if label.HighCardinality {
		if label.Pattern != "" {
			// Use regex pattern to generate values
			values = c.generateValuesFromPattern(label.Pattern, label.Length, label.Count)
		} else {
			// Fallback to old template system for backward compatibility
			template := label.Values.(string)
			for i := 0; i < label.Count; i++ {
				var value string
				if strings.Contains(template, "<randomNum>") {
					value = strings.Replace(template, "<randomNum>", strconv.Itoa(i+1), -1)
				} else {
					value = fmt.Sprintf("%s_%d", template, i+1)
				}
				values = append(values, value)
			}
		}
	} else {
		// For low cardinality, use fixed values
		switch v := label.Values.(type) {
		case []interface{}:
			for _, val := range v {
				values = append(values, val.(string))
			}
		case []string:
			values = v
		}
	}

	return values
}

func (c *MetricsCollector) generateValuesFromPattern(pattern string, length int, count int) []string {
	var values []string

	for i := 0; i < count; i++ {
		value := c.generateStringFromPattern(pattern, length)
		values = append(values, value)
	}

	return values
}

func (c *MetricsCollector) generateStringFromPattern(pattern string, length int) string {
	if length <= 0 {
		length = 10 // default length
	}

	var result strings.Builder

	// Generate string based on pattern
	switch pattern {
	case "[a-z]":
		// Generate lowercase letters
		for i := 0; i < length; i++ {
			result.WriteByte(byte('a' + rand.Intn(26)))
		}
	case "[A-Z]":
		// Generate uppercase letters
		for i := 0; i < length; i++ {
			result.WriteByte(byte('A' + rand.Intn(26)))
		}
	case "[0-9]":
		// Generate numbers
		for i := 0; i < length; i++ {
			result.WriteByte(byte('0' + rand.Intn(10)))
		}
	case "[a-zA-Z]":
		// Generate mixed case letters
		for i := 0; i < length; i++ {
			if rand.Intn(2) == 0 {
				result.WriteByte(byte('a' + rand.Intn(26)))
			} else {
				result.WriteByte(byte('A' + rand.Intn(26)))
			}
		}
	case "[a-zA-Z0-9]":
		// Generate alphanumeric
		chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		for i := 0; i < length; i++ {
			result.WriteByte(chars[rand.Intn(len(chars))])
		}
	case "[a-z0-9]":
		// Generate lowercase alphanumeric
		chars := "abcdefghijklmnopqrstuvwxyz0123456789"
		for i := 0; i < length; i++ {
			result.WriteByte(chars[rand.Intn(len(chars))])
		}
	default:
		// Try to parse as actual regex for more complex patterns
		if matched, _ := regexp.MatchString(`^\[[^\]]+\]$`, pattern); matched {
			// Extract character class from pattern
			charClass := pattern[1 : len(pattern)-1]
			chars := c.expandCharacterClass(charClass)
			for i := 0; i < length; i++ {
				if len(chars) > 0 {
					result.WriteByte(chars[rand.Intn(len(chars))])
				} else {
					result.WriteByte('x') // fallback
				}
			}
		} else {
			// Fallback: generate alphanumeric
			chars := "abcdefghijklmnopqrstuvwxyz0123456789"
			for i := 0; i < length; i++ {
				result.WriteByte(chars[rand.Intn(len(chars))])
			}
		}
	}

	return result.String()
}

func (c *MetricsCollector) expandCharacterClass(charClass string) string {
	var chars strings.Builder

	i := 0
	for i < len(charClass) {
		if i+2 < len(charClass) && charClass[i+1] == '-' {
			// Range like a-z, A-Z, 0-9
			start := charClass[i]
			end := charClass[i+2]
			for ch := start; ch <= end; ch++ {
				chars.WriteByte(ch)
			}
			i += 3
		} else {
			// Single character
			chars.WriteByte(charClass[i])
			i++
		}
	}

	return chars.String()
}

func (c *MetricsCollector) cartesianProduct(sets [][]string) [][]string {
	if len(sets) == 0 {
		return [][]string{}
	}

	if len(sets) == 1 {
		result := make([][]string, len(sets[0]))
		for i, v := range sets[0] {
			result[i] = []string{v}
		}
		return result
	}

	result := [][]string{}
	rest := c.cartesianProduct(sets[1:])

	for _, v := range sets[0] {
		for _, r := range rest {
			combination := append([]string{v}, r...)
			result = append(result, combination)
		}
	}

	return result
}

func (c *MetricsCollector) getMetricsInfo() string {
	info := `{
  "metrics": [`

	for i, metricConfig := range c.config.Metrics {
		if i > 0 {
			info += ","
		}

		totalCombinations := 1
		for _, label := range metricConfig.Labels {
			if label.HighCardinality {
				totalCombinations *= label.Count
			} else {
				switch v := label.Values.(type) {
				case []interface{}:
					totalCombinations *= len(v)
				case []string:
					totalCombinations *= len(v)
				}
			}
		}

		info += fmt.Sprintf(`
    {
      "name": "%s",
      "description": "%s",
      "total_series": %d,
      "labels": [`, metricConfig.Name, metricConfig.Description, totalCombinations)

		for j, label := range metricConfig.Labels {
			if j > 0 {
				info += ","
			}

			patternInfo := ""
			if label.Pattern != "" {
				patternInfo = fmt.Sprintf(`, "pattern": "%s", "length": %d`, label.Pattern, label.Length)
			}

			info += fmt.Sprintf(`
        {
          "name": "%s",
          "high_cardinality": %t,
          "count": %d%s
        }`, label.Name, label.HighCardinality, label.Count, patternInfo)
		}

		info += `
      ]
    }`
	}

	info += `
  ]
}`

	return info
}
