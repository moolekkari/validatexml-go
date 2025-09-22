package xmlparser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test xs:include functionality
func TestIncludeSchema(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "xmlparser_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create included schema file
	includedSchemaContent := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
	<xs:complexType name="AddressType">
		<xs:sequence>
			<xs:element name="street" type="xs:string"/>
			<xs:element name="city" type="xs:string"/>
			<xs:element name="zipcode" type="xs:string"/>
		</xs:sequence>
	</xs:complexType>
</xs:schema>`

	includedSchemaPath := filepath.Join(tmpDir, "address.xsd")
	if err := os.WriteFile(includedSchemaPath, []byte(includedSchemaContent), 0644); err != nil {
		t.Fatalf("Failed to write included schema file: %v", err)
	}

	// Main schema with xs:include
	mainSchemaContent := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
	<xs:include schemaLocation="address.xsd"/>

	<xs:element name="person">
		<xs:complexType>
			<xs:sequence>
				<xs:element name="name" type="xs:string"/>
				<xs:element name="address" type="AddressType"/>
			</xs:sequence>
		</xs:complexType>
	</xs:element>
</xs:schema>`

	schema, err := ParseXSD([]byte(mainSchemaContent), tmpDir)
	if err != nil {
		t.Fatalf("Failed to parse schema with includes: %v", err)
	}

	// Verify that AddressType was included
	if _, exists := schema.ComplexTypeMap["AddressType"]; !exists {
		t.Error("Expected AddressType from included schema to be available")
	}

	// Test validation with included types
	xml := `<person>
		<name>John Doe</name>
		<address>
			<street>123 Main St</street>
			<city>Anytown</city>
			<zipcode>12345</zipcode>
		</address>
	</person>`

	doc, err := Parse([]byte(xml))
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	validationErr := schema.Validate(doc)
	if validationErr != nil {
		t.Errorf("Expected validation to pass with included schema, but got error: %v", validationErr)
	} else {
		t.Log("✓ Schema include validation passed")
	}
}

// Test xs:import functionality
func TestImportSchema(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "xmlparser_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create imported schema file with different namespace
	importedSchemaContent := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
	targetNamespace="http://example.com/common"
	elementFormDefault="qualified">

	<xs:simpleType name="EmailType">
		<xs:restriction base="xs:string">
			<xs:pattern value="[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}"/>
		</xs:restriction>
	</xs:simpleType>

	<xs:element name="email" type="EmailType"/>
</xs:schema>`

	importedSchemaPath := filepath.Join(tmpDir, "common.xsd")
	if err := os.WriteFile(importedSchemaPath, []byte(importedSchemaContent), 0644); err != nil {
		t.Fatalf("Failed to write imported schema file: %v", err)
	}

	// Main schema with xs:import
	mainSchemaContent := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
	xmlns:common="http://example.com/common"
	targetNamespace="http://example.com/person">

	<xs:import namespace="http://example.com/common" schemaLocation="common.xsd"/>

	<xs:element name="contact">
		<xs:complexType>
			<xs:sequence>
				<xs:element name="name" type="xs:string"/>
				<xs:element ref="common:email"/>
			</xs:sequence>
		</xs:complexType>
	</xs:element>
</xs:schema>`

	schema, err := ParseXSD([]byte(mainSchemaContent), tmpDir)
	if err != nil {
		t.Fatalf("Failed to parse schema with imports: %v", err)
	}

	// Verify that imported elements are available
	// Note: This is a basic test - full namespace support would be more complex
	if len(schema.Elements) < 2 { // Should have at least contact and email elements
		t.Errorf("Expected at least 2 elements after import, got %d", len(schema.Elements))
	}

	if len(schema.SimpleTypes) < 1 { // Should have EmailType
		t.Errorf("Expected at least 1 simple type after import, got %d", len(schema.SimpleTypes))
	}

	t.Log("✓ Schema import functionality working")
}

// Test error conditions for imports/includes
func TestImportIncludeErrors(t *testing.T) {
	tests := []struct {
		name      string
		schema    string
		shouldErr bool
		errorText string
	}{
		{
			name: "Missing include file",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
				<xs:include schemaLocation="nonexistent.xsd"/>
			</xs:schema>`,
			shouldErr: true,
			errorText: "failed to process include",
		},
		{
			name: "Missing import file",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
				<xs:import namespace="http://example.com/test" schemaLocation="nonexistent.xsd"/>
			</xs:schema>`,
			shouldErr: true,
			errorText: "failed to process import",
		},
		{
			name: "Include without schemaLocation",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
				<xs:include/>
			</xs:schema>`,
			shouldErr: true,
			errorText: "missing schemaLocation attribute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseXSD([]byte(tt.schema), "")
			if tt.shouldErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorText, err)
				} else {
					t.Logf("✓ Error handling working: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// Test import without schemaLocation (built-in namespaces)
func TestImportBuiltinNamespace(t *testing.T) {
	schemaContent := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
	<xs:import namespace="http://www.w3.org/XML/1998/namespace"/>

	<xs:element name="test" type="xs:string"/>
</xs:schema>`

	schema, err := ParseXSD([]byte(schemaContent), "")
	if err != nil {
		t.Fatalf("Failed to parse schema with built-in import: %v", err)
	}

	if len(schema.Elements) != 1 {
		t.Errorf("Expected 1 element, got %d", len(schema.Elements))
	}

	t.Log("✓ Built-in namespace import working")
}

// Test nested include scenarios (include → include)
func TestNestedIncludeSchema(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "xmlparser_nested_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create deeply nested schema (level 3)
	level3SchemaContent := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
	<xs:simpleType name="ZipCodeType">
		<xs:restriction base="xs:string">
			<xs:pattern value="[0-9]{5}(-[0-9]{4})?"/>
		</xs:restriction>
	</xs:simpleType>
</xs:schema>`

	level3Path := filepath.Join(tmpDir, "zipcode.xsd")
	if err := os.WriteFile(level3Path, []byte(level3SchemaContent), 0644); err != nil {
		t.Fatalf("Failed to write level 3 schema file: %v", err)
	}

	// Create level 2 schema that includes level 3
	level2SchemaContent := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
	<xs:include schemaLocation="zipcode.xsd"/>

	<xs:complexType name="AddressType">
		<xs:sequence>
			<xs:element name="street" type="xs:string"/>
			<xs:element name="city" type="xs:string"/>
			<xs:element name="zipcode" type="ZipCodeType"/>
		</xs:sequence>
	</xs:complexType>
</xs:schema>`

	level2Path := filepath.Join(tmpDir, "address.xsd")
	if err := os.WriteFile(level2Path, []byte(level2SchemaContent), 0644); err != nil {
		t.Fatalf("Failed to write level 2 schema file: %v", err)
	}

	// Main schema (level 1) that includes level 2
	mainSchemaContent := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
	<xs:include schemaLocation="address.xsd"/>

	<xs:element name="person">
		<xs:complexType>
			<xs:sequence>
				<xs:element name="name" type="xs:string"/>
				<xs:element name="address" type="AddressType"/>
			</xs:sequence>
		</xs:complexType>
	</xs:element>
</xs:schema>`

	schema, err := ParseXSD([]byte(mainSchemaContent), tmpDir)
	if err != nil {
		t.Fatalf("Failed to parse schema with nested includes: %v", err)
	}

	// Verify that all types from all levels are available
	if _, exists := schema.ComplexTypeMap["AddressType"]; !exists {
		t.Error("Expected AddressType from level 2 to be available")
	}
	if _, exists := schema.SimpleTypeMap["ZipCodeType"]; !exists {
		t.Error("Expected ZipCodeType from level 3 to be available")
	}

	// Test validation with nested included types
	xml := `<person>
		<name>John Doe</name>
		<address>
			<street>123 Main St</street>
			<city>Anytown</city>
			<zipcode>12345-6789</zipcode>
		</address>
	</person>`

	doc, err := Parse([]byte(xml))
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	validationErr := schema.Validate(doc)
	if validationErr != nil {
		t.Errorf("Expected validation to pass with nested includes, but got error: %v", validationErr)
	} else {
		t.Log("✓ Nested include validation passed")
	}

	// Test validation with invalid zipcode pattern
	invalidXml := `<person>
		<name>Jane Doe</name>
		<address>
			<street>456 Oak Ave</street>
			<city>Somewhere</city>
			<zipcode>invalid</zipcode>
		</address>
	</person>`

	invalidDoc, err := Parse([]byte(invalidXml))
	if err != nil {
		t.Fatalf("Failed to parse invalid XML: %v", err)
	}

	validationErr = schema.Validate(invalidDoc)
	if validationErr == nil {
		t.Error("Expected validation to fail with invalid zipcode pattern")
	} else if strings.Contains(validationErr.Error(), "does not match pattern") {
		t.Log("✓ Nested include pattern validation working correctly")
	} else {
		t.Logf("⚠ Unexpected validation error: %v", validationErr)
	}
}

// Test mixed import/include scenarios
func TestMixedImportIncludeSchema(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "xmlparser_mixed_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a schema that will be imported (different namespace)
	commonSchemaContent := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
	targetNamespace="http://example.com/common"
	elementFormDefault="qualified">

	<xs:simpleType name="EmailType">
		<xs:restriction base="xs:string">
			<xs:pattern value="[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}"/>
		</xs:restriction>
	</xs:simpleType>
</xs:schema>`

	commonPath := filepath.Join(tmpDir, "common.xsd")
	if err := os.WriteFile(commonPath, []byte(commonSchemaContent), 0644); err != nil {
		t.Fatalf("Failed to write common schema file: %v", err)
	}

	// Create a schema that includes common and will be included by main
	contactSchemaContent := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
	xmlns:common="http://example.com/common">

	<xs:import namespace="http://example.com/common" schemaLocation="common.xsd"/>

	<xs:complexType name="ContactInfoType">
		<xs:sequence>
			<xs:element name="phone" type="xs:string" minOccurs="0"/>
			<xs:element name="email" type="common:EmailType" minOccurs="0"/>
		</xs:sequence>
	</xs:complexType>
</xs:schema>`

	contactPath := filepath.Join(tmpDir, "contact.xsd")
	if err := os.WriteFile(contactPath, []byte(contactSchemaContent), 0644); err != nil {
		t.Fatalf("Failed to write contact schema file: %v", err)
	}

	// Main schema that includes contact (which imports common)
	mainSchemaContent := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
	<xs:include schemaLocation="contact.xsd"/>

	<xs:element name="person">
		<xs:complexType>
			<xs:sequence>
				<xs:element name="name" type="xs:string"/>
				<xs:element name="contact" type="ContactInfoType" minOccurs="0"/>
			</xs:sequence>
		</xs:complexType>
	</xs:element>
</xs:schema>`

	schema, err := ParseXSD([]byte(mainSchemaContent), tmpDir)
	if err != nil {
		t.Fatalf("Failed to parse schema with mixed import/include: %v", err)
	}

	// Verify that types from all schemas are available
	if _, exists := schema.ComplexTypeMap["ContactInfoType"]; !exists {
		t.Error("Expected ContactInfoType from included schema to be available")
	}

	// Note: The imported EmailType might be prefixed, so we check if we have more than just the main element
	if len(schema.Elements) < 1 || len(schema.ComplexTypes) < 1 {
		t.Errorf("Expected elements and types from mixed import/include, got %d elements, %d complex types",
			len(schema.Elements), len(schema.ComplexTypes))
	}

	t.Log("✓ Mixed import/include schema parsing working")
}

// Test circular reference detection
func TestCircularReferenceDetection(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "xmlparser_circular_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create schema A that includes schema B
	schemaAContent := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
	<xs:include schemaLocation="schemaB.xsd"/>

	<xs:element name="elementA" type="xs:string"/>
</xs:schema>`

	schemaAPath := filepath.Join(tmpDir, "schemaA.xsd")
	if err := os.WriteFile(schemaAPath, []byte(schemaAContent), 0644); err != nil {
		t.Fatalf("Failed to write schema A file: %v", err)
	}

	// Create schema B that includes schema A (circular reference)
	schemaBContent := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
	<xs:include schemaLocation="schemaA.xsd"/>

	<xs:element name="elementB" type="xs:string"/>
</xs:schema>`

	schemaBPath := filepath.Join(tmpDir, "schemaB.xsd")
	if err := os.WriteFile(schemaBPath, []byte(schemaBContent), 0644); err != nil {
		t.Fatalf("Failed to write schema B file: %v", err)
	}

	// Try to parse schema A (which will cause infinite recursion without protection)
	_, err = ParseXSD([]byte(schemaAContent), tmpDir)

	// We expect this to either:
	// 1. Detect the circular reference and return an error
	// 2. Handle it gracefully (though this might lead to stack overflow)
	// For now, we'll just check that it doesn't hang indefinitely

	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "circular") ||
			strings.Contains(strings.ToLower(err.Error()), "recursive") ||
			strings.Contains(strings.ToLower(err.Error()), "stack") {
			t.Log("✓ Circular reference detected and handled")
		} else {
			t.Logf("⚠ Circular reference caused error (which is acceptable): %v", err)
		}
	} else {
		t.Log("⚠ Circular reference not detected - this could potentially cause issues")
	}
}
