
# Pure Go XML Schema (XSD) Validator

This is a pure Go library for parsing XML Schema Definition (XSD) files and validating XML documents against them. It is a functional port and enhancement of the Rust `xmlschema-rs` library.

## Features

*   **Pure Go**: No CGo or external dependencies are required.
*   **XSD Parsing**: Parses XSD files into an easy-to-use Go struct model using the standard `encoding/xml` library.
*   **XML Validation**: Validates XML documents against the parsed schema rules.
*   **Streaming Validator**: Low memory footprint, suitable for large XML files.
*   **Subset of XSD**: Implements the most common XSD features, including:
    *   `xs:element`, `xs:complexType`, `xs:simpleType`
    *   `xs:sequence`
    *   `xs:restriction` with facets like `minLength`, `maxLength`, `pattern`, and `enumeration`.

## How to Run the Example

1.  Make sure you have Go installed (version 1.18 or newer).
2.  Create the project directory and files as described above.
3.  Tidy the dependencies:
    ```sh
    go mod tidy
    ```

## How it Works (ELI5)

1.  **Parsing (Reading the Rules)**: The `schema.Parse()` function reads your `.xsd` file (the "instruction booklet") and builds a set of Go objects that represent all the rules. It creates an index of these rules so it can find them quickly.

2.  **Validation (Checking the Work)**: The `validator.Validate()` function reads your `.xml` file (the "LEGO car") piece by piece. For each piece, it looks up the corresponding rule in the parsed schema and checks if it's correct. It checks for things like:
    *   Is this piece allowed here?
    *   Are there too many or too few of this piece?
    *   Is the text inside the piece correct (e.g., is an email address actually an email)?

If it finds any mistakes, it returns a list of all the errors it found. If the list is empty, the XML is valid!
