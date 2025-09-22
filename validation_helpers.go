package xmlparser

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// validatePattern checks if content matches the given regex pattern.
func validatePattern(content, pattern string) error {
	matched, err := regexp.MatchString(pattern, content)
	if err != nil {
		return fmt.Errorf("invalid pattern in schema: %s", pattern)
	}
	if !matched {
		return fmt.Errorf("value '%s' does not match pattern '%s'", content, pattern)
	}
	return nil
}

// validateEnumeration checks if content is in the allowed enumeration values.
func validateEnumeration(content string, enumerations []*Facet) error {
	allowedValues := make([]string, len(enumerations))
	for i, enum := range enumerations {
		allowedValues[i] = enum.Value
		if content == enum.Value {
			return nil
		}
	}
	return fmt.Errorf("value '%s' is not in the list of allowed values: [%s]",
		content, strings.Join(allowedValues, ", "))
}

// validateLengthConstraints checks minLength and maxLength constraints.
func validateLengthConstraints(content string, restriction *Restriction) []string {
	var errors []string

	if restriction.MinLength != nil && restriction.MinLength.Value != "" {
		if minLen, err := strconv.Atoi(restriction.MinLength.Value); err != nil {
			errors = append(errors, fmt.Sprintf("invalid minLength value in schema: %s", restriction.MinLength.Value))
		} else if len(content) < minLen {
			errors = append(errors, fmt.Sprintf("value '%s' is too short (minimum length: %d, actual: %d)",
				content, minLen, len(content)))
		}
	}

	if restriction.MaxLength != nil && restriction.MaxLength.Value != "" {
		if maxLen, err := strconv.Atoi(restriction.MaxLength.Value); err != nil {
			errors = append(errors, fmt.Sprintf("invalid maxLength value in schema: %s", restriction.MaxLength.Value))
		} else if len(content) > maxLen {
			errors = append(errors, fmt.Sprintf("value '%s' is too long (maximum length: %d, actual: %d)",
				content, maxLen, len(content)))
		}
	}

	return errors
}

// validateNumericConstraints checks minInclusive and maxInclusive constraints.
func validateNumericConstraints(content string, restriction *Restriction) []string {
	var errors []string

	if restriction.MinInclusive != nil && restriction.MinInclusive.Value != "" {
		if err := validateNumericRange(content, restriction.MinInclusive.Value, true, true, restriction.Base); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if restriction.MaxInclusive != nil && restriction.MaxInclusive.Value != "" {
		if err := validateNumericRange(content, restriction.MaxInclusive.Value, false, true, restriction.Base); err != nil {
			errors = append(errors, err.Error())
		}
	}

	return errors
}

// validateNumericRange validates that a numeric value is within the specified range.
func validateNumericRange(content, limitValue string, isMin, inclusive bool, baseType string) error {
	contentNum, limitNum, err := parseNumericValues(content, limitValue, baseType)
	if err != nil {
		return err
	}

	violatesRange := false
	if isMin {
		violatesRange = (inclusive && contentNum < limitNum) || (!inclusive && contentNum <= limitNum)
	} else {
		violatesRange = (inclusive && contentNum > limitNum) || (!inclusive && contentNum >= limitNum)
	}

	if violatesRange {
		direction := map[bool]string{true: "below minimum", false: "exceeds maximum"}[isMin]
		return fmt.Errorf("value '%s' %s allowed value %s", content, direction, limitValue)
	}

	return nil
}

// parseNumericValues parses content and limit values based on the base type.
func parseNumericValues(content, limitValue, baseType string) (contentNum, limitNum float64, err error) {
	content = strings.TrimSpace(content)

	switch baseType {
	case "xs:integer", "xs:int", "xs:long", "xs:short", "xs:byte":
		contentInt, err1 := strconv.ParseInt(content, 10, 64)
		limitInt, err2 := strconv.ParseInt(limitValue, 10, 64)
		if err1 != nil {
			return 0, 0, fmt.Errorf("value '%s' is not a valid integer", content)
		}
		if err2 != nil {
			return 0, 0, fmt.Errorf("invalid limit value in schema: %s", limitValue)
		}
		return float64(contentInt), float64(limitInt), nil

	case "xs:decimal", "xs:double", "xs:float":
		contentNum, err1 := strconv.ParseFloat(content, 64)
		limitNum, err2 := strconv.ParseFloat(limitValue, 64)
		if err1 != nil {
			return 0, 0, fmt.Errorf("value '%s' is not a valid decimal number", content)
		}
		if err2 != nil {
			return 0, 0, fmt.Errorf("invalid limit value in schema: %s", limitValue)
		}
		return contentNum, limitNum, nil

	default:
		// Try to parse as number for unknown types
		contentNum, err1 := strconv.ParseFloat(content, 64)
		limitNum, err2 := strconv.ParseFloat(limitValue, 64)
		if err1 != nil {
			return 0, 0, nil // Skip numeric validation for non-numeric content
		}
		if err2 != nil {
			return 0, 0, fmt.Errorf("invalid limit value in schema: %s", limitValue)
		}
		return contentNum, limitNum, nil
	}
}

// validateBuiltInType validates content against XML Schema built-in types.
func validateBuiltInType(content, typeName string) error {
	content = strings.TrimSpace(content)

	switch typeName {
	// Integer types
	case "xs:integer":
		if _, err := strconv.ParseInt(content, 10, 64); err != nil {
			return fmt.Errorf("value '%s' is not a valid integer", content)
		}

	case "xs:int":
		if val, err := strconv.ParseInt(content, 10, 32); err != nil {
			return fmt.Errorf("value '%s' is not a valid int", content)
		} else if val > 2147483647 || val < -2147483648 {
			return fmt.Errorf("value '%s' is out of range for int", content)
		}

	case "xs:long":
		if _, err := strconv.ParseInt(content, 10, 64); err != nil {
			return fmt.Errorf("value '%s' is not a valid long", content)
		}

	case "xs:short":
		if val, err := strconv.ParseInt(content, 10, 16); err != nil {
			return fmt.Errorf("value '%s' is not a valid short", content)
		} else if val > 32767 || val < -32768 {
			return fmt.Errorf("value '%s' is out of range for short", content)
		}

	case "xs:byte":
		if val, err := strconv.ParseInt(content, 10, 8); err != nil {
			return fmt.Errorf("value '%s' is not a valid byte", content)
		} else if val > 127 || val < -128 {
			return fmt.Errorf("value '%s' is out of range for byte", content)
		}

	case "xs:nonNegativeInteger":
		if val, err := strconv.ParseInt(content, 10, 64); err != nil {
			return fmt.Errorf("value '%s' is not a valid nonNegativeInteger", content)
		} else if val < 0 {
			return fmt.Errorf("value '%s' must be non-negative", content)
		}

	case "xs:positiveInteger":
		if val, err := strconv.ParseInt(content, 10, 64); err != nil {
			return fmt.Errorf("value '%s' is not a valid positiveInteger", content)
		} else if val <= 0 {
			return fmt.Errorf("value '%s' must be positive", content)
		}

	case "xs:unsignedInt":
		if val, err := strconv.ParseUint(content, 10, 32); err != nil {
			return fmt.Errorf("value '%s' is not a valid unsignedInt", content)
		} else if val > 4294967295 {
			return fmt.Errorf("value '%s' is out of range for unsignedInt", content)
		}

	// Decimal types
	case "xs:decimal":
		if _, err := strconv.ParseFloat(content, 64); err != nil {
			return fmt.Errorf("value '%s' is not a valid decimal", content)
		}

	case "xs:double":
		if _, err := strconv.ParseFloat(content, 64); err != nil {
			return fmt.Errorf("value '%s' is not a valid double", content)
		}

	case "xs:float":
		if _, err := strconv.ParseFloat(content, 32); err != nil {
			return fmt.Errorf("value '%s' is not a valid float", content)
		}

	// Boolean type
	case "xs:boolean":
		if content != "true" && content != "false" && content != "1" && content != "0" {
			return fmt.Errorf("value '%s' is not a valid boolean (expected: true, false, 1, or 0)", content)
		}

	// Date and time types
	case "xs:date":
		if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, content); !matched {
			return fmt.Errorf("value '%s' is not a valid date (expected format: YYYY-MM-DD)", content)
		}

	case "xs:dateTime":
		if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?$`, content); !matched {
			return fmt.Errorf("value '%s' is not a valid dateTime (expected format: YYYY-MM-DDTHH:mm:ss)", content)
		}

	case "xs:time":
		if matched, _ := regexp.MatchString(`^\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?$`, content); !matched {
			return fmt.Errorf("value '%s' is not a valid time (expected format: HH:mm:ss)", content)
		}

	case "xs:gYear":
		if matched, _ := regexp.MatchString(`^\d{4}$`, content); !matched {
			return fmt.Errorf("value '%s' is not a valid gYear (expected format: YYYY)", content)
		}

	case "xs:gMonth":
		if matched, _ := regexp.MatchString(`^--\d{2}$`, content); !matched {
			return fmt.Errorf("value '%s' is not a valid gMonth (expected format: --MM)", content)
		}

	case "xs:gDay":
		if matched, _ := regexp.MatchString(`^---\d{2}$`, content); !matched {
			return fmt.Errorf("value '%s' is not a valid gDay (expected format: ---DD)", content)
		}

	// Duration type
	case "xs:duration":
		if matched, _ := regexp.MatchString(`^-?P(\d+Y)?(\d+M)?(\d+D)?(T(\d+H)?(\d+M)?(\d+(\.\d+)?S)?)?$`, content); !matched {
			return fmt.Errorf("value '%s' is not a valid duration (expected format: PnYnMnDTnHnMnS)", content)
		}

	// String types
	case "xs:string", "xs:normalizedString":
		// All strings are valid

	case "xs:token":
		// Token cannot have leading/trailing whitespace or consecutive spaces
		if strings.TrimSpace(content) != content || strings.Contains(content, "  ") {
			return fmt.Errorf("value '%s' is not a valid token (no leading/trailing/consecutive whitespace allowed)", content)
		}

	case "xs:Name":
		if matched, _ := regexp.MatchString(`^[a-zA-Z_:][\w\-\.]*$`, content); !matched {
			return fmt.Errorf("value '%s' is not a valid Name", content)
		}

	case "xs:NCName":
		if matched, _ := regexp.MatchString(`^[a-zA-Z_][\w\-\.]*$`, content); !matched {
			return fmt.Errorf("value '%s' is not a valid NCName (no colons allowed)", content)
		}

	case "xs:ID", "xs:IDREF":
		if matched, _ := regexp.MatchString(`^[a-zA-Z_][\w\-\.]*$`, content); !matched {
			return fmt.Errorf("value '%s' is not a valid %s", content, typeName)
		}

	// URI types
	case "xs:anyURI":
		// Basic URI validation (simplified)
		if content == "" {
			return fmt.Errorf("URI cannot be empty")
		}
		if strings.Contains(content, " ") {
			return fmt.Errorf("value '%s' is not a valid URI (contains spaces)", content)
		}

	// Base64 and hex
	case "xs:base64Binary":
		if matched, _ := regexp.MatchString(`^[A-Za-z0-9+/]*={0,2}$`, content); !matched {
			return fmt.Errorf("value '%s' is not valid base64Binary", content)
		}

	case "xs:hexBinary":
		if matched, _ := regexp.MatchString(`^[0-9A-Fa-f]*$`, content); !matched {
			return fmt.Errorf("value '%s' is not valid hexBinary", content)
		}

	default:
		// Unknown/user-defined types - skip validation
	}

	return nil
}

// validateSequenceOccurrences validates occurrence constraints for xs:sequence.
func (s *Schema) validateSequenceOccurrences(node *Node, sequence *Sequence, childCounts map[string]int) []string {
	var errors []string

	for _, element := range sequence.Elements {
		count := childCounts[element.Name]

		// Check minOccurs
		if element.MinOccurs != "" {
			if min, _ := strconv.Atoi(element.MinOccurs); count < min {
				errors = append(errors, fmt.Sprintf(
					"element <%s> requires at least %d <%s> child, but found %d",
					node.Name.Local, min, element.Name, count))
			}
		}

		// Check maxOccurs
		if element.MaxOccurs != "" && element.MaxOccurs != "unbounded" {
			if max, err := strconv.Atoi(element.MaxOccurs); err != nil {
				errors = append(errors, fmt.Sprintf(
					"invalid maxOccurs value in schema for element <%s>: %s",
					element.Name, element.MaxOccurs))
			} else if count > max {
				errors = append(errors, fmt.Sprintf(
					"element <%s> allows at most %d <%s> child, but found %d",
					node.Name.Local, max, element.Name, count))
			}
		}
	}

	return errors
}

// validateChoiceOccurrences validates occurrence constraints for xs:choice.
func (s *Schema) validateChoiceOccurrences(node *Node, choice *Choice, validChoices int) []string {
	var errors []string

	// Check minOccurs for choice
	minOccurs := 1 // Default minOccurs for choice is 1
	if choice.MinOccurs != "" {
		if min, err := strconv.Atoi(choice.MinOccurs); err == nil {
			minOccurs = min
		}
	}

	if validChoices < minOccurs {
		errors = append(errors, fmt.Sprintf(
			"element <%s> choice requires at least %d selections, but found %d",
			node.Name.Local, minOccurs, validChoices))
	}

	// Check maxOccurs for choice
	if choice.MaxOccurs != "" && choice.MaxOccurs != "unbounded" {
		if max, err := strconv.Atoi(choice.MaxOccurs); err != nil {
			errors = append(errors, fmt.Sprintf(
				"invalid maxOccurs value in choice for element <%s>: %s",
				node.Name.Local, choice.MaxOccurs))
		} else if validChoices > max {
			errors = append(errors, fmt.Sprintf(
				"element <%s> choice allows at most %d selections, but found %d",
				node.Name.Local, max, validChoices))
		}
	}

	return errors
}

// findChoiceElement finds an element definition in an xs:choice.
func (s *Schema) findChoiceElement(childName xml.Name, choice *Choice) *Element {
	// Check direct elements
	for i := range choice.Elements {
		if s.elementsMatch(childName, choice.Elements[i].Name) {
			return &choice.Elements[i]
		}
	}

	// Check sequences within choice
	for _, sequence := range choice.Sequences {
		for i := range sequence.Elements {
			if s.elementsMatch(childName, sequence.Elements[i].Name) {
				return &sequence.Elements[i]
			}
		}
	}

	// Check nested choices
	for _, nestedChoice := range choice.Choices {
		if elem := s.findChoiceElement(childName, &nestedChoice); elem != nil {
			return elem
		}
	}

	return nil
}

// findAllElement finds an element definition in an xs:all group.
func (s *Schema) findAllElement(childName xml.Name, all *All) *Element {
	for i := range all.Elements {
		if s.elementsMatch(childName, all.Elements[i].Name) {
			return &all.Elements[i]
		}
	}
	return nil
}
