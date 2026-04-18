package streaming

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
)

// AWS Event Stream binary format parser
// Format: [prelude][prelude_crc][headers][payload][message_crc]
// Prelude: 4 bytes total length + 4 bytes headers length
// Each section ends with CRC32 checksum

// EventStreamMessage represents a parsed event stream message
type EventStreamMessage struct {
	Headers map[string]interface{}
	Payload []byte
}

// parseMessage parses a single event stream message
// This function is used by the AWS SDK-compliant incremental parser in parser.go
func parseMessage(reader io.Reader) (*EventStreamMessage, error) {
	// Read prelude (8 bytes: 4 bytes total length + 4 bytes headers length)
	prelude := make([]byte, 8)
	n, err := io.ReadFull(reader, prelude)
	if err != nil {
		return nil, err
	}
	if n != 8 {
		return nil, fmt.Errorf("incomplete prelude: got %d bytes, expected 8", n)
	}
	
	totalLength := binary.BigEndian.Uint32(prelude[0:4])
	headersLength := binary.BigEndian.Uint32(prelude[4:8])
	
	// Read prelude CRC (4 bytes)
	preludeCRC := make([]byte, 4)
	if _, err := io.ReadFull(reader, preludeCRC); err != nil {
		return nil, fmt.Errorf("failed to read prelude CRC: %w", err)
	}
	
	// Verify prelude CRC
	expectedCRC := crc32.ChecksumIEEE(prelude)
	actualCRC := binary.BigEndian.Uint32(preludeCRC)
	if expectedCRC != actualCRC {
		return nil, fmt.Errorf("prelude CRC mismatch: expected %d, got %d", expectedCRC, actualCRC)
	}
	
	// Calculate payload length
	// Total = prelude(8) + prelude_crc(4) + headers + payload + message_crc(4)
	payloadLength := totalLength - 8 - 4 - headersLength - 4
	
	// Read headers
	headersData := make([]byte, headersLength)
	if headersLength > 0 {
		if _, err := io.ReadFull(reader, headersData); err != nil {
			return nil, fmt.Errorf("failed to read headers: %w", err)
		}
	}
	
	// Read payload
	payload := make([]byte, payloadLength)
	if payloadLength > 0 {
		if _, err := io.ReadFull(reader, payload); err != nil {
			return nil, fmt.Errorf("failed to read payload: %w", err)
		}
	}
	
	// Read message CRC (4 bytes)
	messageCRC := make([]byte, 4)
	if _, err := io.ReadFull(reader, messageCRC); err != nil {
		return nil, fmt.Errorf("failed to read message CRC: %w", err)
	}
	
	// Verify message CRC (entire message except the CRC itself)
	messageData := append(prelude, preludeCRC...)
	messageData = append(messageData, headersData...)
	messageData = append(messageData, payload...)
	expectedMsgCRC := crc32.ChecksumIEEE(messageData)
	actualMsgCRC := binary.BigEndian.Uint32(messageCRC)
	if expectedMsgCRC != actualMsgCRC {
		return nil, fmt.Errorf("message CRC mismatch: expected %d, got %d", expectedMsgCRC, actualMsgCRC)
	}
	
	// Parse headers
	headers, err := parseHeaders(headersData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse headers: %w", err)
	}
	
	return &EventStreamMessage{
		Headers: headers,
		Payload: payload,
	}, nil
}

// parseHeaders parses event stream headers
func parseHeaders(data []byte) (map[string]interface{}, error) {
	headers := make(map[string]interface{})
	reader := bytes.NewReader(data)
	
	for reader.Len() > 0 {
		// Read header name length (1 byte)
		nameLen, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		
		// Read header name
		name := make([]byte, nameLen)
		if _, err := io.ReadFull(reader, name); err != nil {
			return nil, err
		}
		
		// Read header value type (1 byte)
		valueType, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		
		// Read header value based on type
		var value interface{}
		switch valueType {
		case 0: // true
			value = true
		case 1: // false
			value = false
		case 2: // byte
			b, err := reader.ReadByte()
			if err != nil {
				return nil, err
			}
			value = b
		case 3: // short (2 bytes)
			var v int16
			if err := binary.Read(reader, binary.BigEndian, &v); err != nil {
				return nil, err
			}
			value = v
		case 4: // integer (4 bytes)
			var v int32
			if err := binary.Read(reader, binary.BigEndian, &v); err != nil {
				return nil, err
			}
			value = v
		case 5: // long (8 bytes)
			var v int64
			if err := binary.Read(reader, binary.BigEndian, &v); err != nil {
				return nil, err
			}
			value = v
		case 6: // byte array
			var length uint16
			if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
				return nil, err
			}
			data := make([]byte, length)
			if _, err := io.ReadFull(reader, data); err != nil {
				return nil, err
			}
			value = data
		case 7: // string
			var length uint16
			if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
				return nil, err
			}
			data := make([]byte, length)
			if _, err := io.ReadFull(reader, data); err != nil {
				return nil, err
			}
			value = string(data)
		case 8: // timestamp (8 bytes)
			var v int64
			if err := binary.Read(reader, binary.BigEndian, &v); err != nil {
				return nil, err
			}
			value = v
		case 9: // uuid (16 bytes)
			data := make([]byte, 16)
			if _, err := io.ReadFull(reader, data); err != nil {
				return nil, err
			}
			value = data
		default:
			return nil, fmt.Errorf("unknown header value type: %d", valueType)
		}
		
		headers[string(name)] = value
	}
	
	return headers, nil
}


