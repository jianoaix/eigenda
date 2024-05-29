package encoding

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"fmt"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/golang/snappy"
)

// EncodeVarint encodes an unsigned integer using varint encoding
func EncodeVarint(value uint64) ([]byte, error) {
	var buffer bytes.Buffer
	for {
		if value < 128 {
			buffer.WriteByte(byte(value))
			return buffer.Bytes(), nil
		}
		buffer.WriteByte(byte(value&0x7F | 0x80)) // Set MSB for continuation
		value >>= 7
	}
}

// PackUint64s encodes a slice of uint64 into a byte buffer using packed encoding.
func PackUint64s(data []uint64) ([]byte, error) {
	var buffer bytes.Buffer

	// Encode the number of elements (length prefix) using varint encoding
	// encodedLength, err := EncodeVarint(uint64(len(data)))
	// if err != nil {
	// 	return nil, err
	// }
	// buffer.Write(encodedLength)

	// Loop through each uint64 and encode it directly (no separators)
	for _, value := range data {
		encodedValue, err := EncodeVarint(value)
		if err != nil {
			return nil, err
		}
		buffer.Write(encodedValue)
	}

	return buffer.Bytes(), nil
}

func compressWithGzip(data []byte) ([]byte, error) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer) // Create Gzip writer targeting the buffer

	defer writer.Close() // Ensure proper closing

	_, err := writer.Write(data) // Write data to the compressed stream
	if err != nil {
		return nil, err
	}

	err = writer.Flush() // Flush remaining data from the writer
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil // Return the compressed data as a byte slice
}

func (c *Frame) Serialize() ([]byte, error) {
	res, err := encode(c)
	if err == nil {
		packSize := 0
		for _, coeff := range c.Coeffs {
			bs, err := PackUint64s(coeff[:])
			if err != nil {
				fmt.Println("xdeb FAILED to pack")
				return nil, err
			}
			packSize += len(bs)
		}
		compressedSnappy := snappy.Encode(nil, res)
		compressedZip, cerr := compressWithGzip(res)
		if cerr != nil {
			return nil, cerr
		}
		fmt.Println("xdeb frame serialization, ori size:", c.Size(), " serialized size:", len(res), "packSize:", packSize, "compressed gob with snappy: ", len(compressedSnappy), "compressed gob with gzip: ", len(compressedZip), "num coeffs:", c.Length())
	}
	return res, err
}

func (c *Frame) Deserialize(data []byte) (*Frame, error) {
	err := decode(data, c)
	if !c.Proof.IsInSubGroup() {
		return nil, fmt.Errorf("proof is in not the subgroup")
	}

	return c, err
}

func (f *Frame) Encode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(f)
	if err != nil {
		return nil, err
	}
	res := buf.Bytes()
	compressedData := snappy.Encode(nil, res)
	fmt.Println("xdeb chunk serialization, ori size:", f.Size(), " serialized size:", len(res), "compressed gob", len(compressedData), "num coeffs:", f.Length())
	return res, nil
}

func Decode(b []byte) (Frame, error) {
	var f Frame
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&f)
	if err != nil {
		return Frame{}, err
	}
	return f, nil
}

func (c *G1Commitment) Serialize() ([]byte, error) {
	res := (*bn254.G1Affine)(c).Bytes()
	return res[:], nil
}

func (c *G1Commitment) Deserialize(data []byte) (*G1Commitment, error) {
	_, err := (*bn254.G1Affine)(c).SetBytes(data)
	if err != nil {
		return nil, err
	}
	return c, err
}

func (c *G1Commitment) UnmarshalJSON(data []byte) error {
	var g1Point bn254.G1Affine
	err := json.Unmarshal(data, &g1Point)
	if err != nil {
		return err
	}
	c.X = g1Point.X
	c.Y = g1Point.Y

	if !(*bn254.G1Affine)(c).IsInSubGroup() {
		return fmt.Errorf("G1Commitment not in the subgroup")
	}

	return nil
}

func (c *G2Commitment) Serialize() ([]byte, error) {
	res := (*bn254.G2Affine)(c).Bytes()
	return res[:], nil
}

func (c *G2Commitment) Deserialize(data []byte) (*G2Commitment, error) {
	_, err := (*bn254.G2Affine)(c).SetBytes(data)
	if err != nil {
		return nil, err
	}

	return c, err
}

func (c *G2Commitment) UnmarshalJSON(data []byte) error {
	var g2Point bn254.G2Affine
	err := json.Unmarshal(data, &g2Point)
	if err != nil {
		return err
	}
	c.X = g2Point.X
	c.Y = g2Point.Y

	if !(*bn254.G2Affine)(c).IsInSubGroup() {
		return fmt.Errorf("G2Commitment not in the subgroup")
	}
	return nil
}

func encode(obj any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(obj)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decode(data []byte, obj any) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(obj)
	if err != nil {
		return err
	}
	return nil
}
