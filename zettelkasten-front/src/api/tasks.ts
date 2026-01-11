import { Task, TaskAuditEvent, TasksResponse } from "src/models/Task";
import { checkStatus } from "./common";
import { processTaskFromAPI } from "../utils/taskDataProcessing";

const base_url = import.meta.env.VITE_URL;

export interface FetchTasksParams {
  showCompleted?: boolean;
  scheduledDate?: Date | null;
  completedDate?: Date | null;
  status?: string | null;
}

export function fetchTasks(params: FetchTasksParams = {}): Promise<Task[]> {
  const { showCompleted = false, scheduledDate = null, completedDate = null, status = null } = params;

  const fetchAllTasks = async (offset = 0, allTasks: Task[] = []): Promise<Task[]> => {
    let token = localStorage.getItem("token");
    let url = base_url + `/tasks?limit=100&offset=${offset}`;
    if (showCompleted) {
      url += "&completed=true";
    }
    if (scheduledDate) {
      const dateStr = scheduledDate.toISOString().split('T')[0];
      url += `&scheduled_date=${dateStr}`;
    }
    if (completedDate) {
      const dateStr = completedDate.toISOString().split('T')[0];
      url += `&completed_date=${dateStr}`;
    }
    if (status) {
      url += `&status=${encodeURIComponent(status)}`;
    }

    const response = await fetch(url, {
      headers: { Authorization: `Bearer ${token}` },
    });

    const checkedResponse = await checkStatus(response);
    if (!checkedResponse) {
      throw new Error("Response is undefined");
    }

    const tasksResponse: TasksResponse = await checkedResponse.json();
    if (!tasksResponse.tasks) {
      return allTasks
    }
    const formattedTasks = tasksResponse.tasks.map(task => processTaskFromAPI(task));

    const combinedTasks = [...allTasks, ...formattedTasks];

    // If we got fewer tasks than the limit, we've reached the end
    if (tasksResponse.tasks.length < tasksResponse.limit) {
      return combinedTasks;
    }

    // If there are more tasks to fetch, make another request
    return fetchAllTasks(offset + tasksResponse.limit, combinedTasks);
  };

  return fetchAllTasks();
}

export function fetchTask(id: string): Promise<Task> {
  let encoded = encodeURIComponent(id);
  const url = base_url + `/tasks/${encoded}`;
  let token = localStorage.getItem("token");

  return fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then(rawTask => processTaskFromAPI(rawTask));
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}
export function saveNewTask(task: Task): Promise<Task> {
  const url = base_url + `/tasks`;
  const method = "POST";
  return saveTask(url, method, task);
}

export function saveExistingTask(task: Task): Promise<Task> {
  const url = base_url + `/tasks/${encodeURIComponent(task.id)}`;
  const method = "PUT";
  return saveTask(url, method, task);
}
export function saveTask(
  url: string,
  method: string,
  task: Task,
): Promise<Task> {
  let token = localStorage.getItem("token");
  return fetch(url, {
    method: method,
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(task),
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<Task>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function deleteTask(id: number): Promise<Task | null> {
  let encodedId = encodeURIComponent(id);
  const url = `${base_url}/tasks/${encodedId}`;

  let token = localStorage.getItem("token");

  return fetch(url, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        if (response.status === 204) {
          return null;
        }
        return response.json() as Promise<Task>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function fetchTaskAuditEvents(taskId: number): Promise<TaskAuditEvent[]> {
  const url = `${base_url}/tasks/${taskId}/audit`;
  let token = localStorage.getItem("token");

  return fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((events: TaskAuditEvent[]) => {
          return events.map((event) => ({
            ...event,
            created_at: new Date(event.created_at)
          }));
        });
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function addTaskDependency(taskId: number, blockingTaskId: number): Promise<void> {
  const url = `${base_url}/tasks/${taskId}/dependencies`;
  let token = localStorage.getItem("token");

  return fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ blocking_task_id: blockingTaskId }),
  })
    .then(checkStatus)
    .then(() => undefined);
}

export function removeTaskDependency(taskId: number, blockingTaskId: number): Promise<void> {
  const url = `${base_url}/tasks/${taskId}/dependencies/${blockingTaskId}`;
  let token = localStorage.getItem("token");

  return fetch(url, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then(() => undefined);
}
