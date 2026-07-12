import React, { useState } from "react";
import ReactMarkdown from "react-markdown";
import { TaskListItem } from "../tasks/TaskListItem";
import { Card, PartialCard } from "../../models/Card";
import { SectionAnalysis, SummarizeJobResponse } from "../../api/summarizer";
import { CategorizedReferences } from "../../api/cards";
import { HeaderSubSection } from "../Header";
import { ChildrenCards } from "./ChildrenCards";
import { CardList } from "./CardList";
import { BacklinkInput } from "./BacklinkInput";
import { CardBody } from "./CardBody";
import { ViewCardTabbedDisplay } from "./ViewCardTabbedDisplay";
import { Collapsible } from "../Collapsible";
import { SortMethod, sortPartialCards } from "../../utils/cards";
import { SortControl as SortControlComponent } from "./SortControl";

interface ViewCardContentSectionProps {
  viewingCard: Card;
  showingSummary?: boolean;
  showingAnalysis?: boolean;
  latestSummary: SummarizeJobResponse | null;
  analysis: SectionAnalysis[] | null;
  onCreateChildCard: () => void;
  onSaveCard?: (updatedCard: Card) => void | Promise<void>;
  /**
   * When true (mobile), render Children + Linked references inline as before.
   * When false/absent (desktop), those live in the right rail's Links tab and
   * this section shows a footer affordance to open it instead.
   */
  showRelationships?: boolean;
  categorizedReferences?: CategorizedReferences;
  onAddBacklink?: (selectedCard: PartialCard) => void;
  /**
   * When true (mobile), render the entities/files/history/summaries tabbed
   * display inline. When false/absent (desktop), it lives in the rail's
   * Metadata tab, so the props below are only needed on the mobile path.
   */
  showTabbedDisplay?: boolean;
  setViewCard?: (card: Card) => void;
  setError?: (error: string) => void;
  summaries?: any;
  fileUploadRef?: React.RefObject<HTMLInputElement>;
}

export function ViewCardContentSection({
  viewingCard,
  showingSummary = false,
  showingAnalysis = false,
  latestSummary,
  analysis,
  onCreateChildCard,
  onSaveCard,
  showRelationships = false,
  showTabbedDisplay = false,
  categorizedReferences,
  onAddBacklink,
  setViewCard,
  setError,
  summaries,
  fileUploadRef,
}: ViewCardContentSectionProps) {
  const [childrenSortMethod, setChildrenSortMethod] = useState<SortMethod>("cardId");
  const [referencesSortMethod, setReferencesSortMethod] = useState<SortMethod>("cardId");

  // Relationship data only computed for the mobile inline path.
  const sortedChildren = showRelationships
    ? sortPartialCards(viewingCard.children, childrenSortMethod)
    : [];
  const sortedBidirectional = showRelationships && categorizedReferences
    ? sortPartialCards(categorizedReferences.bidirectional, referencesSortMethod)
    : [];
  const sortedIncoming = showRelationships && categorizedReferences
    ? sortPartialCards(categorizedReferences.incoming, referencesSortMethod)
    : [];
  const sortedOutgoing = showRelationships && categorizedReferences
    ? sortPartialCards(categorizedReferences.outgoing, referencesSortMethod)
    : [];
  const totalReferences =
    sortedBidirectional.length + sortedIncoming.length + sortedOutgoing.length;

  return (
    <div className="space-y-8">
      <div
        className={`prose prose-sm max-w-none ${
          showingSummary
            ? "bg-yellow-50 border border-yellow-200 rounded-lg px-4 py-3"
            : showingAnalysis
            ? "bg-blue-50 border border-blue-200 rounded-lg px-4 py-3"
            : ""
        }`}
      >
        {showingSummary && latestSummary?.result ? (
          <div>
            <div className="bg-yellow-100 text-yellow-800 font-semibold text-sm px-3 py-2 rounded-md mb-4">
              Summary View
            </div>
            <div className="prose prose-sm">
              <ReactMarkdown>{latestSummary.result}</ReactMarkdown>
            </div>
          </div>
        ) : showingAnalysis && analysis ? (
          <div>
            <div className="bg-blue-100 text-blue-800 font-semibold text-sm px-3 py-2 rounded-md mb-4">
              Analysis View
            </div>
            {analysis.map((section, index) => (
              <div key={index} className="mb-4">
                <h2 className="font-bold text-base">{section.section}</h2>
                {section.theses && section.theses.map((thesis, thesisIndex) => (
                  <div key={thesisIndex} className="ml-4 mt-2">
                    <span className="text-sm">{thesis.thesis}</span>
                    {thesis.arguments && thesis.arguments.length > 0 && (
                      <div className="ml-4">
                        <ul className="list-disc ml-5 text-sm">
                          {thesis.arguments.map((arg, argIndex) => (
                            <li key={argIndex}>{arg.argument}</li>
                          ))}
                        </ul>
                      </div>
                    )}
                    <hr />
                  </div>
                ))}
              </div>
            ))}
          </div>
        ) : (
          <CardBody
            viewingCard={viewingCard}
            entities={viewingCard.entities}
            onSave={onSaveCard}
          />
        )}
      </div>

      {showRelationships && (
        <>
          {/* Children — inline (mobile path). Desktop shows these in the rail. */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-3">
                <HeaderSubSection text="Children" />
                <SortControlComponent
                  sortMethod={childrenSortMethod}
                  onSortChange={setChildrenSortMethod}
                />
              </div>
              <button
                onClick={onCreateChildCard}
                className="text-blue-500 hover:text-blue-700"
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
                </svg>
              </button>
            </div>
            {sortedChildren.length > 0 ? (
              <ChildrenCards
                allChildren={sortedChildren}
                card={viewingCard}
              />
            ) : (
              <div className="text-gray-500 text-sm mt-2">No children yet.</div>
            )}
          </div>

          <Collapsible
            title="Linked references"
            count={totalReferences}
            defaultOpen={totalReferences > 0}
            rightElement={
              <SortControlComponent
                sortMethod={referencesSortMethod}
                onSortChange={setReferencesSortMethod}
              />
            }
          >
            {sortedBidirectional.length > 0 && (
              <div className="mb-3">
                <h3 className="text-xs font-medium text-gray-600 mb-1.5">
                  Two-way Links ({sortedBidirectional.length})
                </h3>
                <CardList cards={sortedBidirectional} />
              </div>
            )}

            {sortedIncoming.length > 0 && (
              <div className="mb-3">
                <h3 className="text-xs font-medium text-gray-600 mb-1.5">
                  Incoming Links ({sortedIncoming.length})
                </h3>
                <CardList cards={sortedIncoming} />
              </div>
            )}

            {sortedOutgoing.length > 0 && (
              <div className="mb-3">
                <h3 className="text-xs font-medium text-gray-600 mb-1.5">
                  Outgoing Links ({sortedOutgoing.length})
                </h3>
                <CardList cards={sortedOutgoing} />
              </div>
            )}

            {totalReferences === 0 && (
              <div className="text-gray-500 text-sm">
                No references yet.
              </div>
            )}

            {onAddBacklink && (
              <div className="mt-4">
                <BacklinkInput addBacklink={onAddBacklink} excludeCardId={viewingCard.id} />
              </div>
            )}
          </Collapsible>
        </>
      )}

      {/* Tasks Section */}
      {viewingCard.tasks.length > 0 && (
        <div>
          <HeaderSubSection text="Tasks" />
          <div className="mt-2 space-y-2">
            {viewingCard.tasks.map((task, index) => (
              <TaskListItem
                key={task.id}
                task={task}
                onTagClick={(tag: string) => { }}
              />
            ))}
          </div>
        </div>
      )}

      {/* Tabbed Display (entities, files, history, summaries).
           Desktop renders this in the rail's Metadata tab; mobile keeps it
           inline behind the showTabbedDisplay flag. */}
      {showTabbedDisplay && (
        <div className="pt-2">
          <ViewCardTabbedDisplay
            viewingCard={viewingCard}
            setViewCard={setViewCard ?? (() => undefined)}
            setError={setError ?? (() => undefined)}
            summaries={summaries ?? null}
            fileUploadRef={fileUploadRef ?? { current: null }}
          />
        </div>
      )}
    </div>
  );
}
