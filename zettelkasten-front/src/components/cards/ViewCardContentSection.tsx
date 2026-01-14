import React from "react";
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
import { compareCardIds } from "../../utils/cards";

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
  setSelectedFact: (fact: any) => void;
  setShowFactDialog: (show: boolean) => void;
  fileUploadRef: React.RefObject<HTMLInputElement>;
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
  setSelectedFact,
  setShowFactDialog,
  fileUploadRef
}: ViewCardContentSectionProps) {
  return (
    <div className="md:w-2/3 space-y-4">
      <div
        className={`rounded-lg py-4 prose shadow-sm max-w-none px-4 ${showingSummary ? "bg-yellow-50 border border-yellow-200" : showingAnalysis ? "bg-blue-50 border border-blue-200" : "bg-white"
          }`}
      >
        {showingSummary && latestSummary?.result ? (
          <div>
            <div className="bg-yellow-100 text-yellow-800 font-semibold px-3 py-2 rounded-md mb-4">
              Summary View
            </div>
            <ReactMarkdown>{latestSummary.result}</ReactMarkdown>
          </div>
        ) : showingAnalysis && analysis ? (
          <div>
            <div className="bg-blue-100 text-blue-800 font-semibold px-3 py-2 rounded-md mb-4">
              Analysis View
            </div>
            {analysis.map((section, index) => (
              <div key={index} className="mb-4">
                <h2 className="font-bold text-lg">{section.section}</h2>
                {section.theses && section.theses.map((thesis, thesisIndex) => (
                  <div key={thesisIndex} className="ml-4 mt-2">
                    <span className="text-base">{thesis.thesis}</span>
                    {thesis.arguments && thesis.arguments.length > 0 && (
                      <div className="ml-4">
                        <ul className="list-disc ml-5">
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
          />
        )}
      </div>

      <div>
        <div className="flex items-center justify-between">
          <HeaderSubSection text="Children" />
          <button
            onClick={onCreateChildCard}
            className="text-blue-500 hover:text-blue-700"
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
            </svg>
          </button>
        </div>
        {viewingCard.children.length > 0 ? (
          <ChildrenCards
            allChildren={viewingCard.children.sort((a, b) =>
              compareCardIds(a.card_id, b.card_id),
            )}
            card={viewingCard}
          />
        ) : (
          <div className="text-gray-500 text-sm mt-2">No children yet.</div>
        )}
        <hr className="my-4" />
      </div>

      <div>
        <HeaderSubSection text="References" />

        {/* Bidirectional Links */}
        {categorizedReferences.bidirectional.length > 0 && (
          <div className="mb-3">
            <h3 className="text-xs font-medium text-gray-600 mb-1.5">
              Two-way Links ({categorizedReferences.bidirectional.length})
            </h3>
            <CardList
              cards={categorizedReferences.bidirectional.sort((a, b) =>
                compareCardIds(a.card_id, b.card_id),
              )}
            />
          </div>
        )}

        {/* Incoming Links */}
        {categorizedReferences.incoming.length > 0 && (
          <div className="mb-3">
            <h3 className="text-xs font-medium text-gray-600 mb-1.5">
              Incoming Links ({categorizedReferences.incoming.length})
            </h3>
            <CardList
              cards={categorizedReferences.incoming.sort((a, b) =>
                compareCardIds(a.card_id, b.card_id),
              )}
            />
          </div>
        )}

        {/* Outgoing Links */}
        {categorizedReferences.outgoing.length > 0 && (
          <div className="mb-3">
            <h3 className="text-xs font-medium text-gray-600 mb-1.5">
              Outgoing Links ({categorizedReferences.outgoing.length})
            </h3>
            <CardList
              cards={categorizedReferences.outgoing.sort((a, b) =>
                compareCardIds(a.card_id, b.card_id),
              )}
            />
          </div>
        )}

        {/* Show message if no references at all */}
        {categorizedReferences.bidirectional.length === 0 &&
         categorizedReferences.incoming.length === 0 &&
         categorizedReferences.outgoing.length === 0 && (
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
          setSelectedFact={setSelectedFact}
          setShowFactDialog={setShowFactDialog}
          fileUploadRef={fileUploadRef}
        />
      </div>
    </div>
  );
}