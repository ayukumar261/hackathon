import { ApiError, api } from "./api";

export type SessionUser = {
  id: string;
  email: string;
  name: string;
  picture: string;
};

type SessionResponse = { user: SessionUser };

export async function getSession(): Promise<SessionUser | null> {
  try {
    const data = await api.get<SessionResponse>("/sessions");
    return data.user;
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) return null;
    throw err;
  }
}

export async function logout(): Promise<void> {
  await api.delete("/sessions");
}
