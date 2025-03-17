package tests

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestByteLHOperation(t *testing.T) {
	var origin uint16 = 23333
	lengthLow, lengthHigh := uint8(origin&0xff), uint8((origin>>8)&0xff)
	fmt.Println(lengthHigh, lengthLow)
	fmt.Println((uint16(lengthHigh) << 8) | uint16(lengthLow))
	assert.Equal(t, origin, (uint16(lengthHigh)<<8)|uint16(lengthLow))

	bf := make([]byte, 2)
	// 写入小端
	binary.LittleEndian.PutUint16(bf, origin)
	t1 := binary.LittleEndian.Uint16(bf)
	fmt.Println(t1, uint16(bf[0])|(uint16(bf[1])<<8), (uint16(bf[1])<<8)|uint16(bf[0]))
}
