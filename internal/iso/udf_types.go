package iso

import (
	"encoding/binary"
	"time"
	"unicode/utf16"

	"github.com/blazium-games/blazium-toolchain/internal/settings"
)

func encodeDString(s string, fieldLen int) []byte {
	buf := make([]byte, fieldLen)
	if s == "" {
		return buf
	}
	eightBit := true
	for _, r := range s {
		if r > 0xFF {
			eightBit = false
			break
		}
	}
	var used int
	if eightBit {
		buf[0] = 8
		used = 1
		for _, r := range s {
			if used >= fieldLen-1 {
				break
			}
			buf[used] = byte(r)
			used++
		}
	} else {
		buf[0] = 16
		used = 1
		for _, u := range utf16.Encode([]rune(s)) {
			if used+2 > fieldLen-1 {
				break
			}
			binary.BigEndian.PutUint16(buf[used:], u)
			used += 2
		}
	}
	buf[fieldLen-1] = byte(used)
	return buf
}

func encodeTimestamp(t time.Time) []byte {
	t = t.UTC()
	b := make([]byte, 12)
	binary.LittleEndian.PutUint16(b[0:], 1<<12)
	binary.LittleEndian.PutUint16(b[2:], uint16(t.Year()))
	b[4] = byte(t.Month())
	b[5] = byte(t.Day())
	b[6] = byte(t.Hour())
	b[7] = byte(t.Minute())
	b[8] = byte(t.Second())
	return b
}

func charSpec() []byte {
	b := make([]byte, 64)
	copy(b[1:], "OSTA Compressed Unicode")
	return b
}

func entityID(id string, suffix []byte) []byte {
	b := make([]byte, 32)
	copy(b[1:24], id)
	copy(b[24:32], suffix)
	return b
}

func domainEntityID() []byte {
	suffix := make([]byte, 8)
	binary.LittleEndian.PutUint16(suffix[0:], 0x0102)
	return entityID("*OSTA UDF Compliant", suffix)
}

func implEntityID(provider string) []byte {
	if provider == "" {
		provider = settings.Current().InterDVD.Provider
	}
	if len(provider) > 23 {
		provider = provider[:23]
	}
	return entityID("*"+provider, make([]byte, 8))
}

func utf16Encode(s string) []uint16 { return utf16.Encode([]rune(s)) }

func utf16Count(s string) int { return len(utf16.Encode([]rune(s))) }

func shortAD(length, location uint32) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b[0:], length)
	binary.LittleEndian.PutUint32(b[4:], location)
	return b
}

func longAD(length uint32, location uint32, partition uint16) []byte {
	b := make([]byte, 16)
	binary.LittleEndian.PutUint32(b[0:], length)
	binary.LittleEndian.PutUint32(b[4:], location)
	binary.LittleEndian.PutUint16(b[8:], partition)
	return b
}

func extentAD(length, location uint32) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b[0:], length)
	binary.LittleEndian.PutUint32(b[4:], location)
	return b
}

func volStructDesc(id string) []byte {
	b := make([]byte, SectorSize)
	copy(b[1:6], id)
	b[6] = 1
	return b
}
