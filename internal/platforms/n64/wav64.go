package n64

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

// writeWav64File converts a host RIFF WAVE (PCM) into uncompressed libdragon wav64
// (WV64 v10 format 0). This lives in the toolchain so Blazium never runs audioconv64.
func writeWav64File(wavPath, outPath string, loop bool) error {
	raw, err := os.ReadFile(wavPath)
	if err != nil {
		return err
	}
	pcm, rate, err := pcmFromWav(raw)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	out := make([]byte, 0, 32+len(pcm)*2)
	out = append(out, 'W', 'V', '6', '4')
	out = append(out, 10, 0, 1, 16) // version, format RAW, mono, 16-bit
	out = binary.BigEndian.AppendUint32(out, uint32(rate))
	out = binary.BigEndian.AppendUint32(out, uint32(len(pcm)))
	loopLen := uint32(0)
	if loop && len(pcm) > 0 {
		loopLen = uint32(len(pcm))
	}
	out = binary.BigEndian.AppendUint32(out, loopLen)
	out = binary.BigEndian.AppendUint32(out, 0)  // loop_end 0 ⇒ len
	out = binary.BigEndian.AppendUint32(out, 32) // samples start
	out = binary.BigEndian.AppendUint32(out, 0)  // state_size
	for _, s := range pcm {
		out = binary.BigEndian.AppendUint16(out, uint16(s))
	}
	return os.WriteFile(outPath, out, 0o644)
}

func pcmFromWav(raw []byte) ([]int16, int, error) {
	if len(raw) < 44 || string(raw[0:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		return nil, 0, fmt.Errorf("not RIFF WAVE")
	}
	off := 12
	rate := 22050
	bits := 16
	ch := 1
	var data []byte
	for off+8 <= len(raw) {
		id := string(raw[off : off+4])
		sz := int(binary.LittleEndian.Uint32(raw[off+4 : off+8]))
		body := off + 8
		if body+sz > len(raw) {
			sz = len(raw) - body
		}
		if id == "fmt " && sz >= 16 {
			if binary.LittleEndian.Uint16(raw[body:body+2]) != 1 {
				return nil, 0, fmt.Errorf("WAVE must be PCM")
			}
			ch = int(binary.LittleEndian.Uint16(raw[body+2 : body+4]))
			rate = int(binary.LittleEndian.Uint32(raw[body+4 : body+8]))
			bits = int(binary.LittleEndian.Uint16(raw[body+14 : body+16]))
		}
		if id == "data" {
			data = raw[body : body+sz]
			break
		}
		off = body + sz
		if sz&1 == 1 {
			off++
		}
	}
	if data == nil || rate < 4000 || ch < 1 {
		return nil, 0, fmt.Errorf("WAVE missing PCM data")
	}
	if bits == 8 {
		pcm := make([]int16, 0, len(data)/ch)
		for i := 0; i+ch <= len(data); i += ch {
			pcm = append(pcm, int16((int(data[i])-128)*256))
		}
		return pcm, rate, nil
	}
	step := 2 * ch
	pcm := make([]int16, 0, len(data)/step)
	for i := 0; i+1 < len(data); i += step {
		pcm = append(pcm, int16(binary.LittleEndian.Uint16(data[i:i+2])))
	}
	if len(pcm) == 0 {
		return nil, 0, fmt.Errorf("WAVE PCM empty")
	}
	return pcm, rate, nil
}
