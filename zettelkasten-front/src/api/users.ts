import {
  User,
  CreateUserParams,
  CreateUserResponse,
  EditUserParams,
  UserSubscription,
} from "../models/User";
import { GenericResponse } from "../models/common";
import { checkStatus } from "./common";

const base_url = import.meta.env.VITE_URL;

export function createUser(
  userData: CreateUserParams,
): Promise<CreateUserResponse> {
  let token = localStorage.getItem("token");
  return fetch(base_url + "/users", {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(userData),
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<CreateUserResponse>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function getUserMemory(): Promise<{ memory: string }> {
  const url = `${base_url}/user/memory`;
  let token = localStorage.getItem("token");

  return fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<{ memory: string }>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function getUser(id: string): Promise<User> {
  let encoded = encodeURIComponent(id);
  const url = base_url + `/users/${encoded}`;
  let token = localStorage.getItem("token");

  // Send a GET request to the URL
  return fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<User>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}
export interface GetUsersParams {
  page?: number;
  per_page?: number;
}

export interface GetUsersResponse {
  users: User[];
  pagination: {
    page: number;
    per_page: number;
    total: number;
    total_pages: number;
  };
}

export function getUsers(params?: GetUsersParams): Promise<GetUsersResponse> {
  const searchParams = new URLSearchParams();
  if (params?.page) searchParams.append("page", params.page.toString());
  if (params?.per_page) searchParams.append("per_page", params.per_page.toString());

  const url = base_url + `/users` + (searchParams.toString() ? `?${searchParams}` : "");
  let token = localStorage.getItem("token");

  return fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<GetUsersResponse>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export async function updateUser(user: User): Promise<User> {
  return editUser(user.id.toString(), {
    username: user.username,
    email: user.email,
    is_admin: user.is_admin,
    dashboard_card_pk: user.dashboard_card_pk,
    has_seen_getting_started: user.has_seen_getting_started,
    timezone: user.timezone,
  });
}

export async function editUser(
  userId: string,
  updateData: EditUserParams,
): Promise<User> {
  let token = localStorage.getItem("token");
  const url = `${base_url}/users/${userId}`;

  const response = await fetch(url, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(updateData),
  });

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }
  return response.json();
}

export function getCurrentUser(): Promise<User> {
  const url = base_url + `/current`;
  let token = localStorage.getItem("token");

  // Send a GET request to the URL
  return fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<User>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export async function checkAdmin(): Promise<boolean> {
  let url = `${base_url}/admin`;
  let token = localStorage.getItem("token");
  let response = await fetch(url, {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
  });
  if (response.status === 204) {
    return true;
  } else {
    return false;
  }
}

export function validateEmail(token: string): Promise<GenericResponse> {
  const url = `${base_url}/email-validate`;

  return fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ token }),
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<GenericResponse>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}
export async function resendValidateEmail(): Promise<GenericResponse> {
  let token = localStorage.getItem("token");
  const url = `${base_url}/email-validate`;

  return fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<GenericResponse>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function getUserSubscription(id: number): Promise<UserSubscription> {
  let encodedId = encodeURIComponent(id);
  const url = `${base_url}/users/${encodedId}/subscription`;
  let token = localStorage.getItem("token");

  // Send a GET request to the URL
  return fetch(url, {
    method: "GET", // Specify the method
    headers: {
      Authorization: `Bearer ${token}`, // Include the JWT token in the Authorization header
    },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<UserSubscription>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    })
    .catch((error) => {
      console.error("Error fetching user subscription:", error);
      throw error;
    });
}

export function addToMailingList(email: string): Promise<{ email: string }> {
  let url = base_url + "/mailing-list";

  return fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email }),
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json();
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export interface MailingListSubscriber {
  id: number;
  email: string;
  welcome_email_sent: boolean;
  subscribed: boolean;
  has_account: boolean;
  created_at: string;
  updated_at: string;
}

export function getMailingListSubscribers(): Promise<MailingListSubscriber[]> {
  const url = `${base_url}/mailing-list`;
  let token = localStorage.getItem("token");

  return fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<MailingListSubscriber[]>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function unsubscribeMailingList(email: string): Promise<{ message: string }> {
  const url = `${base_url}/mailing-list/unsubscribe`;
  let token = localStorage.getItem("token");

  return fetch(url, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email }),
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<{ message: string }>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function updateUserMemory(memory: string): Promise<{ message: string }> {
  const url = `${base_url}/user/memory`;
  let token = localStorage.getItem("token");

  return fetch(url, {
    method: 'PUT',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ memory }),
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json() as Promise<{ message: string }>;
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}
