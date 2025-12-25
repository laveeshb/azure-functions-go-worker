package bindings

import (
	"testing"

	pb "github.com/laveeshb/azure-functions-go-worker/internal/rpc/proto"
)

func TestTypedDataToString(t *testing.T) {
	tests := []struct {
		name     string
		input    *pb.TypedData
		expected string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name: "string data",
			input: &pb.TypedData{
				Data: &pb.TypedData_String_{String_: "hello"},
			},
			expected: "hello",
		},
		{
			name: "json data",
			input: &pb.TypedData{
				Data: &pb.TypedData_Json{Json: `{"key":"value"}`},
			},
			expected: `{"key":"value"}`,
		},
		{
			name: "bytes data",
			input: &pb.TypedData{
				Data: &pb.TypedData_Bytes{Bytes: []byte("byte string")},
			},
			expected: "byte string",
		},
		{
			name: "int data",
			input: &pb.TypedData{
				Data: &pb.TypedData_Int{Int: 42},
			},
			expected: "42",
		},
		{
			name: "double data",
			input: &pb.TypedData{
				Data: &pb.TypedData_Double{Double: 3.14},
			},
			expected: "3.140000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TypedDataToString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestTypedDataToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    *pb.TypedData
		expected []byte
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "string data",
			input: &pb.TypedData{
				Data: &pb.TypedData_String_{String_: "hello"},
			},
			expected: []byte("hello"),
		},
		{
			name: "bytes data",
			input: &pb.TypedData{
				Data: &pb.TypedData_Bytes{Bytes: []byte{0x01, 0x02, 0x03}},
			},
			expected: []byte{0x01, 0x02, 0x03},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TypedDataToBytes(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}
			if string(result) != string(tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestStringToTypedData(t *testing.T) {
	result := StringToTypedData("test string")
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	str, ok := result.Data.(*pb.TypedData_String_)
	if !ok {
		t.Fatal("Expected string data type")
	}
	if str.String_ != "test string" {
		t.Errorf("Expected 'test string', got '%s'", str.String_)
	}
}

func TestBytesToTypedData(t *testing.T) {
	input := []byte{0x01, 0x02, 0x03}
	result := BytesToTypedData(input)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	bytes, ok := result.Data.(*pb.TypedData_Bytes)
	if !ok {
		t.Fatal("Expected bytes data type")
	}
	if string(bytes.Bytes) != string(input) {
		t.Errorf("Expected %v, got %v", input, bytes.Bytes)
	}
}

func TestJSONToTypedData(t *testing.T) {
	input := map[string]interface{}{
		"name": "test",
		"value": 123,
	}
	result, err := JSONToTypedData(input)
	if err != nil {
		t.Fatalf("JSONToTypedData failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	json, ok := result.Data.(*pb.TypedData_Json)
	if !ok {
		t.Fatal("Expected JSON data type")
	}
	if json.Json == "" {
		t.Error("Expected non-empty JSON string")
	}
}

func TestTypedDataToJSON(t *testing.T) {
	input := &pb.TypedData{
		Data: &pb.TypedData_Json{Json: `{"name":"test","value":123}`},
	}

	var result map[string]interface{}
	err := TypedDataToJSON(input, &result)
	if err != nil {
		t.Fatalf("TypedDataToJSON failed: %v", err)
	}
	if result["name"] != "test" {
		t.Errorf("Expected name 'test', got '%v'", result["name"])
	}
	// JSON numbers decode as float64
	if result["value"].(float64) != 123 {
		t.Errorf("Expected value 123, got %v", result["value"])
	}
}
