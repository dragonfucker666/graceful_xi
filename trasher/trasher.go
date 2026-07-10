package trasher

import (
	"math"
	"math/rand"
	"io"
	"encoding/binary"
)

type bufType [math.MaxUint16]byte

func readChunk(in io.Reader, buf []byte) (int, error) {
	chunkSize := uint16(0)
	err := binary.Read(in, binary.BigEndian, &chunkSize)
	if err != nil {
		return 0, err
	}
	n, err := io.ReadFull(in, buf)
	return n, err
}

func Clean(in io.Reader, out io.Writer) {
	buf := bufType{}
	for {
		n, err := readChunk(in, buf[:])
		if err != nil {
			return
		}
		if _, err = out.Write(buf[:n]); err != nil {
			return
		}
		if _, err = readChunk(in, buf[:]); err != nil {
			return
		}
	}
}

func writeChunk(out io.Writer, buf []byte) error {
	chunkSize := uint16(len(buf))
	err := binary.Write(out, binary.BigEndian, &chunkSize)
	if err != nil {
		return err
	}
	n, err := out.Write(buf)
	_ = n
	return err
}

var minTrashRatio float64 = 1/6
var targetTrashRatio float64 = 1/3
var maxTrashRatio float64 = 1/1

func Dirty(in io.Reader, out io.Writer) {
	buf := bufType{}
	for {
		n, err := in.Read(buf[:])
		if n == 0 {
			return
		}
		err = writeChunk(out, buf[:n])
		if err != nil {
			return
		}
		trashRatio := minTrashRatio + math.Pow(rand.Float64(), 1/(targetTrashRatio - minTrashRatio) - 1) * (maxTrashRatio - minTrashRatio)
		trashSize := uint(trashRatio * float64(n))
		err = writeChunk(out, buf[:trashSize])
		if err != nil {
			return
		}
	}
}
