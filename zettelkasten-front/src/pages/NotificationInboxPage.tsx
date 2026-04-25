import React, { useState, useCallback, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { setDocumentTitle } from "../utils/title";
import {
  listNotifications,
  markAsRead,
  getUnreadCount,
  Notification,
} from "../api/notifications";
import { fetchTask, saveExistingTask } from "../api/tasks";
import { RSSArticle, getArticle } from "../api/rss";
import { CreateTaskWindow } from "../components/tasks/CreateTaskWindow";
import { RssConvertDialog } from "../components/rss/RssConvertDialog";
import { useAuth } from "../contexts/AuthContext";

type SourceTab = "all" | "rss" | "task";

/**
 * Notification Inbox Page
 *
 * Displays a unified inbox of unread notifications from multiple sources:
 * - RSS article notifications
 * - Task notifications
 *
 * Features:
 * - Filter by source type (All, RSS, Tasks)
 * - Mark as read removes notification from list (hides read notifications)
 * - Navigate to source content by clicking notification
 * - Context-aware actions: Card (RSS), Done (tasks)
 * - Pagination with "Load more" button
 * - Unread count display
 */
export function NotificationInboxPage() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const userTimezone = user?.timezone || "UTC";

  // State management
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [loadingMore, setLoadingMore] = useState<boolean>(false);
  const [activeTab, setActiveTab] = useState<SourceTab>("all");
  const [unreadCount, setUnreadCount] = useState<number>(0);
  const [total, setTotal] = useState<number>(0);
  const [offset, setOffset] = useState<number>(0);
  const [hasMore, setHasMore] = useState<boolean>(true);
  const [markingRead, setMarkingRead] = useState<Set<number>>(new Set());
  const [togglingTaskComplete, setTogglingTaskComplete] = useState<Set<number>>(new Set());
  const [showCreateTaskWindow, setShowCreateTaskWindow] = useState(false);
  const [notificationForTask, setNotificationForTask] = useState<Notification | null>(null);
  const [showConvertDialog, setShowConvertDialog] = useState(false);
  const [articleToConvert, setArticleToConvert] = useState<RSSArticle | null>(null);

  const ITEMS_PER_PAGE = 25;

  /**
   * Fetch unread count
   */
  const fetchUnreadCount = useCallback(async () => {
    try {
      const response = await getUnreadCount();
      setUnreadCount(response.unread_count);
    } catch (error) {
      console.error("Failed to fetch unread count:", error);
    }
  }, []);

  /**
   * Fetch notifications with current filters
   */
  const fetchNotifications = useCallback(async (currentOffset = 0, isLoadMore = false) => {
    if (isLoadMore) {
      setLoadingMore(true);
    } else {
      setLoading(true);
    }

    try {
      const response = await listNotifications({
        source_type: activeTab === "all" ? undefined : activeTab,
        limit: ITEMS_PER_PAGE,
        offset: currentOffset,
      });

      if (isLoadMore) {
        setNotifications((prev) => [...prev, ...response.notifications]);
      } else {
        setNotifications(response.notifications);
      }

      setTotal(response.total);
      setHasMore(response.notifications.length === ITEMS_PER_PAGE);
      setOffset(currentOffset);
    } catch (error) {
      console.error("Failed to fetch notifications:", error);
      if (!isLoadMore) {
        setNotifications([]);
      }
      setTotal(0);
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  }, [activeTab, ITEMS_PER_PAGE]);

  // Initial data fetch
  useEffect(() => {
    fetchNotifications(0, false);
    fetchUnreadCount();
  }, [activeTab, fetchNotifications, fetchUnreadCount]);

  // Suppress unused variable warning
  void userTimezone;

  // Update document title
  useEffect(() => {
    if (unreadCount > 0) {
      setDocumentTitle(`Notifications (${unreadCount})`);
    } else {
      setDocumentTitle("Notifications");
    }
  }, [unreadCount]);

  /**
   * Handle notification click - mark as read and navigate to source
   */
  const handleNotificationClick = async (notification: Notification) => {
    // Mark as read if not already read
    if (!notification.is_read) {
      try {
        await markAsRead(notification.id, true);
        setNotifications((prev) =>
          prev.map((n) =>
            n.id === notification.id ? { ...n, is_read: true } : n
          )
        );
        setUnreadCount((prev) => Math.max(0, prev - 1));
      } catch (error) {
        console.error("Failed to mark notification as read:", error);
      }
    }

    // Navigate to source based on type
    switch (notification.source_type) {
      case "rss":
        navigate(`/app/rss?article=${notification.source_id}`);
        break;
      case "task":
        navigate("/app");
        break;
      default:
        console.warn("Unknown notification source type:", notification.source_type);
    }
  };

  /**
   * Handle mark as read (removes notification from list)
   */
  const handleMarkRead = async (notification: Notification, event: React.MouseEvent) => {
    event.stopPropagation();

    // Only mark if not already read
    if (notification.is_read) return;

    setMarkingRead((prev) => new Set(prev).add(notification.id));

    try {
      await markAsRead(notification.id, true);

      // Remove from list after marking as read
      setNotifications((prev) => prev.filter((n) => n.id !== notification.id));
      setTotal((prev) => prev - 1);
      setUnreadCount((prev) => Math.max(0, prev - 1));
    } catch (error) {
      console.error("Failed to mark notification as read:", error);
    } finally {
      setMarkingRead((prev) => {
        const next = new Set(prev);
        next.delete(notification.id);
        return next;
      });
    }
  };

  /**
   * Handle load more notifications
   */
  const handleLoadMore = () => {
    const newOffset = offset + ITEMS_PER_PAGE;
    fetchNotifications(newOffset, true);
  };

  /**
   * Handle create task from notification
   */
  const handleCreateTaskFromNotification = (notification: Notification, event: React.MouseEvent) => {
    event.stopPropagation();
    setNotificationForTask(notification);
    setShowCreateTaskWindow(true);
  };

  /**
   * Handle close create task window
   */
  const handleCloseTaskWindow = () => {
    setShowCreateTaskWindow(false);
    setNotificationForTask(null);
  };

  /**
   * Handle convert RSS article to card - opens convert dialog
   */
  const handleConvertToCard = async (notification: Notification, event: React.MouseEvent) => {
    event.stopPropagation();

    if (notification.source_type !== "rss") return;

    try {
      // Fetch the article details for the dialog
      const article = await getArticle(notification.source_id);
      setArticleToConvert(article);
      setShowConvertDialog(true);
    } catch (error) {
      console.error("Failed to fetch article for conversion:", error);
    }
  };

  /**
   * Handle successful card conversion
   */
  const handleConverted = (cardId: number) => {
    setShowConvertDialog(false);
    setArticleToConvert(null);

    // Remove the notification from the list
    if (articleToConvert) {
      setNotifications((prev) => prev.filter((n) => n.source_id !== articleToConvert.id));
      setTotal((prev) => prev - 1);
      setUnreadCount((prev) => Math.max(0, prev - 1));
    }

    // Navigate to the new card
    navigate(`/app/card/${cardId}`);
  };

  /**
   * Handle close convert dialog
   */
  const handleCloseConvertDialog = () => {
    setShowConvertDialog(false);
    setArticleToConvert(null);
  };

  /**
   * Handle toggle task complete/incomplete
   */
  const handleToggleTaskComplete = async (notification: Notification, event: React.MouseEvent) => {
    event.stopPropagation();

    setTogglingTaskComplete((prev) => new Set(prev).add(notification.id));

    try {
      // Fetch the current task state
      const task = await fetchTask(notification.source_id.toString());

      // Toggle the complete state
      const updatedTask = {
        ...task,
        is_complete: !task.is_complete,
        completed_at: !task.is_complete ? new Date() : null,
      };

      await saveExistingTask(updatedTask);

      // Mark notification as read when action is taken
      if (!notification.is_read) {
        await markAsRead(notification.id, true);
        setNotifications((prev) =>
          prev.map((n) =>
            n.id === notification.id ? { ...n, is_read: true } : n
          )
        );
        setUnreadCount((prev) => Math.max(0, prev - 1));
      }
    } catch (error) {
      console.error("Failed to toggle task completion:", error);
    } finally {
      setTogglingTaskComplete((prev) => {
        const next = new Set(prev);
        next.delete(notification.id);
        return next;
      });
    }
  };

  /**
   * Get source icon for notification
   */
  const getSourceIcon = (sourceType: string): string => {
    switch (sourceType) {
      case "rss":
        return "📰";
      case "task":
        return "✓";
      default:
        return "📄";
    }
  };

  /**
   * Format timestamp for display
   */
  const formatTimestamp = (timestamp: string): string => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return "Just now";
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;

    return date.toLocaleDateString();
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100vh" }}>
      {/* Header */}
      <div
        style={{
          borderBottom: "1px solid #e5e7eb",
          backgroundColor: "#ffffff",
          padding: "16px 24px",
        }}
      >
        {/* Title and unread count */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            marginBottom: "16px",
          }}
        >
          <h1
            style={{
              fontSize: "24px",
              fontWeight: "700",
              color: "#111827",
              margin: 0,
            }}
          >
            Notifications
          </h1>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "16px",
            }}
          >
            {unreadCount > 0 && (
              <div style={{ fontSize: "14px", color: "#6b7280" }}>
                {unreadCount} unread
              </div>
            )}
            {total > 0 && (
              <div style={{ fontSize: "14px", color: "#6b7280" }}>
                {total} total
              </div>
            )}
          </div>
        </div>

        {/* Tab buttons */}
        <div style={{ display: "flex", gap: "8px" }}>
          <TabButton active={activeTab === "all"} onClick={() => setActiveTab("all")}>
            All
          </TabButton>
          <TabButton active={activeTab === "rss"} onClick={() => setActiveTab("rss")}>
            <span style={{ marginRight: "4px" }}>📰</span>
            RSS
          </TabButton>
          <TabButton active={activeTab === "task"} onClick={() => setActiveTab("task")}>
            <span style={{ marginRight: "4px" }}>✓</span>
            Tasks
          </TabButton>
        </div>
      </div>

      {/* Notification list */}
      <div
        style={{
          flex: 1,
          overflowY: "auto",
          backgroundColor: "#ffffff",
        }}
      >
        {loading ? (
          <div
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              height: "200px",
              color: "#6b7280",
            }}
          >
            Loading notifications...
          </div>
        ) : notifications.length === 0 ? (
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
              justifyContent: "center",
              height: "300px",
              color: "#6b7280",
              textAlign: "center",
              padding: "24px",
            }}
          >
            <div style={{ fontSize: "48px", marginBottom: "16px" }}>📭</div>
            <p style={{ fontSize: "16px", marginBottom: "8px" }}>No notifications</p>
            <p style={{ fontSize: "14px" }}>
              {activeTab === "all"
                ? "You're all caught up!"
                : `No ${activeTab} notifications`}
            </p>
          </div>
        ) : (
          <>
            {notifications.map((notification) => (
              <NotificationItem
                key={notification.id}
                notification={notification}
                onClick={() => handleNotificationClick(notification)}
                onMarkRead={(e) => handleMarkRead(notification, e)}
                onCreateTask={(e) => handleCreateTaskFromNotification(notification, e)}
                onConvertToCard={notification.source_type === "rss" ? (e) => handleConvertToCard(notification, e) : undefined}
                onToggleComplete={notification.source_type === "task" ? (e) => handleToggleTaskComplete(notification, e) : undefined}
                isMarkingRead={markingRead.has(notification.id)}
                isTogglingComplete={togglingTaskComplete.has(notification.id)}
                getSourceIcon={getSourceIcon}
                formatTimestamp={formatTimestamp}
              />
            ))}

            {/* Load more button */}
            {hasMore && !loading && (
              <div style={{ display: "flex", justifyContent: "center", padding: "16px" }}>
                <button
                  onClick={handleLoadMore}
                  disabled={loadingMore}
                  style={{
                    padding: "10px 24px",
                    fontSize: "14px",
                    fontWeight: "500",
                    borderRadius: "8px",
                    border: "1px solid #d1d5db",
                    backgroundColor: loadingMore ? "#f3f4f6" : "#ffffff",
                    color: loadingMore ? "#9ca3af" : "#374151",
                    cursor: loadingMore ? "not-allowed" : "pointer",
                    opacity: loadingMore ? 0.6 : 1,
                    transition: "all 0.15s ease",
                  }}
                  onMouseEnter={(e) => {
                    if (!loadingMore) {
                      e.currentTarget.style.backgroundColor = "#f9fafb";
                      e.currentTarget.style.borderColor = "#9ca3af";
                    }
                  }}
                  onMouseLeave={(e) => {
                    if (!loadingMore) {
                      e.currentTarget.style.backgroundColor = "#ffffff";
                      e.currentTarget.style.borderColor = "#d1d5db";
                    }
                  }}
                >
                  {loadingMore ? "Loading..." : "Load more"}
                </button>
              </div>
            )}
          </>
        )}
      </div>

      {/* Create Task Window */}
      {showCreateTaskWindow && (
        <CreateTaskWindow
          currentCard={null}
          setShowTaskWindow={handleCloseTaskWindow}
          currentFilter={notificationForTask?.title ? `Notification: ${notificationForTask.title}` : undefined}
        />
      )}

      {/* RSS Convert to Card Dialog */}
      <RssConvertDialog
        isOpen={showConvertDialog}
        onClose={handleCloseConvertDialog}
        article={articleToConvert}
        onConverted={handleConverted}
      />
    </div>
  );
}

/**
 * Tab button component
 */
interface TabButtonProps {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}

function TabButton({ active, onClick, children }: TabButtonProps) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: "8px 16px",
        fontSize: "14px",
        fontWeight: "500",
        borderRadius: "8px",
        border: "none",
        cursor: "pointer",
        transition: "all 0.15s ease",
        backgroundColor: active ? "#3b82f6" : "#f3f4f6",
        color: active ? "#ffffff" : "#374151",
      }}
      onMouseEnter={(e) => {
        if (!active) {
          e.currentTarget.style.backgroundColor = "#e5e7eb";
        }
      }}
      onMouseLeave={(e) => {
        if (!active) {
          e.currentTarget.style.backgroundColor = "#f3f4f6";
        }
      }}
    >
      {children}
    </button>
  );
}

/**
 * Notification item component
 */
interface NotificationItemProps {
  notification: Notification;
  onClick: () => void;
  onMarkRead: (event: React.MouseEvent) => void;
  onCreateTask?: (event: React.MouseEvent) => void;
  onConvertToCard?: (event: React.MouseEvent) => void;
  onToggleComplete?: (event: React.MouseEvent) => void;
  isMarkingRead: boolean;
  isTogglingComplete?: boolean;
  getSourceIcon: (sourceType: string) => string;
  formatTimestamp: (timestamp: string) => string;
}

function NotificationItem({
  notification,
  onClick,
  onMarkRead,
  onCreateTask,
  onConvertToCard,
  onToggleComplete,
  isMarkingRead,
  isTogglingComplete,
  getSourceIcon,
  formatTimestamp,
}: NotificationItemProps) {
  return (
    <div
      onClick={onClick}
      style={{
        display: "flex",
        alignItems: "flex-start",
        padding: "16px 24px",
        borderBottom: "1px solid #f3f4f6",
        cursor: "pointer",
        backgroundColor: notification.is_read ? "#ffffff" : "#f9fafb",
        transition: "background-color 0.15s ease",
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.backgroundColor = "#f3f4f6";
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.backgroundColor = notification.is_read
          ? "#ffffff"
          : "#f9fafb";
      }}
    >
      {/* Source icon */}
      <div
        style={{
          fontSize: "20px",
          marginRight: "16px",
          flexShrink: 0,
        }}
      >
        {getSourceIcon(notification.source_type)}
      </div>

      {/* Content */}
      <div style={{ flex: 1, minWidth: 0 }}>
        {/* Title row */}
        <div
          style={{
            display: "flex",
            alignItems: "flex-start",
            gap: "8px",
            marginBottom: "4px",
          }}
        >
          <span
            style={{
              fontSize: "15px",
              fontWeight: notification.is_read ? "400" : "600",
              color: "#111827",
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
            }}
          >
            {notification.title}
          </span>
          {!notification.is_read && (
            <span
              style={{
                width: "8px",
                height: "8px",
                borderRadius: "50%",
                backgroundColor: "#3b82f6",
                flexShrink: 0,
                marginTop: "6px",
              }}
            />
          )}
        </div>

        {/* Preview */}
        <div
          style={{
            fontSize: "14px",
            color: "#6b7280",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
            marginBottom: "6px",
          }}
        >
          {notification.preview}
        </div>

        {/* Metadata row */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "12px",
            fontSize: "12px",
            color: "#9ca3af",
          }}
        >
          <span>{formatTimestamp(notification.timestamp)}</span>
          {notification.filter_tags.length > 0 && (
            <>
              <span style={{ color: "#d1d5db" }}>•</span>
              <div style={{ display: "flex", gap: "4px" }}>
                {notification.filter_tags.slice(0, 2).map((tag) => (
                  <span
                    key={tag}
                    style={{
                      padding: "2px 6px",
                      backgroundColor: "#f3f4f6",
                      borderRadius: "4px",
                      fontSize: "11px",
                    }}
                  >
                    {tag}
                  </span>
                ))}
                {notification.filter_tags.length > 2 && (
                  <span style={{ color: "#9ca3af" }}>
                    +{notification.filter_tags.length - 2}
                  </span>
                )}
              </div>
            </>
          )}
        </div>
      </div>

      {/* Action buttons */}
      <div
        style={{
          display: "flex",
          gap: "8px",
          marginLeft: "12px",
          flexShrink: 0,
        }}
      >
        {/* Create Task button */}
        {onCreateTask && (
          <button
            onClick={onCreateTask}
            style={{
              padding: "6px 10px",
              fontSize: "13px",
              fontWeight: "500",
              borderRadius: "6px",
              border: "none",
              backgroundColor: "#eff6ff",
              color: "#1d4ed8",
              cursor: "pointer",
              display: "flex",
              alignItems: "center",
              gap: "4px",
              transition: "all 0.15s ease",
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = "#dbeafe";
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.backgroundColor = "#eff6ff";
            }}
            title="Create task from this notification"
          >
            <span>✓</span>
            <span>Task</span>
          </button>
        )}

        {/* Convert to Card button (for RSS notifications) */}
        {notification.source_type === "rss" && onConvertToCard && (
          <button
            onClick={onConvertToCard}
            style={{
              padding: "6px 10px",
              fontSize: "13px",
              fontWeight: "500",
              borderRadius: "6px",
              border: "none",
              backgroundColor: "#f0fdf4",
              color: "#15803d",
              cursor: "pointer",
              display: "flex",
              alignItems: "center",
              gap: "4px",
              transition: "all 0.15s ease",
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = "#dcfce7";
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.backgroundColor = "#f0fdf4";
            }}
            title="Convert article to card"
          >
            <span>📄</span>
            <span>Card</span>
          </button>
        )}

        {/* Toggle Complete button (for task notifications) */}
        {notification.source_type === "task" && onToggleComplete && (
          <button
            onClick={onToggleComplete}
            disabled={isTogglingComplete}
            style={{
              padding: "6px 10px",
              fontSize: "13px",
              fontWeight: "500",
              borderRadius: "6px",
              border: "none",
              backgroundColor: isTogglingComplete ? "#f3f4f6" : "#f0fdf4",
              color: isTogglingComplete ? "#9ca3af" : "#15803d",
              cursor: isTogglingComplete ? "not-allowed" : "pointer",
              opacity: isTogglingComplete ? 0.6 : 1,
              display: "flex",
              alignItems: "center",
              gap: "4px",
              transition: "all 0.15s ease",
            }}
            onMouseEnter={(e) => {
              if (!isTogglingComplete) {
                e.currentTarget.style.backgroundColor = "#dcfce7";
              }
            }}
            onMouseLeave={(e) => {
              if (!isTogglingComplete) {
                e.currentTarget.style.backgroundColor = "#f0fdf4";
              }
            }}
            title="Toggle task completion"
          >
            <span>✓</span>
            <span>{isTogglingComplete ? "..." : "Done"}</span>
          </button>
        )}

        {/* Mark read button (all types, only for unread notifications) */}
        {!notification.is_read && (
          <button
            onClick={onMarkRead}
            disabled={isMarkingRead}
            style={{
              padding: "6px 10px",
              fontSize: "13px",
              fontWeight: "500",
              borderRadius: "6px",
              border: "none",
              backgroundColor: isMarkingRead ? "#f3f4f6" : "#eff6ff",
              color: isMarkingRead ? "#9ca3af" : "#1d4ed8",
              cursor: isMarkingRead ? "not-allowed" : "pointer",
              opacity: isMarkingRead ? 0.6 : 1,
              display: "flex",
              alignItems: "center",
              gap: "4px",
              transition: "all 0.15s ease",
            }}
            onMouseEnter={(e) => {
              if (!isMarkingRead) {
                e.currentTarget.style.backgroundColor = "#dbeafe";
              }
            }}
            onMouseLeave={(e) => {
              if (!isMarkingRead) {
                e.currentTarget.style.backgroundColor = "#eff6ff";
              }
            }}
            title="Mark as read"
          >
            <span>✓</span>
            <span>{isMarkingRead ? "..." : "Read"}</span>
          </button>
        )}
      </div>
    </div>
  );
}
