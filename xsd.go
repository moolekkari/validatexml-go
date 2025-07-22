package xmlparser

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

// Parse reads XSD data and returns a new Schema object ready for validation.
func ParseXSD(xsdBytes []byte) (*Schema, error) {
	var internalSchema = new(Schema)
	decoder := xml.NewDecoder(bytes.NewReader(xsdBytes))

	if err := decoder.Decode(internalSchema); err != nil {
		return nil, fmt.Errorf("error decoding XSD: %w", err)
	}

	// Prepare the internal maps for fast lookups
	if err := internalSchema.prepare(); err != nil {
		return nil, fmt.Errorf("error preparing schema: %w", err)
	}

	return internalSchema, nil
}

// prepare builds internal maps for quick access to definitions by name.
// Without this, finding a rule would mean searching through slices every time, which is slow.
func (s *Schema) prepare() error {
	s.ElementMap = make(map[string]*Element)
	s.ComplexTypeMap = make(map[string]*ComplexType)
	s.SimpleTypeMap = make(map[string]*SimpleType)

	for i := range s.Elements {
		el := &s.Elements[i]
		if el.Name == "" {
			return fmt.Errorf("top-level element missing name")
		}
		s.ElementMap[el.Name] = el
	}

	for i := range s.ComplexTypes {
		ct := &s.ComplexTypes[i]
		if ct.Name == "" {
			return fmt.Errorf("top-level complexType missing name")
		}
		s.ComplexTypeMap[ct.Name] = ct
	}

	for i := range s.SimpleTypes {
		st := &s.SimpleTypes[i]
		if st.Name == "" {
			return fmt.Errorf("top-level simpleType missing name")
		}
		s.SimpleTypeMap[st.Name] = st
	}

	return nil
}
