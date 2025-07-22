package xmlparser

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

// Parse reads XML data and constructs an in-memory Document tree.
// This mimics the behavior of libxml2's parsing functions.
func Parse(xmlBytes []byte) (*Document, error) {
	decoder := xml.NewDecoder(bytes.NewReader(xmlBytes))
	doc := &Document{}
	var currentNode *Node

	for {
		token, err := decoder.Token()
		if err != nil {
			// io.EOF is the normal end of the document
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("error reading token: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Create a new node for this element
			node := &Node{
				Parent: currentNode,
				Name:   t.Name,
				Attrs:  t.Copy().Attr,
			}
			// If this is the first element, it's the root
			if doc.Root == nil {
				doc.Root = node
			}
			// If we are inside another node, add this as a child
			if currentNode != nil {
				currentNode.Children = append(currentNode.Children, node)
			}
			// Descend into the new node
			currentNode = node

		case xml.CharData:
			// Add text content to the current node
			if currentNode != nil {
				currentNode.Content += string(t)
			}

		case xml.EndElement:
			// Ascend back to the parent node
			if currentNode != nil {
				currentNode = currentNode.Parent
			}
		}
	}

	if doc.Root == nil {
		return nil, fmt.Errorf("xml document is empty or invalid")
	}

	return doc, nil
}
