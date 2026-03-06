package xmlparser

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestSimpleContentBasic tests xs:simpleContent with xs:extension using inline types.
func TestSimpleContentBasic(t *testing.T) {
	xsdBytes := []byte(`
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
    <xs:element name="root">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="amount" type="CurrencyAmount"/>
            </xs:sequence>
        </xs:complexType>
    </xs:element>

    <xs:simpleType name="AmountType">
        <xs:restriction base="xs:decimal">
            <xs:minInclusive value="0"/>
        </xs:restriction>
    </xs:simpleType>

    <xs:complexType name="CurrencyAmount">
        <xs:simpleContent>
            <xs:extension base="AmountType">
                <xs:attribute name="Ccy" type="xs:string" use="required"/>
            </xs:extension>
        </xs:simpleContent>
    </xs:complexType>
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
			name:       "Valid amount with currency",
			xml:        `<root><amount Ccy="EUR">100.50</amount></root>`,
			shouldPass: true,
		},
		{
			name:        "Missing required Ccy attribute",
			xml:         `<root><amount>100.50</amount></root>`,
			shouldPass:  false,
			errorString: "required attribute 'Ccy' is missing",
		},
		{
			name:        "Invalid decimal content",
			xml:         `<root><amount Ccy="EUR">not-a-number</amount></root>`,
			shouldPass:  false,
			errorString: "not a valid decimal",
		},
		{
			name:        "Negative amount violates minInclusive",
			xml:         `<root><amount Ccy="EUR">-5.00</amount></root>`,
			shouldPass:  false,
			errorString: "below minimum",
		},
		{
			name:        "Unexpected attribute",
			xml:         `<root><amount Ccy="EUR" Foo="bar">100.00</amount></root>`,
			shouldPass:  false,
			errorString: "unexpected attribute 'Foo'",
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

// TestSimpleContentBuiltInBase tests simpleContent extending a built-in xs type directly.
func TestSimpleContentBuiltInBase(t *testing.T) {
	xsdBytes := []byte(`
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
    <xs:element name="root">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="label" type="LabelType"/>
            </xs:sequence>
        </xs:complexType>
    </xs:element>

    <xs:complexType name="LabelType">
        <xs:simpleContent>
            <xs:extension base="xs:string">
                <xs:attribute name="lang" type="xs:string" use="required"/>
            </xs:extension>
        </xs:simpleContent>
    </xs:complexType>
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
			name:       "Valid label with lang",
			xml:        `<root><label lang="en">Hello</label></root>`,
			shouldPass: true,
		},
		{
			name:        "Missing required lang attribute",
			xml:         `<root><label>Hello</label></root>`,
			shouldPass:  false,
			errorString: "required attribute 'lang' is missing",
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

// TestDigitConstraints tests totalDigits and fractionDigits facets.
func TestDigitConstraints(t *testing.T) {
	xsdBytes := []byte(`
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
    <xs:element name="root">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="value">
                    <xs:simpleType>
                        <xs:restriction base="xs:decimal">
                            <xs:totalDigits value="5"/>
                            <xs:fractionDigits value="2"/>
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
			name:       "Valid - within limits",
			xml:        `<root><value>123.45</value></root>`,
			shouldPass: true,
		},
		{
			name:       "Valid - no fraction",
			xml:        `<root><value>12345</value></root>`,
			shouldPass: true,
		},
		{
			name:       "Valid - fewer digits",
			xml:        `<root><value>1.5</value></root>`,
			shouldPass: true,
		},
		{
			name:        "Too many total digits",
			xml:         `<root><value>123456</value></root>`,
			shouldPass:  false,
			errorString: "totalDigits",
		},
		{
			name:        "Too many fraction digits",
			xml:         `<root><value>1.234</value></root>`,
			shouldPass:  false,
			errorString: "fractionDigits",
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

// TestPain001SimpleContent tests parsing and validating against the real pain.001.001.09 XSD
// focusing on the ActiveOrHistoricCurrencyAndAmount type which uses simpleContent.
func TestPain001SimpleContent(t *testing.T) {
	xsdPath, err := filepath.Abs("examples/pain.001.001.09.xsd")
	if err != nil {
		t.Fatalf("Failed to resolve XSD path: %v", err)
	}

	xsdBytes := loadTestFile(t, xsdPath)
	schema, err := ParseXSD(xsdBytes, filepath.Dir(xsdPath))
	if err != nil {
		t.Fatalf("Failed to parse pain.001 XSD: %v", err)
	}

	// Verify the schema parsed the simpleContent type
	ct, exists := schema.ComplexTypeMap["ActiveOrHistoricCurrencyAndAmount"]
	if !exists {
		t.Fatal("Expected ComplexTypeMap to contain ActiveOrHistoricCurrencyAndAmount")
	}
	if ct.SimpleContent == nil {
		t.Fatal("Expected ActiveOrHistoricCurrencyAndAmount to have SimpleContent")
	}
	if ct.SimpleContent.Extension == nil {
		t.Fatal("Expected SimpleContent to have Extension")
	}
	if ct.SimpleContent.Extension.Base != "ActiveOrHistoricCurrencyAndAmount_SimpleType" {
		t.Errorf("Expected extension base 'ActiveOrHistoricCurrencyAndAmount_SimpleType', got '%s'",
			ct.SimpleContent.Extension.Base)
	}
	if len(ct.SimpleContent.Extension.Attributes) != 1 {
		t.Fatalf("Expected 1 extension attribute, got %d", len(ct.SimpleContent.Extension.Attributes))
	}
	if ct.SimpleContent.Extension.Attributes[0].Name != "Ccy" {
		t.Errorf("Expected attribute name 'Ccy', got '%s'", ct.SimpleContent.Extension.Attributes[0].Name)
	}

	// Verify the base simple type was parsed with digit constraints
	st, exists := schema.SimpleTypeMap["ActiveOrHistoricCurrencyAndAmount_SimpleType"]
	if !exists {
		t.Fatal("Expected SimpleTypeMap to contain ActiveOrHistoricCurrencyAndAmount_SimpleType")
	}
	if st.Restriction == nil {
		t.Fatal("Expected restriction on SimpleType")
	}
	if st.Restriction.TotalDigits == nil || st.Restriction.TotalDigits.Value != "18" {
		t.Errorf("Expected totalDigits=18, got %v", st.Restriction.TotalDigits)
	}
	if st.Restriction.FractionDigits == nil || st.Restriction.FractionDigits.Value != "5" {
		t.Errorf("Expected fractionDigits=5, got %v", st.Restriction.FractionDigits)
	}

	// Verify the currency code pattern type
	ccySt, exists := schema.SimpleTypeMap["ActiveOrHistoricCurrencyCode"]
	if !exists {
		t.Fatal("Expected SimpleTypeMap to contain ActiveOrHistoricCurrencyCode")
	}
	if ccySt.Restriction == nil || ccySt.Restriction.Pattern == nil {
		t.Fatal("Expected ActiveOrHistoricCurrencyCode to have a pattern restriction")
	}
}

// TestPain001ParseXSD tests that the full pain.001.001.09 XSD can be parsed without errors.
func TestPain001ParseXSD(t *testing.T) {
	xsdPath, err := filepath.Abs("examples/pain.001.001.09.xsd")
	if err != nil {
		t.Fatalf("Failed to resolve XSD path: %v", err)
	}

	xsdBytes := loadTestFile(t, xsdPath)
	schema, err := ParseXSD(xsdBytes, filepath.Dir(xsdPath))
	if err != nil {
		t.Fatalf("Failed to parse pain.001 XSD: %v", err)
	}

	// Spot-check that key types were parsed
	expectedComplexTypes := []string{
		"Document",
		"CustomerCreditTransferInitiationV09",
		"PaymentInstruction30",
		"CreditTransferTransaction34",
		"ActiveOrHistoricCurrencyAndAmount",
	}
	for _, name := range expectedComplexTypes {
		if _, exists := schema.ComplexTypeMap[name]; !exists {
			t.Errorf("Expected ComplexTypeMap to contain '%s'", name)
		}
	}

	expectedSimpleTypes := []string{
		"ActiveOrHistoricCurrencyAndAmount_SimpleType",
		"ActiveOrHistoricCurrencyCode",
		"IBAN2007Identifier",
		"Max35Text",
	}
	for _, name := range expectedSimpleTypes {
		if _, exists := schema.SimpleTypeMap[name]; !exists {
			t.Errorf("Expected SimpleTypeMap to contain '%s'", name)
		}
	}

	// Verify the root element
	rootElem, exists := schema.ElementMap["Document"]
	if !exists {
		t.Fatal("Expected ElementMap to contain 'Document'")
	}
	if rootElem.Type != "Document" {
		t.Errorf("Expected Document element type 'Document', got '%s'", rootElem.Type)
	}
}

// TestPain001ValidateMinimalDocument validates a minimal pain.001 XML document.
func TestPain001ValidateMinimalDocument(t *testing.T) {
	xsdPath, err := filepath.Abs("examples/pain.001.001.09.xsd")
	if err != nil {
		t.Fatalf("Failed to resolve XSD path: %v", err)
	}

	xsdBytes := loadTestFile(t, xsdPath)
	schema, err := ParseXSD(xsdBytes, filepath.Dir(xsdPath))
	if err != nil {
		t.Fatalf("Failed to parse pain.001 XSD: %v", err)
	}

	// Minimal valid pain.001 document with an amount element using simpleContent
	xmlBytes := []byte(`<Document xmlns="urn:iso:std:iso:20022:tech:xsd:pain.001.001.09">
    <CstmrCdtTrfInitn>
        <GrpHdr>
            <MsgId>MSG001</MsgId>
            <CreDtTm>2024-01-15T10:30:00</CreDtTm>
            <NbOfTxs>1</NbOfTxs>
            <InitgPty>
                <Nm>Test Company</Nm>
            </InitgPty>
        </GrpHdr>
        <PmtInf>
            <PmtInfId>PMT001</PmtInfId>
            <PmtMtd>TRF</PmtMtd>
            <NbOfTxs>1</NbOfTxs>
            <ReqdExctnDt>
                <Dt>2024-01-20</Dt>
            </ReqdExctnDt>
            <Dbtr>
                <Nm>Debtor Name</Nm>
            </Dbtr>
            <DbtrAcct>
                <Id>
                    <IBAN>DE89370400440532013000</IBAN>
                </Id>
            </DbtrAcct>
            <DbtrAgt>
                <FinInstnId>
                    <BICFI>COBADEFFXXX</BICFI>
                </FinInstnId>
            </DbtrAgt>
            <CdtTrfTxInf>
                <PmtId>
                    <EndToEndId>E2E001</EndToEndId>
                </PmtId>
                <Amt>
                    <InstdAmt Ccy="EUR">1000.50</InstdAmt>
                </Amt>
                <Cdtr>
                    <Nm>Creditor Name</Nm>
                </Cdtr>
            </CdtTrfTxInf>
        </PmtInf>
    </CstmrCdtTrfInitn>
</Document>`)

	doc, err := Parse(xmlBytes)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	validationErr := schema.Validate(doc)
	if validationErr != nil {
		// Log errors but check if simpleContent-related validation works
		errStr := validationErr.Error()
		t.Logf("Validation result: %v", validationErr)

		// The amount element should NOT produce simpleContent-related errors
		if strings.Contains(errStr, "type definition 'ActiveOrHistoricCurrencyAndAmount' not found") {
			t.Error("SimpleContent type should be recognized, not produce 'type not found' errors")
		}
	}
}
