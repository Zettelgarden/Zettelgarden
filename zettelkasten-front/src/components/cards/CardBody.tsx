import Markdown from "react-markdown";
import React, { useState, useEffect } from "react";
import { downloadFile } from "../../api/files";
import { Card, Entity } from "../../models/Card";
import remarkEntity from "../../remark-entity";
import remarkTaskQuery from "../../remark-task-query";
import remarkSchemaTable from "../../remark-schema-table";
import { useNavigate } from "react-router-dom";
import remarkGfm from "remark-gfm";

import { useShortcutContext } from "../../contexts/ShortcutContext";

import { CardLinkWithPreview } from "./CardLinkWithPreview";
import { DynamicTaskList } from "./DynamicTaskList";
import { DynamicSchemaTable } from "./DynamicSchemaTable";
import { H1, H2, H3, H4, H5, H6 } from "../Header";
import {
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableHeader,
  TableCell
} from "../table/TableComponents";
//import { fetchEntityByName } from "../../api/entities";
import { fetchEntityById } from "../../api/entities";

interface CustomImageRendererProps {
  src?: string; // Make src optional
  alt?: string; // Make alt optional
  title?: string; // Make title optional
}

interface CardBodyProps {
  viewingCard: Card;
  entities?: Entity[];
}

function preprocessTaskQueries(body: string): string {
  const regex = /\{\{tasks:\s*([^}]+)\}\}/g;
  return body.replace(regex, (match, query) => {
    const trimmedQuery = query.trim();
    return `&TASKQUERY:${trimmedQuery}&`;
  });
}

function preprocessCardLinks(body: string): string {
  // Only match IDs without parentheses after - this preserves standard markdown links
  return body.replace(/\[([A-Za-z0-9_.-/]+)\](?!\()/g, "[$1](#)");
}

function preprocessSchemaTables(body: string): string {
  // Match {{schema: <id>}} or {{schema_table: <id>}} syntax with optional columns and filter parameters
  // Supports: {{schema:1}}, {{schema:1 columns:title,status}}, {{schema:1 filter:status=In Progress}}
  // Supports: {{schema:1 columns:title,status filter:priority>High,created<2025-01-01}}
  const regex = /\{\{schema(?:_table)?:\s*(\d+)(?:\s+columns:\s*([^\}]+))?(?:\s+filter:\s*([^\}]+))?\}\}/gi;
  return body.replace(regex, (match, schemaId, columns, filters) => {
    let placeholder = `&SCHEMATABLE:${schemaId}`;
    if (columns) {
      const columnsList = columns.split(',').map((c: string) => c.trim()).join(',');
      placeholder += `|${columnsList}`;
    }
    if (filters) {
      // Encode filters (preserve spaces in values by encoding special chars)
      const encodedFilters = filters.replace(/&/g, '%26').replace(/\|/g, '%7C');
      placeholder += columns ? `|${encodedFilters}` : `||${encodedFilters}`;
    }
    return placeholder + '&';
  });
}

// Preprocess entity highlighting by injecting placeholder markers into markdown text
function preprocessEntities(body: string, entities?: Entity[]): string {
  if (!entities || entities.length === 0) return body;

  // Define regions to protect from entity highlighting
  const protectedRegions: Array<{ start: number; end: number }> = [];

  // Protect task query markers &TASKQUERY:..&
  const taskQueryRegex = /&TASKQUERY:[^&]+&/g;
  let match;
  while ((match = taskQueryRegex.exec(body)) !== null) {
    protectedRegions.push({ start: match.index, end: match.index + match[0].length });
  }

  // Protect markdown links [text](url) - including card links [CardId](#)
  const linkRegex = /\[([^\]]+)\]\(([^)]*)\)/g;
  while ((match = linkRegex.exec(body)) !== null) {
    protectedRegions.push({ start: match.index, end: match.index + match[0].length });
  }

  // Protect inline code `...`
  const inlineCodeRegex = /`[^`]+`/g;
  while ((match = inlineCodeRegex.exec(body)) !== null) {
    protectedRegions.push({ start: match.index, end: match.index + match[0].length });
  }

  // Protect code blocks ```...```
  const codeBlockRegex = /```[\s\S]*?```/g;
  while ((match = codeBlockRegex.exec(body)) !== null) {
    protectedRegions.push({ start: match.index, end: match.index + match[0].length });
  }

  // Sort and merge overlapping protected regions
  protectedRegions.sort((a, b) => a.start - b.start);
  const mergedRegions: Array<{ start: number; end: number }> = [];
  protectedRegions.forEach(region => {
    if (mergedRegions.length === 0 || region.start > mergedRegions[mergedRegions.length - 1].end) {
      mergedRegions.push(region);
    } else {
      mergedRegions[mergedRegions.length - 1].end = Math.max(
        mergedRegions[mergedRegions.length - 1].end,
        region.end
      );
    }
  });

  // Helper function to check if a range is protected
  const isProtected = (start: number, end: number): boolean => {
    return mergedRegions.some(region =>
      (start >= region.start && start < region.end) ||
      (end > region.start && end <= region.end) ||
      (start <= region.start && end >= region.end)
    );
  };

  // Sort entities by length (desc) to give priority to longer entity names
  const sortedEntities = [...entities].sort((a, b) => b.name.length - a.name.length);

  // Collect all matches for all entities that are not in protected regions
  type Match = { start: number; end: number; id: number; text: string };
  const matches: Match[] = [];

  sortedEntities.forEach(entity => {
    const escapedName = entity.name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const regex = new RegExp(`\\b(${escapedName})\\b`, "gi");
    let match;
    while ((match = regex.exec(body)) !== null) {
      const matchStart = match.index;
      const matchEnd = match.index + match[0].length;

      // Only add if not in a protected region
      if (!isProtected(matchStart, matchEnd)) {
        matches.push({
          start: matchStart,
          end: matchEnd,
          id: entity.id,
          text: match[0]
        });
      }
    }
  });

  // Sort matches by start index, then by length descending
  matches.sort((a, b) => a.start - b.start || (b.end - b.start) - (a.end - a.start));

  // Filter to remove overlapping matches, keeping the longest first
  const nonOverlapping: Match[] = [];
  let lastEnd = -1;
  matches.forEach(m => {
    if (m.start >= lastEnd) {
      nonOverlapping.push(m);
      lastEnd = m.end;
    }
  });

  // Build the processed string with replacements
  let result = "";
  let currentIndex = 0;
  nonOverlapping.forEach(m => {
    result += body.slice(currentIndex, m.start);
    result += `&ENTITY:${m.id}:${m.text}&`;
    currentIndex = m.end;
  });
  result += body.slice(currentIndex);

  return result;
}

const CustomImageRenderer: React.FC<CustomImageRendererProps> = ({
  src,
  alt,
  title,
}) => {
  const [imageSrc, setImageSrc] = useState<string>("");

  useEffect(() => {
    if (src) {
      downloadFile(src)
        .then((blobUrl) => {
          if (blobUrl) {
            setImageSrc(blobUrl);
          }
        })
        .catch((error) => {
          console.error("Error fetching image:", error);
        });
    }
  }, [src]);

  if (!imageSrc) {
    return <div>Loading...</div>;
  }

  return (
    <img
      src={imageSrc}
      alt={alt || "Image"}
      title={title || ""}
      style={{ maxWidth: "100%", height: "auto" }}
      onClick={() => console.log(`Image clicked: ${src}`)}
    />
  );
};

function renderCardTextWithDialog(
  card: Card,
  handleViewBacklink: (card_id: number) => void,
  entities?: Entity[]
) {
  const [isEntityDialogOpen, setIsEntityDialogOpen] = React.useState(false);


  const {
    showEntityDialog,
    setShowEntityDialog,
    selectedEntity,
    setSelectedEntity,
  } = useShortcutContext();

  async function handleEntityClickById(id: string, name: string) {
    try {
      const entity = await fetchEntityById(Number(id));
      setSelectedEntity(entity);
    } catch (error) {
      console.error("Failed to fetch entity details:", error);
      const fallbackEntity: Entity = {
        id: Number(id) || 0,
        user_id: 0,
        name,
        type: "UNKNOWN",
        description: "",
        created_at: new Date(),
        updated_at: new Date(),
        card_count: 0,
        card_pk: null,
      };
      setSelectedEntity(fallbackEntity);
    }
    setShowEntityDialog(true);
  }

  const markdown = renderCardText(card, handleViewBacklink, entities, handleEntityClickById as any);
  return (
    <>
      {markdown}
    </>
  );
}

function renderCardText(
  card: Card,
  handleViewBacklink: (card_id: number) => void,
  entities?: Entity[],
  onEntityClick?: (id: string, name: string) => void
) {

  // Preprocess task queries first, then card links, then schema tables, then entities
  let processedBody = preprocessTaskQueries(card.body);
  processedBody = preprocessCardLinks(processedBody);
  processedBody = preprocessSchemaTables(processedBody);
  processedBody = preprocessEntities(processedBody, entities);

  // Custom component for inline code only
  const CustomCode = ({ node, inline, className, children, ...props }: any) => {
    return (
      <code
        className="bg-gray-100 px-1 rounded font-mono text-sm not-prose"
        style={{ display: 'inline', whiteSpace: 'nowrap' }}
        {...props}
      >
        {children}
      </code>
    );
  };

  // Custom component for code blocks (pre)
  const CustomPre = ({ children, ...props }: any) => {
    return (
      <pre className="bg-gray-50 p-3 rounded overflow-x-auto text-gray-800" {...props}>
        {children}
      </pre>
    );
  };

  return (
    <Markdown
      children={processedBody}
      remarkPlugins={[remarkGfm, remarkTaskQuery, remarkEntity, remarkSchemaTable]}
      components={{
        // Add our custom components for code
        code: CustomCode,
        pre: CustomPre,
        a({ children, href, ...props }) {
          // For internal links, href will be "#" and children will be the card ID
          if (href === "#") {
            const cardId = children as string;
            return (
              <CardLinkWithPreview
                currentCard={card}
                card_id={cardId}
                handleViewBacklink={handleViewBacklink}
              />
            );
          }
          // For external links, render a regular anchor tag
          else {
            return (
              <a href={href} target="_blank" rel="noopener noreferrer" {...props}>
                {children}
              </a>
            );
          }
        },
        h1({ children, ...props }) {
          return (<H1 children={children as string} />)
        },
        h2({ children, ...props }) {
          return (<H2 children={children as string} />)
        },
        h3({ children, ...props }) {
          return (<H3 children={children as string} />)
        },
        h4({ children, ...props }) {
          return (<H4 children={children as string} />)
        },
        h5({ children, ...props }) {
          return (<H5 children={children as string} />)
        },
        h6({ children, ...props }) {
          return (<H6 children={children as string} />)
        },
        // Table components
        table({ children, ...props }) {
          return <Table {...props}>{children}</Table>;
        },
        thead({ children, ...props }) {
          return <TableHead {...props}>{children}</TableHead>;
        },
        tbody({ children, ...props }) {
          return <TableBody {...props}>{children}</TableBody>;
        },
        tr({ children, ...props }) {
          return <TableRow {...props}>{children}</TableRow>;
        },
        th({ children, ...props }) {
          return <TableHeader {...props}>{children}</TableHeader>;
        },
        td({ children, ...props }) {
          return <TableCell {...props}>{children}</TableCell>;
        },
        span: ({ node, children, ...props }) => {
          const propsData = (node as any).properties || {};
          if (propsData.className === "entity" || propsData["data-id"]) {
            const id = propsData["data-id"];
            const name = propsData["data-name"] || children;
            return (
              <span
                style={{ backgroundColor: "#fff9c4", cursor: "pointer" }}
                onClick={() => onEntityClick?.(id, name)}
              >
                {name}
              </span>
            );
          }
          return <span {...props}>{children}</span>;
        },
        img({ src, alt, title, ...props }) {
          return (
            <CustomImageRenderer src={src} alt={alt} title={title} {...props} />
          );
        },
        div: ({ node, children, ...props }) => {
          const propsData = (node as any).properties || {};

          // Check if this is a task query container
          if (propsData.className === "task-query-container" || propsData["data-query"] !== undefined) {
            const query = propsData["data-query"] || "";
            return <DynamicTaskList query={query} />;
          }

          // Check if this is a schema table container
          if (propsData.className === "schema-table-container" || propsData["data-schema-id"] !== undefined) {
            const schemaId = propsData["data-schema-id"] || "";
            const columns = propsData["data-columns"];
            const filters = propsData["data-filters"];
            return <DynamicSchemaTable schemaId={schemaId} columns={columns} filters={filters} />;
          }

          // Default div rendering
          return <div {...props}>{children}</div>;
        },
      }}
    />
  );
}

export const CardBody: React.FC<CardBodyProps> = ({ viewingCard, entities }) => {
  const navigate = useNavigate();
  const [isCollapsed, setIsCollapsed] = useState(false);
  const [shouldShowToggle, setShouldShowToggle] = useState(false);
  const contentRef = React.useRef<HTMLDivElement>(null);

  // Height threshold for showing the collapse toggle (in pixels)
  const HEIGHT_THRESHOLD = 600;

  function handleCardClick(card_id: number) {
    navigate(`/app/card/${card_id}`);
  }

  useEffect(() => {
    // Check if content height exceeds threshold
    if (contentRef.current) {
      const height = contentRef.current.scrollHeight;
      setShouldShowToggle(height > HEIGHT_THRESHOLD);
      // Start collapsed if content is too long
      if (height > HEIGHT_THRESHOLD) {
        setIsCollapsed(true);
      }
    }
  }, [viewingCard.body]);

  const toggleCollapse = () => {
    setIsCollapsed(!isCollapsed);
  };

  return (
    <div className="relative">
      <div
        ref={contentRef}
        className={`transition-all duration-300 overflow-hidden ${
          isCollapsed && shouldShowToggle ? `max-h-[${HEIGHT_THRESHOLD}px]` : 'max-h-none'
        }`}
        style={{
          maxHeight: isCollapsed && shouldShowToggle ? `${HEIGHT_THRESHOLD}px` : 'none'
        }}
      >
        {renderCardTextWithDialog(viewingCard, handleCardClick, entities)}
      </div>

      {/* Gradient fade effect when collapsed */}
      {isCollapsed && shouldShowToggle && (
        <div className="absolute bottom-0 left-0 right-0 h-16 bg-gradient-to-t from-white to-transparent pointer-events-none" />
      )}

      {/* Toggle button */}
      {shouldShowToggle && (
        <div className="mt-4 text-center">
          <button
            onClick={toggleCollapse}
            className="inline-flex items-center gap-2 px-4 py-2 text-sm text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded-md transition-colors"
          >
            {isCollapsed ? (
              <>
                Show more
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </>
            ) : (
              <>
                Show less
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
                </svg>
              </>
            )}
          </button>
        </div>
      )}
    </div>
  );
};
