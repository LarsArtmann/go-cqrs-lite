package main

import (
	"fmt"
	"strings"
)

// writeConstructor: the library constructor's error already names the wire
// name and carries its error family, so it is returned unwrapped.
func writeConstructor(b *strings.Builder, m *marker) {
	if !m.constructor {
		return
	}

	fmt.Fprintf(b, "// %s stamps the %s envelope onto the payload; options reach the library constructor.\n",
		constructorName(m), m.wire)

	if m.kind == kindCommand {
		fmt.Fprintf(b, "func %s(\n\tstreamID id.StreamID,\n\tpayload %s,\n\topts ...command.Option,\n) (%s, error) {\n",
			constructorName(m), m.structName, m.structName)
		fmt.Fprintf(b, "\tbasic, err := command.New(%s, streamID, opts...)\n", m.constName)
		fmt.Fprintf(b, "\tif err != nil {\n\t\treturn %s{}, err\n\t}\n\n", m.structName)
		b.WriteString("\tpayload.BasicCommand = basic\n\n\treturn payload, nil\n}\n\n")

		return
	}

	fmt.Fprintf(b, "func %s(\n\tpayload %s,\n\topts ...query.Option,\n) (%s, error) {\n",
		constructorName(m), m.structName, m.structName)
	fmt.Fprintf(b, "\tbasic, err := query.New(%s, opts...)\n", m.constName)
	fmt.Fprintf(b, "\tif err != nil {\n\t\treturn %s{}, err\n\t}\n\n", m.structName)
	b.WriteString("\tpayload.BasicQuery = basic\n\n\treturn payload, nil\n}\n\n")
}
