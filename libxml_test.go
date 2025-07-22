package xmlparser

import (
	"strings"
	"testing"
)

// TestValidation covers the end-to-end process:
// 1. Parsing an XSD schema.
// 2. Parsing an XML document.
// 3. Validating the document against the schema.
func TestValidation(t *testing.T) {
	// --- 1. ARRANGE: Define the schema and XML documents for our tests ---

	// A simple XSD schema defining a 'user' element.
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
</xs:schema>
`)

	// --- Test Cases ---
	testCases := []struct {
		name          string
		xmlBytes      []byte
		shouldBeValid bool
		expectedError string // A substring of the expected error message
	}{
		{
			name:          "Valid XML",
			xmlBytes:      []byte(`<user><id>123</id><email>test@example.com</email><status>active</status></user>`),
			shouldBeValid: true,
		},
		{
			name:          "Valid XML without optional element",
			xmlBytes:      []byte(`<user><id>456</id><email>another@example.com</email></user>`),
			shouldBeValid: true,
		},
		{
			name:          "Invalid XML - Missing required element",
			xmlBytes:      []byte(`<user><email>test@example.com</email></user>`),
			shouldBeValid: false,
			expectedError: "requires at least 1 <id> child",
		},
		{
			name:          "Invalid XML - Pattern mismatch",
			xmlBytes:      []byte(`<user><id>789</id><email>not-a-valid-email</email></user>`),
			shouldBeValid: false,
			expectedError: "does not match pattern",
		},
		{
			name:          "Invalid XML - Enum mismatch",
			xmlBytes:      []byte(`<user><id>101</id><email>good@email.com</email><status>pending</status></user>`),
			shouldBeValid: false,
			expectedError: "is not in the list of allowed values",
		},
		{
			name:          "Invalid XML - Wrong root element",
			xmlBytes:      []byte(`<customer><id>112</id></customer>`),
			shouldBeValid: false,
			expectedError: "root element <customer> is not defined",
		},
	}

	// --- 2. ACT: Parse the schema once for all tests ---
	schema, err := ParseXSD(xsdBytes)
	if err != nil {
		// If schema parsing fails, none of the tests can run.
		t.Fatalf("Failed to parse XSD schema: %v", err)
	}

	// --- 3. ASSERT: Run each test case as a sub-test ---
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the XML document for this specific test case
			doc, err := Parse(tc.xmlBytes)
			if err != nil {
				// If XML parsing fails, we can't proceed with validation.
				t.Fatalf("Failed to parse XML: %v", err)
			}

			// Validate the document against the schema
			validationErr := schema.Validate(doc)

			// Check if the outcome is what we expected
			if tc.shouldBeValid {
				if validationErr != nil {
					t.Errorf("Expected XML to be valid, but got error: %v", validationErr)
				}
			} else { // We expect an error
				if validationErr == nil {
					t.Errorf("Expected XML to be invalid, but no error was returned")
				} else if !strings.Contains(validationErr.Error(), tc.expectedError) {
					// This is an optional but very useful check to ensure we got the *right* error.
					t.Errorf("Expected error to contain '%s', but got: %v", tc.expectedError, validationErr)
				}
			}
		})
	}
}
