package writer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/xitongsys/parquet-go-source/buffer"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go/writer"

	"github.com/janovincze/philotes/internal/cdc"
	cdcbuffer "github.com/janovincze/philotes/internal/cdc/buffer"
	"github.com/janovincze/philotes/internal/iceberg"
)

// ParquetWriter converts CDC events to Parquet format.
type ParquetWriter struct {
	// CompressionCodec is the compression codec to use.
	CompressionCodec parquet.CompressionCodec
}

// NewParquetWriter creates a new Parquet writer.
func NewParquetWriter() *ParquetWriter {
	return &ParquetWriter{
		CompressionCodec: parquet.CompressionCodec_SNAPPY,
	}
}

// ParquetResult contains the result of writing events to Parquet.
type ParquetResult struct {
	// Data is the Parquet file data.
	Data []byte

	// FileName is the generated file name.
	FileName string

	// RecordCount is the number of records written.
	RecordCount int64

	// FileSizeInBytes is the size of the Parquet data.
	FileSizeInBytes int64
}

// CDCRecord represents a CDC record for Parquet serialization.
// The struct tags define the Parquet schema.
type CDCRecord struct {
	// Data holds the actual row data as JSON string (for flexibility).
	Data string `parquet:"name=data, type=BYTE_ARRAY, convertedtype=UTF8"`

	// CDCOperation is the CDC operation type.
	CDCOperation string `parquet:"name=_cdc_operation, type=BYTE_ARRAY, convertedtype=UTF8"`

	// CDCTimestamp is the event timestamp.
	CDCTimestamp int64 `parquet:"name=_cdc_timestamp, type=INT64, convertedtype=TIMESTAMP_MILLIS"`

	// CDCLSN is the PostgreSQL LSN.
	CDCLSN string `parquet:"name=_cdc_lsn, type=BYTE_ARRAY, convertedtype=UTF8"`

	// CDCTable is the source table name.
	CDCTable string `parquet:"name=_cdc_table, type=BYTE_ARRAY, convertedtype=UTF8"`

	// CDCSchema is the source schema name.
	CDCSchema string `parquet:"name=_cdc_schema, type=BYTE_ARRAY, convertedtype=UTF8"`
}

// WriteEvents converts a slice of BufferedEvents to Parquet format.
func (p *ParquetWriter) WriteEvents(events []cdcbuffer.BufferedEvent) (*ParquetResult, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("no events to write")
	}

	// Create an in-memory buffer
	buf := new(bytes.Buffer)
	fw := buffer.NewBufferFileFromBytes(buf.Bytes())

	// Create Parquet writer
	pw, err := writer.NewParquetWriter(fw, new(CDCRecord), 4)
	if err != nil {
		return nil, fmt.Errorf("create parquet writer: %w", err)
	}

	pw.CompressionType = p.CompressionCodec
	pw.RowGroupSize = 128 * 1024 * 1024 // 128MB row groups

	// Write each event
	for _, be := range events {
		record, err := eventToRecord(be.Event)
		if err != nil {
			return nil, fmt.Errorf("convert event to record: %w", err)
		}

		if err := pw.Write(record); err != nil {
			return nil, fmt.Errorf("write record: %w", err)
		}
	}

	// Close the writer to flush
	if err := pw.WriteStop(); err != nil {
		return nil, fmt.Errorf("close parquet writer: %w", err)
	}

	// Get the written data
	data := fw.Bytes()

	// Generate unique file name
	fileName := generateFileName()

	return &ParquetResult{
		Data:            data,
		FileName:        fileName,
		RecordCount:     int64(len(events)),
		FileSizeInBytes: int64(len(data)),
	}, nil
}

// eventToRecord converts a CDC event to a CDCRecord.
func eventToRecord(event cdc.Event) (*CDCRecord, error) {
	// Get the data to serialize (prefer After for INSERT/UPDATE, Before for DELETE)
	var data map[string]any
	if event.HasAfter() {
		data = event.After
	} else if event.HasBefore() {
		data = event.Before
	} else {
		data = make(map[string]any)
	}

	// Serialize data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal data: %w", err)
	}

	return &CDCRecord{
		Data:         string(dataJSON),
		CDCOperation: string(event.Operation),
		CDCTimestamp: event.Timestamp.UnixMilli(),
		CDCLSN:       event.LSN,
		CDCTable:     event.Table,
		CDCSchema:    event.Schema,
	}, nil
}

// generateFileName generates a unique Parquet file name.
func generateFileName() string {
	id := uuid.New().String()
	timestamp := time.Now().UnixMilli()
	return fmt.Sprintf("%s-%d.parquet", id, timestamp)
}

// ResultToDataFile converts a ParquetResult to an Iceberg DataFile.
func ResultToDataFile(result *ParquetResult, basePath string, partition map[string]any) iceberg.DataFile {
	filePath := fmt.Sprintf("%s/%s", basePath, result.FileName)

	return iceberg.DataFile{
		FilePath:        filePath,
		FileFormat:      "parquet",
		RecordCount:     result.RecordCount,
		FileSizeInBytes: result.FileSizeInBytes,
		PartitionData:   partition,
	}
}
