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
import { SortMethod, sortPartialCards } from "../../utils/cards";
import { CardStructuredDataDisplay } from "../schemas/CardStructuredDataDisplay";
import { SortControl as SortControlComponent } from "./SortControl";

interface ViewCardContentSectionProps {
  viewingCard: Card;
  showingSummary: boolean;
  showingAnalysis: boolean;
  latestSummary: SummarizeJobResponse | null;
  analysis: SectionAnalysis[] | null;
  onCreateChildCard: () => void;
  categorizedReferences: CategorizedReferences;
  onAddBacklink: (selectedCard: PartialCard) => void;
  setViewCard: (card: Card) => void;
  setError: (error: string) => void;
  handleOpenEntity: (entity: any) => void;
  summaries: any;
  fileUploadRef: React.RefObject<HTMLInputElement>;
  onSaveCard?: (updatedCard: Card) => void | Promise<void>;
}

export function ViewCardContentSection({
  viewingCard,
  showingSummary,
  showingAnalysis,
  latestSummary,
  analysis,
  onCreateChildCard,
  categorizedReferences,
  onAddBacklink,
  setViewCard,
  setError,
  handleOpenEntity,
  summaries,
  fileUploadRef,
  onSaveCard
}: ViewCardContentSectionProps) {
  const [childrenSortMethod, setChildrenSortMethod] = useState<SortMethod>("cardId");
  const [referencesSortMethod, setReferencesSortMethod] = useState<SortMethod>("cardId");

  const sortedChildren = sortPartialCards(viewingCard.children, childrenSortMethod);
  const sortedBidirectional = sortPartialCards(categorizedReferences.bidirectional, referencesSortMethod);
  const sortedIncoming = sortPartialCards(categorizedReferences.incoming, referencesSortMethod);
  const sortedOutgoing = sortPartialCards(categorizedReferences.outgoing, referencesSortMethod);

  return (
    <div className="space-y-4">
      <div
        className={`rounded-lg py-3 prose prose-sm shadow-sm max-w-none px-3 ${showingSummary ? "bg-yellow-50 border border-yellow-200" : showingAnalysis ? "bg-blue-50 border border-blue-200" : "bg-white"
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
        <hr className="my-4" />
      </div>

      <div>
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-3">
            <HeaderSubSection text="References" />
            <SortControlComponent
              sortMethod={referencesSortMethod}
              onSortChange={setReferencesSortMethod}
            />
          </div>
        </div>

        {/* Bidirectional Links */}
        {sortedBidirectional.length > 0 && (
          <div className="mb-3">
            <h3 className="text-xs font-medium text-gray-600 mb-1.5">
              Two-way Links ({sortedBidirectional.length})
            </h3>
            <CardList cards={sortedBidirectional} />
          </div>
        )}

        {/* Incoming Links */}
        {sortedIncoming.length > 0 && (
          <div className="mb-3">
            <h3 className="text-xs font-medium text-gray-600 mb-1.5">
              Incoming Links ({sortedIncoming.length})
            </h3>
            <CardList cards={sortedIncoming} />
          </div>
        )}

        {/* Outgoing Links */}
        {sortedOutgoing.length > 0 && (
          <div className="mb-3">
            <h3 className="text-xs font-medium text-gray-600 mb-1.5">
              Outgoing Links ({sortedOutgoing.length})
            </h3>
            <CardList cards={sortedOutgoing} />
          </div>
        )}

        {/* Show message if no references at all */}
        {sortedBidirectional.length === 0 &&
         sortedIncoming.length === 0 &&
         sortedOutgoing.length === 0 && (
          <div className="text-gray-500 text-sm mt-2">No references yet.</div>
        )}

        <div className="mt-4">
          <BacklinkInput addBacklink={onAddBacklink} excludeCardId={viewingCard.id} />
        </div>
        <hr className="my-4" />
      </div>

      {/* Tasks Section */}
      {viewingCard.tasks.length > 0 && (
        <div className="bg-white rounded-lg p-4 shadow-sm">
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

      {/* Tabbed Display (entities, facts, etc.) */}
      <div className="bg-white rounded-lg shadow-sm">
        <ViewCardTabbedDisplay
          viewingCard={viewingCard}
          setViewCard={setViewCard}
          setError={setError}
          handleOpenEntity={handleOpenEntity}
          summaries={summaries}
          fileUploadRef={fileUploadRef}
        />
      </div>
    </div>
  );
}