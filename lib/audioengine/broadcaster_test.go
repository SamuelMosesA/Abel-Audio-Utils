package audioengine

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestCalculatePeakMeters(t *testing.T) {
	buffer := []float32{0.5, -0.1, 0.2, 0.8}
	maxL, maxR := CalculatePeakMeters(buffer)
	assert.Equal(t, float32(0.5), maxL)
	assert.Equal(t, float32(0.8), maxR)
}
