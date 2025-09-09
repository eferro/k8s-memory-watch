package monitor

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/eduardoferro/k8s-memory-watch/internal/config"
	"github.com/eduardoferro/k8s-memory-watch/internal/k8s"
)

// CSVFormatter handles CSV output formatting for memory reports
type CSVFormatter struct {
	writer *csv.Writer
}

// NewCSVFormatter creates a new CSV formatter
func NewCSVFormatter() *CSVFormatter {
	return &CSVFormatter{
		writer: csv.NewWriter(os.Stdout),
	}
}

// FormatReport formats and prints the memory report as CSV
func (f *CSVFormatter) FormatReport(report *MemoryReport, cfg *config.Config, showHeader bool) {
	defer f.writer.Flush()

	if showHeader {
		f.writeHeader(cfg)
	}

	f.writeData(report, cfg)
}

// writeHeader writes the CSV header row
func (f *CSVFormatter) writeHeader(cfg *config.Config) {
	header := f.buildHeader(cfg)
	if err := f.writer.Write(header); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing CSV header: %v\n", err)
	}
}

// buildHeader creates the CSV header based on configuration
func (f *CSVFormatter) buildHeader(cfg *config.Config) []string {
	header := []string{
		"timestamp",
		"memory_status",
		"namespace",
		"pod_name",
		"phase",
		"ready",
		"usage_bytes",
		"request_bytes",
		"limit_bytes",
		"usage_percent",
		"limit_usage_percent",
		"container_name",
	}

	// Add label columns
	for _, label := range cfg.Labels {
		header = append(header, "label_"+strings.ReplaceAll(label, ".", "_"))
	}

	// Add annotation columns
	for _, annotation := range cfg.Annotations {
		header = append(header, "annotation_"+strings.ReplaceAll(annotation, ".", "_"))
	}

	return header
}

// writeData writes the pod data rows
func (f *CSVFormatter) writeData(report *MemoryReport, cfg *config.Config) {
	for i := range report.Pods {
		pod := &report.Pods[i]
		pod.CalculateUsagePercent()

		if len(pod.Containers) > 0 {
			f.writeContainerRows(pod, cfg, report.Summary.Timestamp)
		} else {
			f.writePodRow(pod, cfg, report.Summary.Timestamp)
		}
	}
}

// writeContainerRows writes one row per container
func (f *CSVFormatter) writeContainerRows(pod *k8s.PodMemoryInfo, cfg *config.Config, timestamp time.Time) {
	for _, c := range pod.Containers {
		c.CalculateUsagePercent()
		record := buildCSVRecord(pod, &c, cfg, timestamp)
		if err := f.writer.Write(record); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing CSV record: %v\n", err)
		}
	}
}

// writePodRow writes a single row for the pod
func (f *CSVFormatter) writePodRow(pod *k8s.PodMemoryInfo, cfg *config.Config, timestamp time.Time) {
	record := buildCSVRecordForPod(pod, cfg, timestamp)
	if err := f.writer.Write(record); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing CSV record: %v\n", err)
	}
}
