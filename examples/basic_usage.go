// Example usage of the xmlparser package
package main

import (
	"fmt"
	"log"

	"github.com/moolekkari/validatexml-go"
)

func main() {
	// Example XSD schema
	xsdData := []byte(`
<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
    <xs:element name="person">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="name">
                    <xs:simpleType>
                        <xs:restriction base="xs:string">
                            <xs:minLength value="1"/>
                            <xs:maxLength value="50"/>
                        </xs:restriction>
                    </xs:simpleType>
                </xs:element>
                <xs:element name="age">
                    <xs:simpleType>
                        <xs:restriction base="xs:integer">
                            <xs:minInclusive value="0"/>
                            <xs:maxInclusive value="120"/>
                        </xs:restriction>
                    </xs:simpleType>
                </xs:element>
                <xs:element name="email">
                    <xs:simpleType>
                        <xs:restriction base="xs:string">
                            <xs:pattern value="[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}"/>
                        </xs:restriction>
                    </xs:simpleType>
                </xs:element>
                <xs:element name="status" minOccurs="0">
                    <xs:simpleType>
                        <xs:restriction base="xs:string">
                            <xs:enumeration value="active"/>
                            <xs:enumeration value="inactive"/>
                            <xs:enumeration value="pending"/>
                        </xs:restriction>
                    </xs:simpleType>
                </xs:element>
            </xs:sequence>
        </xs:complexType>
    </xs:element>
</xs:schema>`)

	// Parse the XSD schema
	schema, err := xmlparser.ParseXSD(xsdData)
	if err != nil {
		log.Fatalf("Failed to parse XSD schema: %v", err)
	}

	// Valid XML document
	validXML := []byte(`
<person>
    <name>John Doe</name>
    <age>30</age>
    <email>john.doe@example.com</email>
    <status>active</status>
</person>`)

	// Parse the XML document
	document, err := xmlparser.Parse(validXML)
	if err != nil {
		log.Fatalf("Failed to parse XML document: %v", err)
	}

	// Validate the document against the schema
	if err := schema.Validate(document); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("✓ Document is valid!")
	}

	// Invalid XML document (age exceeds maximum)
	invalidXML := []byte(`
<person>
    <name>Jane Smith</name>
    <age>150</age>
    <email>jane@example.com</email>
</person>`)

	// Parse and validate invalid document
	invalidDocument, err := xmlparser.Parse(invalidXML)
	if err != nil {
		log.Fatalf("Failed to parse invalid XML: %v", err)
	}

	if err := schema.Validate(invalidDocument); err != nil {
		fmt.Printf("✓ Validation correctly failed: %v\n", err)
	} else {
		fmt.Println("❌ Validation unexpectedly passed")
	}
}