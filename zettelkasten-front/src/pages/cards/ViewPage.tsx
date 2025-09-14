import React, { useState, useEffect } from "react";
import { setDocumentTitle } from "../../utils/title";
import { CardItem } from "../../components/cards/CardItem";
import { BacklinkInput } from "../../components/cards/BacklinkInput";
import { getCard, saveExistingCard, starCard, unstarCard, getCardReferences, getCardChildren, getCardFiles, getCardTags, getCardTasks, getCardEntities, getLinkedEntitiesByCardPK } from "../../api/cards";
import { Menu } from "@headlessui/react";
import { useParams } from "react-router-dom";
import { useNavigate } from "react-router-dom";

import { Card, PartialCard, Entity } from "../../models/Card";
import { isErrorResponse } from "../../models/common";
import { TaskListItem } from "../../components/tasks/TaskListItem";
import { useTaskContext } from "../../contexts/TaskContext";
import { useFileContext } from "../../contexts/FileContext";

import { Button } from "../../components/Button";
import { HeaderTop, HeaderSubSection } from "../../components/Header";
import { linkifyWithDefaultOptions } from "../../utils/strings";
import { convertCardToPartialCard } from "../../utils/cards";
import { ViewCardTabbedDisplay } from "../../components/cards/ViewCardTabbedDisplay";

import { usePartialCardContext } from "../../contexts/CardContext";
import { useCardRefresh } from "../../contexts/CardRefreshContext";
import { findNextChildId } from "../../utils/cards";

import { useShortcutContext } from "../../contexts/ShortcutContext";
import { useTagContext } from "../../contexts/TagContext";

import { SearchTagDropdown } from "../../components/tags/SearchTagDropdown";
import { FileUpload } from "../../components/files/FileUpload";

import { ChildrenCards } from "../../components/cards/ChildrenCards";
import { compareCardIds } from "../../utils/cards";
import ReactMarkdown from "react-markdown";

import { CardList } from "../../components/cards/CardList";
import { CardBody } from "../../components/cards/CardBody";
import { fetchSummariesForCard, fetchAnalysisForCard, SectionAnalysis, SummarizeJobResponse } from "../../api/summarizer";
import { FactWithCard } from "../../models/Fact";

interface ViewPageProps { }

export function ViewPage({ }: ViewPageProps) {
  const [error, setError] = useState("");
  const [viewingCard, setViewCard] = useState<Card | null>(null);
  const [parentCard, setParentCard] = useState<Card | null>(null);
  const { refreshTasks, setRefreshTasks } = useTaskContext();
  const { refreshFiles } = useFileContext();
  const { id } = useParams<{ id: string }>();
  const { refreshTrigger } = useCardRefresh();

  const fileUploadRef = React.useRef<HTMLInputElement>(null);

  const {
    showCreateTaskWindow,
    setShowCreateTaskWindow,
    setShowEntityDialog,
    setSelectedEntity,
    setSelectedFact,
    setShowFactDialog,
  } = useShortcutContext();

  const { tags } = useTagContext();

  const [summaries, setSummaries] = useState<SummarizeJobResponse[] | null>(null);
  const [latestSummary, setLatestSummary] = useState<SummarizeJobResponse | null>(null);
  const [showingSummary, setShowingSummary] = useState(false);
  const [analysis, setAnalysis] = useState<SectionAnalysis[] | null>(null);
  const [showingAnalysis, setShowingAnalysis] = useState(false);


  const [linkedEntities, setLinkedEntities] = useState<Entity[]>([]);

  const navigate = useNavigate();

  const { setLastCard, setNextCardId } = usePartialCardContext();

  function handleOpenEntity(entity: Entity) {
    setSelectedEntity(entity)
    setShowEntityDialog(true);
  }

  async function handleTagClick(tagName: string) {
    if (viewingCard === null) {
      return;
    }

    let editedCard = {
      ...viewingCard,
      body: viewingCard.body + "\n\n#" + tagName,
    };
    let response = await saveExistingCard(editedCard);
    setViewCard(editedCard);
  }

  async function handleRemoveTag(tagName: string) {
    if (viewingCard === null) {
      return;
    }

    const tagRegex = new RegExp(`\\n*#${tagName}\\b`, 'g');
    let editedCard = {
      ...viewingCard,
      body: viewingCard.body.replace(tagRegex, ''),
    };
    let response = await saveExistingCard(editedCard);
    setViewCard(editedCard);
    fetchCard(id!);
  }

  async function handleAddBacklink(selectedCard: PartialCard) {
    if (viewingCard === null || selectedCard === null) {
      return;
    }
    let text = "";
    if (selectedCard) {
      text = "\n\n[" + selectedCard.card_id + "] - " + selectedCard.title;
    } else {
      text = "";
    }
    let editedCard = {
      ...viewingCard,
      body: viewingCard.body + text,
    };
    let response = await saveExistingCard(editedCard);
    setViewCard(editedCard);
    fetchCard(id!);
  }



  function handleEditCard() {
    if (viewingCard === null) {
      return;
    }
    navigate(`/app/card/${viewingCard.id}/edit`);
  }

  function handleCreateChildCard() {
    if (viewingCard === null) return;
    const nextId = findNextChildId(viewingCard.card_id, viewingCard.children);
    setNextCardId(nextId);
    navigate('/app/card/new');
  }


  async function loadSummaries(id: number) {
    try {
      const jobs = await fetchSummariesForCard(id);
      setSummaries(jobs);
    } catch (err: any) {
      console.error("Failed to fetch summaries", err);
    }
  }

  async function loadAnalysis(id: number) {
    try {
      const analysisData = await fetchAnalysisForCard(id);
      setAnalysis(analysisData);
    } catch (err: any) {
      console.error("Failed to fetch analysis", err);
    }
  }

  async function fetchCard(id: string) {
    try {
      let refreshed = await getCard(id);

      if (isErrorResponse(refreshed)) {
        setError(refreshed["error"]);
      } else {
        // Also fetch references via new endpoint
        const refs = await getCardReferences(id);
        refreshed.references = refs;
        // Also fetch children via new endpoint
        const kids = await getCardChildren(id);
        refreshed.children = kids;
        // Also fetch files via new endpoint
        const files = await getCardFiles(id);
        refreshed.files = files;
        // Also fetch tags via new endpoint
        const tags = await getCardTags(id);
        refreshed.tags = tags;
        // Also fetch tasks via new endpoint
        const tasks = await getCardTasks(id);
        refreshed.tasks = tasks;
        // Also fetch entities via new endpoint
        const entities = await getCardEntities(id);
        refreshed.entities = entities;

        // Also fetch linked entities via new endpoint
        const linked = await getLinkedEntitiesByCardPK(id);
        setLinkedEntities(linked);

        setViewCard(refreshed);
        setDocumentTitle(refreshed.card_id + " - View");
        setLastCard(convertCardToPartialCard(refreshed));

        if (refreshed.parent && "id" in refreshed.parent) {
          let parentCardId = refreshed.parent.id;
          const parentCard = await getCard(parentCardId.toString());
          setParentCard(parentCard);
        } else {
          setParentCard(null);
        }
      }
    } catch (error: any) {
      setError(error.message);
    }
  }
  const handleToggleStar = async () => {
    if (viewingCard === null) {
      return
    }
    console.log("?", viewingCard)
    const card = viewingCard
    try {
      console.log(viewingCard, viewingCard.is_starred)
      if (viewingCard.is_starred) {
        await unstarCard(viewingCard.id);
        setViewCard({
          ...card,
          is_starred: false
        })
      } else {
        await starCard(viewingCard.id);
        setViewCard({
          ...card,
          is_starred: true
        })
      }
    } catch (error) {
      console.log(error);
    }
  };

  function toggleCreateTaskWindow() {
    setShowCreateTaskWindow(!showCreateTaskWindow);
  }
  // For initial fetch and when id changes
  useEffect(() => {
    setError("");
    fetchCard(id!);
    if (id) {
      loadSummaries(parseInt(id));
      loadAnalysis(parseInt(id));
    }
  }, [id, refreshTasks, refreshFiles, refreshTrigger]);

  useEffect(() => {
    if (summaries && summaries.length > 0) {
      // Filter to only "complete" summaries
      const completeSummaries = summaries.filter(s => s.status === "complete");

      if (completeSummaries.length > 0) {
        // Find the one with the highest ID
        const latest = completeSummaries.reduce((max, s) =>
          s.id > max.id ? s : max
        );

        setLatestSummary(latest);
      } else {
        // Optional: fallback if none are "complete"
        console.log("No complete summaries yet");
        setLatestSummary(null);
      }
    }
  }, [summaries]);

  return (
    <div className="overflow-x-hidden">
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4 mb-4">
          <div className="text-red-700">{error}</div>
        </div>
      )}
      {viewingCard && (
        <div className="space-y-6">
          {/* Header Section */}
          <div className="flex flex-col md:flex-row items-start md:items-center justify-between bg-white rounded-lg p-3 shadow-sm">
            <div className="flex-grow">
              <div className="flex items-center flex-wrap md:flex-nowrap gap-2">
                <span className="font-bold text-gray-600">
                  Viewing:
                </span>

                <span className="text-blue-600">
                  [{viewingCard.card_id}]
                </span>
                <span className="text-gray-600 md:truncate">{"- "}
                  {viewingCard.title}
                </span>
              </div>
            </div>
            <div className="mt-2 md:mt-0 flex justify-end gap-2 flex-shrink">
              <Button onClick={handleEditCard}>Edit</Button>
              <Menu as="div" className="relative inline-block text-right">
                <div>
                  <Menu.Button className="inline-flex justify-center w-full rounded-md border border-gray-300 shadow-sm px-2 py-2 bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-gray-100 focus:ring-indigo-500">
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-5" viewBox="0 0 20 20" fill="currentColor">
                      <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
                    </svg>
                  </Menu.Button>
                </div>
                <Menu.Items className="origin-top-left md:origin-top-right absolute right-0 md:right-0 left-0 md:left-auto mt-2 w-56 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
                  <div className="py-1">
                    <Menu.Item>
                      {({ active }) => (
                        <button
                          onClick={toggleCreateTaskWindow}
                          className={`${active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                            } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                        >
                          Add Task
                        </button>
                      )}
                    </Menu.Item>
                    <Menu.Item>
                      {({ active }) => (
                        <button
                          onClick={handleToggleStar}
                          className={`${active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                            } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                        >
                          {viewingCard.is_starred ? 'Unstar Card' : 'Star Card'}
                        </button>
                      )}
                    </Menu.Item>
                  </div>
                </Menu.Items>
              </Menu>
            </div>
          </div>

          <div className="">
            <div className="flex flex-col md:flex-row gap-4">
              {/* Card Body */}
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
                              {thesis.arguments.length > 0 && (
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
                      onClick={handleCreateChildCard}
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
                  {viewingCard.references.length > 0 ? (
                    <CardList
                      cards={viewingCard.references.sort((a, b) =>
                        compareCardIds(a.card_id, b.card_id),
                      )}
                    />
                  ) : (
                    <div className="text-gray-500 text-sm mt-2">No references yet.</div>
                  )}
                  <div className="mt-4">
                    <BacklinkInput addBacklink={handleAddBacklink} />
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


                {/* Tabbed Display (now also contains Facts) */}
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

              {/* Backlink and Options Section */}
              <div className="md:w-1/3 bg-white rounded-lg p-4 shadow-sm space-y-4">
                {/* Parent Card Section */}
                {parentCard && (
                  <div>
                    <HeaderSubSection text="Parent" />
                    <div className="mt-2">
                      <CardItem card={parentCard} />
                    </div>
                    <hr className="my-4" />
                  </div>
                )}

                {/* Linked Entities Section */}
                {linkedEntities.length > 0 && (
                  <div>
                    <HeaderSubSection text="Linked Entities" />
                    <div className="mt-2 space-y-2">
                      {linkedEntities.map(entity => (
                        <div
                          key={entity.id}
                          className="text-xs text-blue-600 cursor-pointer hover:underline"
                          onClick={() => handleOpenEntity(entity)}
                        >
                          {entity.name}
                        </div>
                      ))}
                    </div>
                    <hr className="my-4" />
                  </div>
                )}

                {/* Card Views Section */}
                {(latestSummary || (analysis && analysis.length > 0)) && (
                  <div>
                    <HeaderSubSection text="Card Views" />
                    <div className="">
                      <button
                        onClick={() => {
                          setShowingSummary(false);
                          setShowingAnalysis(false);
                        }}
                        className={`w-full text-left px-3 py-2 rounded-md text-sm transition-colors ${!showingSummary && !showingAnalysis
                          ? 'bg-blue-100 text-blue-800 font-medium'
                          : 'text-gray-600 hover:bg-gray-100'
                          }`}
                      >
                        📄 Show Card
                      </button>
                      {latestSummary && (
                        <button
                          onClick={() => {
                            setShowingSummary(!showingSummary);
                            if (showingAnalysis) setShowingAnalysis(false);
                          }}
                          className={`w-full text-left px-3 py-2 rounded-md text-sm transition-colors ${showingSummary
                            ? 'bg-yellow-100 text-yellow-800 font-medium'
                            : 'text-gray-600 hover:bg-gray-100'
                            }`}
                        >
                          📝 Show Summary
                        </button>
                      )}
                      {analysis && analysis.length > 0 && (
                        <button
                          onClick={() => {
                            setShowingAnalysis(!showingAnalysis);
                            if (showingSummary) setShowingSummary(false);
                          }}
                          className={`w-full text-left px-3 py-2 rounded-md text-sm transition-colors ${showingAnalysis
                            ? 'bg-blue-100 text-blue-800 font-medium'
                            : 'text-gray-600 hover:bg-gray-100'
                            }`}
                        >
                          🔍 Show Analysis
                        </button>
                      )}
                    </div>
                    <hr className="my-4" />
                  </div>
                )}

                {/* Tags Section */}
                <div>
                  <div className="flex items-center justify-between">
                    <HeaderSubSection text="Tags" />
                    <SearchTagDropdown
                      tags={tags}
                      handleTagClick={handleTagClick}
                    />
                  </div>
                  <div className="flex flex-wrap gap-1.5 mt-2">
                    {viewingCard.tags.map((tag) => (
                      <span
                        key={tag.name}
                        className="inline-flex items-center px-1.5 py-0.5 bg-purple-50 text-purple-600 text-xs rounded-full"
                      >
                        <span
                          className="cursor-pointer hover:bg-purple-100"
                          onClick={() => navigate(`/app/search?term=${encodeURIComponent('#' + tag.name)}`)}
                        >
                          #{tag.name}
                        </span>
                        {viewingCard.body.includes(`#${tag.name}`) && (
                          <button
                            onClick={() => handleRemoveTag(tag.name)}
                            className="ml-1.5 text-purple-400 hover:text-purple-600"
                          >
                            &times;
                          </button>
                        )}
                      </span>
                    ))}
                  </div>
                  <hr className="my-4" />
                </div>



                {/* Details Section */}
                <div className="text-xs text-gray-600 space-y-1 pt-4 border-t">
                  {viewingCard.link && (
                    <div className="flex items-start">
                      <span className="font-medium w-20">Link:</span>
                      <div
                        className="flex-1 break-all"
                        dangerouslySetInnerHTML={{ __html: linkifyWithDefaultOptions(viewingCard.link) }}
                      />
                    </div>
                  )}
                  <div className="flex items-start">
                    <span className="font-medium w-20">Created:</span>
                    <span className="flex-1">{viewingCard.created_at.toISOString()}</span>
                  </div>
                  <div className="flex items-start">
                    <span className="font-medium w-20">Updated:</span>
                    <span className="flex-1">{viewingCard.updated_at.toISOString()}</span>
                  </div>
                </div>

              </div>
            </div>

            {/* Link Section */}

          </div>
        </div>
      )}
    </div>
  );
}
