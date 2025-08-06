package logger

import (
	"bytes"
	"encoding/json"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log := New(InfoLevel)
	log.logger = zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(&buf),
			zap.InfoLevel,
		)).Sugar()

	log.Info("test message", "key", "value")
	log.Sync()

	var logData map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logData)
	assert.NoError(t, err)

	assert.Equal(t, "test message", logData["msg"])
	assert.Equal(t, "value", logData["key"])
	assert.Equal(t, "info", logData["level"])
}

func TestLogLevels(t *testing.T) {
	levels := []struct {
		level    Level
		funcName string
	}{
		{DebugLevel, "Debug"},
		{InfoLevel, "Info"},
		{WarnLevel, "Warn"},
		{ErrorLevel, "Error"},
	}

	for _, l := range levels {
		t.Run(l.funcName, func(t *testing.T) {
			log := New(l.level)
			assert.NotNil(t, log)
		})
	}
}
