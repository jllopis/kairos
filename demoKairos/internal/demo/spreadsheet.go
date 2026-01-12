package demo

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// QuerySpec defines a safe query the spreadsheet agent can execute.
type QuerySpec struct {
	Type    string `json:"type"`
	Quarter string `json:"quarter,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Month   string `json:"month,omitempty"`
}

// QueryResult returns a structured payload with rows and trace metadata.
type QueryResult struct {
	Headers []string               `json:"headers"`
	Rows    [][]string             `json:"rows"`
	Meta    map[string]interface{} `json:"meta"`
}

type sheetData struct {
	headers []string
	rows    []map[string]string
}

// SpreadsheetStore loads CSV data and exposes query helpers.
type SpreadsheetStore struct {
	sheets map[string]sheetData
}

// LoadSpreadsheetStore reads CSV files from a directory.
func LoadSpreadsheetStore(dir string) (*SpreadsheetStore, error) {
	store := &SpreadsheetStore{sheets: make(map[string]sheetData)}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".csv") {
			continue
		}
		sheetName := strings.TrimSuffix(name, filepath.Ext(name))
		path := filepath.Join(dir, name)
		headers, rows, err := readCSV(path)
		if err != nil {
			return nil, err
		}
		store.sheets[titleCase(sheetName)] = sheetData{headers: headers, rows: rows}
	}
	return store, nil
}

// Sheets returns available sheet names.
func (s *SpreadsheetStore) Sheets() []string {
	out := make([]string, 0, len(s.sheets))
	for name := range s.sheets {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Schema returns headers for a sheet.
func (s *SpreadsheetStore) Schema(sheet string) ([]string, error) {
	data, ok := s.sheets[sheet]
	if !ok {
		return nil, fmt.Errorf("sheet %q not found", sheet)
	}
	return append([]string(nil), data.headers...), nil
}

// Query executes a safe, predefined query.
func (s *SpreadsheetStore) Query(spec QuerySpec) (QueryResult, error) {
	switch spec.Type {
	case "sales_by_region":
		return s.salesByRegion(spec)
	case "top_products_margin_compare":
		return s.topProductsMarginCompare(spec)
	case "gastos_anomalies":
		return s.gastosAnomalies(spec)
	default:
		return QueryResult{}, fmt.Errorf("unsupported query type %q", spec.Type)
	}
}

func (s *SpreadsheetStore) salesByRegion(spec QuerySpec) (QueryResult, error) {
	data, ok := s.sheets["Ventas"]
	if !ok {
		return QueryResult{}, fmt.Errorf("Ventas sheet not found")
	}
	quarter := spec.Quarter
	if quarter == "" {
		quarter = "Q4"
	}
	byRegion := make(map[string]float64)
	for _, row := range data.rows {
		if row["quarter"] != quarter {
			continue
		}
		value, err := strconv.ParseFloat(row["net_sales"], 64)
		if err != nil {
			continue
		}
		byRegion[row["region"]] += value
	}
	regions := make([]string, 0, len(byRegion))
	for region := range byRegion {
		regions = append(regions, region)
	}
	sort.Strings(regions)
	rows := make([][]string, 0, len(regions))
	for _, region := range regions {
		rows = append(rows, []string{region, fmt.Sprintf("%.2f", byRegion[region])})
	}
	return QueryResult{
		Headers: []string{"region", "net_sales_total"},
		Rows:    rows,
		Meta: map[string]interface{}{
			"sheet":   "Ventas",
			"quarter": quarter,
		},
	}, nil
}

func (s *SpreadsheetStore) topProductsMarginCompare(spec QuerySpec) (QueryResult, error) {
	data, ok := s.sheets["Ventas"]
	if !ok {
		return QueryResult{}, fmt.Errorf("Ventas sheet not found")
	}
	quarter := spec.Quarter
	if quarter == "" {
		quarter = "Q4"
	}
	prevQuarter := previousQuarter(quarter)
	if prevQuarter == "" {
		return QueryResult{}, fmt.Errorf("cannot derive previous quarter from %q", quarter)
	}
	limit := spec.Limit
	if limit <= 0 {
		limit = 10
	}
	current := make(map[string][]float64)
	prev := make(map[string][]float64)
	currentSales := make(map[string]float64)
	for _, row := range data.rows {
		margin, err := strconv.ParseFloat(row["margin"], 64)
		if err != nil {
			continue
		}
		sales, err := strconv.ParseFloat(row["net_sales"], 64)
		if err != nil {
			continue
		}
		product := row["product"]
		switch row["quarter"] {
		case quarter:
			current[product] = append(current[product], margin)
			currentSales[product] += sales
		case prevQuarter:
			prev[product] = append(prev[product], margin)
		}
	}
	type entry struct {
		product    string
		marginNow  float64
		marginPrev float64
		delta      float64
		salesNow   float64
	}
	entries := make([]entry, 0, len(current))
	for product, margins := range current {
		if len(margins) == 0 {
			continue
		}
		marginNow := average(margins)
		marginPrev := average(prev[product])
		delta := marginNow - marginPrev
		entries = append(entries, entry{
			product:    product,
			marginNow:  marginNow,
			marginPrev: marginPrev,
			delta:      delta,
			salesNow:   currentSales[product],
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].marginNow > entries[j].marginNow
	})
	if limit > len(entries) {
		limit = len(entries)
	}
	rows := make([][]string, 0, limit)
	for _, e := range entries[:limit] {
		rows = append(rows, []string{
			e.product,
			fmt.Sprintf("%.2f", e.marginNow),
			fmt.Sprintf("%.2f", e.marginPrev),
			fmt.Sprintf("%.2f", e.delta),
			fmt.Sprintf("%.2f", e.salesNow),
		})
	}
	return QueryResult{
		Headers: []string{"product", "margin_current", "margin_prev", "margin_delta", "net_sales_current"},
		Rows:    rows,
		Meta: map[string]interface{}{
			"sheet":        "Ventas",
			"quarter":      quarter,
			"prev_quarter": prevQuarter,
			"limit":        limit,
		},
	}, nil
}

func (s *SpreadsheetStore) gastosAnomalies(spec QuerySpec) (QueryResult, error) {
	data, ok := s.sheets["Gastos"]
	if !ok {
		return QueryResult{}, fmt.Errorf("Gastos sheet not found")
	}
	month := spec.Month
	if month == "" {
		month = latestMonth(data.rows)
	}
	var values []float64
	for _, row := range data.rows {
		if !strings.HasPrefix(row["date"], month) {
			continue
		}
		amount, err := strconv.ParseFloat(row["amount"], 64)
		if err != nil {
			continue
		}
		values = append(values, amount)
	}
	mean := average(values)
	std := stddev(values, mean)
	threshold := mean + 2*std
	rows := make([][]string, 0)
	for _, row := range data.rows {
		if !strings.HasPrefix(row["date"], month) {
			continue
		}
		amount, err := strconv.ParseFloat(row["amount"], 64)
		if err != nil {
			continue
		}
		if amount > threshold {
			rows = append(rows, []string{row["date"], row["category"], row["department"], fmt.Sprintf("%.2f", amount)})
		}
	}
	return QueryResult{
		Headers: []string{"date", "category", "department", "amount"},
		Rows:    rows,
		Meta: map[string]interface{}{
			"sheet":     "Gastos",
			"month":     month,
			"mean":      mean,
			"stddev":    std,
			"threshold": threshold,
		},
	}, nil
}

func readCSV(path string) ([]string, []map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	headers, err := reader.Read()
	if err != nil {
		return nil, nil, err
	}
	var rows []map[string]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		row := make(map[string]string, len(headers))
		for i, header := range headers {
			if i < len(record) {
				row[header] = record[i]
			}
		}
		rows = append(rows, row)
	}
	return headers, rows, nil
}

func previousQuarter(q string) string {
	switch strings.ToUpper(q) {
	case "Q1":
		return "Q4"
	case "Q2":
		return "Q1"
	case "Q3":
		return "Q2"
	case "Q4":
		return "Q3"
	default:
		return ""
	}
}

func latestMonth(rows []map[string]string) string {
	latest := time.Time{}
	for _, row := range rows {
		t, err := time.Parse("2006-01-02", row["date"])
		if err != nil {
			continue
		}
		if t.After(latest) {
			latest = t
		}
	}
	if latest.IsZero() {
		return ""
	}
	return latest.Format("2006-01")
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stddev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		delta := v - mean
		sum += delta * delta
	}
	return math.Sqrt(sum / float64(len(values)))
}

func titleCase(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
