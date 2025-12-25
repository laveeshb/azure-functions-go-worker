// Package bindings provides converters between protobuf TypedData and Go types.
package bindings

import (
	"encoding/json"
	"fmt"

	pb "github.com/laveeshb/azure-functions-go-worker/internal/rpc/proto"
)

// TypedDataToString converts TypedData to a string.
func TypedDataToString(td *pb.TypedData) string {
	if td == nil {
		return ""
	}

	switch data := td.Data.(type) {
	case *pb.TypedData_String_:
		return data.String_
	case *pb.TypedData_Json:
		return data.Json
	case *pb.TypedData_Bytes:
		return string(data.Bytes)
	case *pb.TypedData_Stream:
		return string(data.Stream)
	case *pb.TypedData_Int:
		return fmt.Sprintf("%d", data.Int)
	case *pb.TypedData_Double:
		return fmt.Sprintf("%f", data.Double)
	default:
		return ""
	}
}

// TypedDataToBytes converts TypedData to bytes.
func TypedDataToBytes(td *pb.TypedData) []byte {
	if td == nil {
		return nil
	}

	switch data := td.Data.(type) {
	case *pb.TypedData_String_:
		return []byte(data.String_)
	case *pb.TypedData_Json:
		return []byte(data.Json)
	case *pb.TypedData_Bytes:
		return data.Bytes
	case *pb.TypedData_Stream:
		return data.Stream
	default:
		return nil
	}
}

// StringToTypedData converts a string to TypedData.
func StringToTypedData(s string) *pb.TypedData {
	return &pb.TypedData{
		Data: &pb.TypedData_String_{
			String_: s,
		},
	}
}

// BytesToTypedData converts bytes to TypedData.
func BytesToTypedData(b []byte) *pb.TypedData {
	return &pb.TypedData{
		Data: &pb.TypedData_Bytes{
			Bytes: b,
		},
	}
}

// JSONToTypedData converts a value to JSON TypedData.
func JSONToTypedData(v interface{}) (*pb.TypedData, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	return &pb.TypedData{
		Data: &pb.TypedData_Json{
			Json: string(data),
		},
	}, nil
}

// TypedDataToJSON unmarshals TypedData JSON into a value.
func TypedDataToJSON(td *pb.TypedData, v interface{}) error {
	if td == nil {
		return nil
	}

	var jsonStr string
	switch data := td.Data.(type) {
	case *pb.TypedData_Json:
		jsonStr = data.Json
	case *pb.TypedData_String_:
		jsonStr = data.String_
	default:
		return fmt.Errorf("TypedData does not contain JSON data")
	}

	return json.Unmarshal([]byte(jsonStr), v)
}
