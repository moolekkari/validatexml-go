package xmlparser

import (
	"strings"
	"testing"
)

// Test individual validation features

func TestPatternValidation2(t *testing.T) {
	xsdBytes := []byte(`
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
    <xs:element name="test">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="isbn">
                    <xs:simpleType>
                        <xs:restriction base="xs:string">
                            <xs:pattern value="^(97[89]\d{10}|\d{9}[\dX])$" />
                        </xs:restriction>
                    </xs:simpleType>
                </xs:element>
                <xs:element name="email">
                    <xs:simpleType>
                        <xs:restriction base="xs:string">
                            <xs:pattern value="[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}" />
                        </xs:restriction>
                    </xs:simpleType>
                </xs:element>
            </xs:sequence>
        </xs:complexType>
    </xs:element>
</xs:schema>`)

	schema, err := ParseXSD(xsdBytes)
	if err != nil {
		t.Fatalf("Failed to parse XSD: %v", err)
	}

	tests := []struct {
		name        string
		xml         string
		shouldPass  bool
		errorString string
	}{
		{
			name:       "Valid ISBN and email",
			xml:        `<test><isbn>9780743273565</isbn><email>test@example.com</email></test>`,
			shouldPass: true,
		},
		{
			name:        "Invalid ISBN pattern",
			xml:         `<test><isbn>12345</isbn><email>valid@email.com</email></test>`,
			shouldPass:  false,
			errorString: "does not match pattern",
		},
		{
			name:        "Invalid email pattern",
			xml:         `<test><isbn>9780743273565</isbn><email>invalid-email</email></test>`,
			shouldPass:  false,
			errorString: "does not match pattern",
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

func TestLengthConstraints2(t *testing.T) {
	xsdBytes := []byte(`
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
    <xs:element name="test">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="name">
                    <xs:simpleType>
                        <xs:restriction base="xs:string">
                            <xs:minLength value="3"/>
                            <xs:maxLength value="10"/>
                        </xs:restriction>
                    </xs:simpleType>
                </xs:element>
            </xs:sequence>
        </xs:complexType>
    </xs:element>
</xs:schema>`)

	schema, err := ParseXSD(xsdBytes)
	if err != nil {
		t.Fatalf("Failed to parse XSD: %v", err)
	}

	tests := []struct {
		name       string
		xml        string
		shouldPass bool
		errorCheck func(error) bool
	}{
		{
			name:       "Valid length",
			xml:        `<test><name>Alice</name></test>`,
			shouldPass: true,
		},
		{
			name:       "Too short",
			xml:        `<test><name>Al</name></test>`,
			shouldPass: false,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "too short")
			},
		},
		{
			name:       "Too long",
			xml:        `<test><name>VeryLongNameThatExceedsLimit</name></test>`,
			shouldPass: false,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "too long")
			},
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
				if validationErr == nil {
					t.Error("Expected validation to fail, but it passed")
				} else if tt.errorCheck != nil && !tt.errorCheck(validationErr) {
					t.Errorf("Error check failed for: %v", validationErr)
				} else {
					t.Logf("✓ Length constraint validation working: %v", validationErr)
				}
			}
		})
	}
}

func TestNumericRangeConstraints2(t *testing.T) {
	xsdBytes := []byte(`
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
    <xs:element name="test">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="rating">
                    <xs:simpleType>
                        <xs:restriction base="xs:integer">
                            <xs:minInclusive value="1"/>
                            <xs:maxInclusive value="5"/>
                        </xs:restriction>
                    </xs:simpleType>
                </xs:element>
            </xs:sequence>
        </xs:complexType>
    </xs:element>
</xs:schema>`)

	schema, err := ParseXSD(xsdBytes)
	if err != nil {
		t.Fatalf("Failed to parse XSD: %v", err)
	}

	tests := []struct {
		name       string
		xml        string
		shouldPass bool
		errorCheck func(error) bool
	}{
		{
			name:       "Valid range",
			xml:        `<test><rating>3</rating></test>`,
			shouldPass: true,
		},
		{
			name:       "Below minimum",
			xml:        `<test><rating>0</rating></test>`,
			shouldPass: false,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "below minimum")
			},
		},
		{
			name:       "Above maximum",
			xml:        `<test><rating>6</rating></test>`,
			shouldPass: false,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "exceeds maximum")
			},
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
				if validationErr == nil {
					t.Error("Expected validation to fail, but it passed")
				} else if tt.errorCheck != nil && !tt.errorCheck(validationErr) {
					t.Errorf("Error check failed for: %v", validationErr)
				} else {
					t.Logf("✓ Numeric range validation working: %v", validationErr)
				}
			}
		})
	}
}

func TestMaxOccursValidation(t *testing.T) {
	xsdBytes := []byte(`
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
    <xs:element name="test">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="item" type="xs:string" maxOccurs="2"/>
            </xs:sequence>
        </xs:complexType>
    </xs:element>
</xs:schema>`)

	schema, err := ParseXSD(xsdBytes)
	if err != nil {
		t.Fatalf("Failed to parse XSD: %v", err)
	}

	tests := []struct {
		name       string
		xml        string
		shouldPass bool
		errorCheck func(error) bool
	}{
		{
			name:       "Within limit",
			xml:        `<test><item>first</item><item>second</item></test>`,
			shouldPass: true,
		},
		{
			name:       "Exceeds limit",
			xml:        `<test><item>first</item><item>second</item><item>third</item></test>`,
			shouldPass: false,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "allows at most")
			},
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
				if validationErr == nil {
					t.Error("Expected validation to fail, but it passed")
				} else if tt.errorCheck != nil && !tt.errorCheck(validationErr) {
					t.Errorf("Error check failed for: %v", validationErr)
				} else {
					t.Logf("✓ MaxOccurs validation working: %v", validationErr)
				}
			}
		})
	}
}

func TestEmptyElement(t *testing.T) {
	xsdBytes := []byte(`
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
	<xs:element name="test">
		<xs:simpleType>
			<xs:restriction base="xs:string">
				<xs:minLength value="1"/>
			</xs:restriction>
		</xs:simpleType>
	</xs:element>
</xs:schema>`)

	schema, err := ParseXSD(xsdBytes)
	if err != nil {
		t.Fatalf("Failed to parse XSD: %v", err)
	}

	tests := []struct {
		name       string
		xml        string
		shouldPass bool
		errorCheck func(error) bool
	}{
		{
			name:       "Valid length",
			xml:        `<test>Test</test>`,
			shouldPass: true,
		},
		{
			name:       "Too short",
			xml:        `<test></test>`,
			shouldPass: false,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "too short")
			},
		},
		{
			name:       "Can't be empty",
			xml:        `<test/>`,
			shouldPass: false,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "too short")
			},
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
				if validationErr == nil {
					t.Error("Expected validation to fail, but it passed")
				} else if tt.errorCheck != nil && !tt.errorCheck(validationErr) {
					t.Errorf("Error check failed for: %v", validationErr)
				} else {
					t.Logf("✓ Length constraint validation working: %v", validationErr)
				}
			}
		})
	}
}
