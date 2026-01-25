import { visit } from "unist-util-visit";
import { Node } from "unist";

interface ParsedOptions {
    columns?: string;
    filters?: string;
}

// Parse the options string (e.g., "columns:title,status|filter:status=active")
function parseOptions(optionsStr: string): ParsedOptions {
    const result: ParsedOptions = {};

    // Split by pipe to get sections
    const sections = optionsStr.split('|');

    for (const section of sections) {
        const trimmed = section.trim();

        // Check for prefixed sections
        if (trimmed.startsWith('columns:')) {
            result.columns = trimmed.slice(8).trim();
        } else if (trimmed.startsWith('filter:')) {
            result.filters = trimmed.slice(7).trim();
        } else {
            // Backward compatibility: if no prefix, treat as columns
            result.columns = trimmed;
        }
    }

    return result;
}

export default function remarkSchemaTable() {
    return (tree: Node) => {
        visit(tree, "text", (node: any, index: number | undefined, parent: any) => {
            if (!parent || typeof node.value !== "string" || index === undefined) return;

            // Match both old and new schema table syntax:
            // Old: &SCHEMATABLE:schemaId& or &SCHEMATABLE:schemaId|col1,col2&
            // New: {{schema:slug}} or {{schema:slug|options}}
            // Options can be: cols:field1,field2 or filter:field1=value1,field2=value2
            const regex = /(?:&SCHEMATABLE:|{{schema:)([^&|}\s]+)(?:\|([^&}]+?))?(?:&|}})/g;
            let match;
            const newNodes: any[] = [];
            let lastIndex = 0;

            while ((match = regex.exec(node.value)) !== null) {
                const [fullMatch, schemaRef, optionsStr] = match;

                // Push text before match
                if (match.index > lastIndex) {
                    newNodes.push({
                        type: "text",
                        value: node.value.slice(lastIndex, match.index),
                    });
                }

                // Parse options
                const options = optionsStr ? parseOptions(optionsStr) : {};

                // Push schema table node with optional data attributes
                // schemaRef can be either an ID (old format) or a slug (new format)
                const hProperties: any = {
                    className: "schema-table-container",
                    "data-schema-ref": schemaRef
                };

                if (options.columns) {
                    hProperties["data-columns"] = options.columns;
                }

                if (options.filters) {
                    hProperties["data-filters"] = options.filters;
                }

                newNodes.push({
                    type: "schemaTable",
                    data: {
                        schemaRef,
                        columns: options.columns,
                        filters: options.filters,
                        hName: "div",
                        hProperties
                    },
                    children: [],
                });

                lastIndex = match.index + fullMatch.length;
            }

            // Push remaining text after last match
            if (lastIndex < node.value.length) {
                newNodes.push({
                    type: "text",
                    value: node.value.slice(lastIndex),
                });
            }

            if (newNodes.length > 0) {
                parent.children.splice(index, 1, ...newNodes);
                return index + newNodes.length;
            }
        });
    };
}
