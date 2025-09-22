package xmlparser

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test helper functions

func loadTestFile(t *testing.T, filename string) []byte {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read test file %s: %v", filename, err)
	}
	return content
}

func expectValidationError(t *testing.T, err error, expectedSubstring string) {
	if err == nil {
		t.Errorf("Expected validation to fail with error containing '%s', but validation passed", expectedSubstring)
		return
	}
	if !strings.Contains(err.Error(), expectedSubstring) {
		t.Errorf("Expected error to contain '%s', but got: %v", expectedSubstring, err)
	}
}

// Basic validation tests

func TestBasicValidation(t *testing.T) {
	xsdBytes := []byte(`
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
    <xs:element name="user">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="id" type="xs:integer" minOccurs="1" />
                <xs:element name="email">
                    <xs:simpleType>
                        <xs:restriction base="xs:string">
                            <xs:pattern value="[^@]+@[^@]+\.[^@]+" />
                        </xs:restriction>
                    </xs:simpleType>
                </xs:element>
                <xs:element name="status" minOccurs="0">
                     <xs:simpleType>
                        <xs:restriction base="xs:string">
                            <xs:enumeration value="active"/>
                            <xs:enumeration value="inactive"/>
                        </xs:restriction>
                    </xs:simpleType>
                </xs:element>
            </xs:sequence>
        </xs:complexType>
    </xs:element>
</xs:schema>`)

	schema, err := ParseXSD(xsdBytes)
	if err != nil {
		t.Fatalf("Failed to parse XSD schema: %v", err)
	}

	tests := []struct {
		name        string
		xml         string
		shouldPass  bool
		errorString string
	}{
		{
			name:       "Valid user with all fields",
			xml:        `<user><id>123</id><email>test@example.com</email><status>active</status></user>`,
			shouldPass: true,
		},
		{
			name:       "Valid user without optional status",
			xml:        `<user><id>456</id><email>another@example.com</email></user>`,
			shouldPass: true,
		},
		{
			name:        "Missing required ID",
			xml:         `<user><email>test@example.com</email></user>`,
			shouldPass:  false,
			errorString: "requires at least 1 <id> child",
		},
		{
			name:        "Invalid email pattern",
			xml:         `<user><id>789</id><email>not-a-valid-email</email></user>`,
			shouldPass:  false,
			errorString: "does not match pattern",
		},
		{
			name:        "Invalid enum value",
			xml:         `<user><id>101</id><email>good@email.com</email><status>pending</status></user>`,
			shouldPass:  false,
			errorString: "is not in the list of allowed values",
		},
		{
			name:        "Wrong root element",
			xml:         `<customer><id>112</id></customer>`,
			shouldPass:  false,
			errorString: "root element <customer> is not defined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Parse([]byte(tt.xml))
			if err != nil {
				t.Fatalf("Failed to parse XML: %v", err)
			}

			validationErr := schema.Validate(doc)
			if tt.shouldPass {
				if validationErr != nil {
					t.Errorf("Expected validation to pass, but got error: %v", validationErr)
				}
			} else {
				expectValidationError(t, validationErr, tt.errorString)
			}
		})
	}
}

// Test complex file-based validation

func TestFileBasedValidation(t *testing.T) {
	// Test files must exist in examples directory
	testCases := []struct {
		name           string
		xsdFile        string
		xmlFile        string
		shouldPass     bool
		skipIfMissing  bool
	}{
		{
			name:          "Simple library schema validation",
			xsdFile:       filepath.Join("examples", "simple_test.xsd"),
			xmlFile:       filepath.Join("examples", "simple_test.xml"),
			shouldPass:    true,
			skipIfMissing: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip if files don't exist and skipIfMissing is true
			if tc.skipIfMissing {
				if _, err := os.Stat(tc.xsdFile); os.IsNotExist(err) {
					t.Skip("Test files not found, skipping")
					return
				}
			}

			xsdBytes := loadTestFile(t, tc.xsdFile)
			xmlBytes := loadTestFile(t, tc.xmlFile)

			schema, err := ParseXSD(xsdBytes)
			if err != nil {
				t.Fatalf("Failed to parse XSD: %v", err)
			}

			doc, err := Parse(xmlBytes)
			if err != nil {
				t.Fatalf("Failed to parse XML: %v", err)
			}

			validationErr := schema.Validate(doc)
			if tc.shouldPass {
				if validationErr != nil {
					t.Errorf("Expected validation to pass, but got error: %v", validationErr)
				} else {
					t.Log("✓ File-based validation passed")
				}
			} else {
				if validationErr == nil {
					t.Error("Expected validation to fail, but it passed")
				}
			}
		})
	}
}

// Test error conditions

func TestErrorConditions(t *testing.T) {
	t.Run("Invalid XSD", func(t *testing.T) {
		invalidXSD := []byte(`<not-valid-xsd>`)
		_, err := ParseXSD(invalidXSD)
		if err == nil {
			t.Error("Expected error when parsing invalid XSD")
		}
	})

	t.Run("Invalid XML", func(t *testing.T) {
		invalidXML := []byte(`<unclosed-tag>`)
		_, err := Parse(invalidXML)
		if err == nil {
			t.Error("Expected error when parsing invalid XML")
		}
	})

	t.Run("Empty document validation", func(t *testing.T) {
		schema := &Schema{ElementMap: make(map[string]*Element)}
		err := schema.Validate(nil)
		if err == nil {
			t.Error("Expected error when validating nil document")
		}
	})
}