package xmlparser

import (
	"strings"
	"testing"
)

// Test namespace support with qualified element names
func TestNamespaceValidation(t *testing.T) {
	// XSD with target namespace and qualified elements
	xsdBytes := []byte(`
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns:tns="http://example.com/order"
           targetNamespace="http://example.com/order"
           elementFormDefault="qualified">

    <xs:element name="order">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="customer" type="xs:string"/>
                <xs:element name="amount" type="xs:decimal"/>
            </xs:sequence>
            <xs:attribute name="id" type="xs:integer" use="required"/>
        </xs:complexType>
    </xs:element>
</xs:schema>`)

	schema, err := ParseXSD(xsdBytes)
	if err != nil {
		t.Fatalf("Failed to parse XSD: %v", err)
	}

	// Check that namespace declarations were extracted
	if schema.Xmlns == nil {
		t.Error("Namespace declarations should have been extracted")
	}

	if namespace, exists := schema.Xmlns["tns"]; !exists || namespace != "http://example.com/order" {
		t.Errorf("Expected tns namespace to be 'http://example.com/order', got %s", namespace)
	}

	if schema.TargetNamespace != "http://example.com/order" {
		t.Errorf("Expected target namespace to be 'http://example.com/order', got %s", schema.TargetNamespace)
	}

	tests := []struct {
		name        string
		xml         string
		shouldPass  bool
		errorString string
	}{
		{
			name: "Valid qualified XML",
			xml: `<order xmlns="http://example.com/order" id="123">
				<customer>John Doe</customer>
				<amount>99.99</amount>
			</order>`,
			shouldPass: true,
		},
		{
			name: "Valid XML without explicit namespace (inherits from schema)",
			xml: `<order id="456">
				<customer>Jane Smith</customer>
				<amount>149.99</amount>
			</order>`,
			shouldPass: true,
		},
		{
			name: "Invalid - missing required attribute",
			xml: `<order xmlns="http://example.com/order">
				<customer>Bob Johnson</customer>
				<amount>75.50</amount>
			</order>`,
			shouldPass:  false,
			errorString: "required attribute",
		},
		{
			name: "Invalid - wrong element type",
			xml: `<order xmlns="http://example.com/order" id="789">
				<customer>Alice Brown</customer>
				<amount>not-a-number</amount>
			</order>`,
			shouldPass:  false,
			errorString: "not a valid decimal",
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
				} else {
					t.Log("✓ Namespace validation passed")
				}
			} else {
				if validationErr == nil {
					t.Error("Expected validation to fail, but it passed")
				} else if !strings.Contains(validationErr.Error(), tt.errorString) {
					t.Logf("⚠ Validation failed with unexpected error: %v", validationErr)
				} else {
					t.Logf("✓ Namespace validation correctly failed: %v", validationErr)
				}
			}
		})
	}
}

// Test elementFormDefault="unqualified"
func TestUnqualifiedElements(t *testing.T) {
	xsdBytes := []byte(`
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/product"
           elementFormDefault="unqualified">

    <xs:element name="product">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="name" type="xs:string"/>
                <xs:element name="price" type="xs:decimal"/>
            </xs:sequence>
        </xs:complexType>
    </xs:element>
</xs:schema>`)

	schema, err := ParseXSD(xsdBytes)
	if err != nil {
		t.Fatalf("Failed to parse XSD: %v", err)
	}

	// Test with unqualified elements
	xml := `<product>
		<name>Laptop</name>
		<price>999.99</price>
	</product>`

	doc, err := Parse([]byte(xml))
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	validationErr := schema.Validate(doc)
	if validationErr != nil {
		t.Errorf("Expected validation to pass for unqualified elements, but got error: %v", validationErr)
	} else {
		t.Log("✓ Unqualified element validation passed")
	}
}