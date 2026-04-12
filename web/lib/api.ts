/** Same-origin `/api/*` only — never use localhost or absolute URLs in the browser (breaks on public HTTP origins). */

// Auth types
export interface User {
  id: string;
  email: string;
  name: string;
  isPublic: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface UpdateUserRequest {
  name?: string;
  isPublic?: boolean;
}

export interface RegisterRequest {
  email: string;
  password: string;
  name: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

// Schedule types (updated from AvailabilityRule)
export interface Schedule {
  id: string;
  userId: string;
  type: "recurring" | "one-time";
  dayOfWeek?: number;
  date?: string;
  startTime: string;
  endTime: string;
  isBlocked: boolean;
  groupIds?: string[];
  createdAt?: string;
}

export interface CreateScheduleRequest {
  type: "recurring" | "one-time";
  dayOfWeek?: number;
  date?: string;
  startTime: string;
  endTime: string;
  isBlocked?: boolean;
  groupIds?: string[];
}

export interface SchedulesListResponse {
  schedules: Schedule[];
}

// Visibility Group types (fixed groups: Family, Work, Friends)
export interface VisibilityGroup {
  id: string;
  ownerId: string;
  name: string;
  visibilityLevel: "family" | "work" | "friends";
  createdAt?: string;
}

export interface GroupsListResponse {
  groups: VisibilityGroup[];
}

// Group Member types
export interface GroupMember {
  id: string;
  groupId: string;
  member: User;
  addedBy: string;
  addedAt: string;
}

export interface AddMemberRequest {
  email?: string;
  userId?: string;
}

export interface MembersListResponse {
  members: GroupMember[];
}

// Booking types (updated)
export interface Booking {
  id: string;
  scheduleId: string;
  booker: User;
  owner: User;
  date: string;
  startTime: string;
  endTime: string;
  status: "active" | "cancelled";
  createdAt?: string;
  cancelledAt?: string;
  groupIds?: string[];
  groups?: VisibilityGroup[]; // Full group details with names
}

export interface CreateBookingRequest {
  ownerId: string;
  scheduleId: string;
  slotStartTime: string;
  slotDate: string;
}

export interface BookingsListResponse {
  bookings: Booking[];
}

// Slot type
export interface Slot {
  id: string;
  date: string;
  startTime: string;
  endTime: string;
  isBooked: boolean;
}

export interface SlotsListResponse {
  slots: Slot[];
}

export interface UsersListResponse {
  users: User[];
}

// Token storage
let authToken: string | null = null;

export function setAuthToken(token: string | null) {
  authToken = token;
  if (token) {
    localStorage.setItem("auth_token", token);
  } else {
    localStorage.removeItem("auth_token");
  }
}

export function getAuthToken(): string | null {
  if (!authToken) {
    authToken = localStorage.getItem("auth_token");
  }
  return authToken;
}

// Helper function for authenticated requests
async function authFetch(url: string, options?: RequestInit): Promise<Response> {
  const token = getAuthToken();
  const baseHeaders = options?.headers as Record<string, string> | undefined;
  const headers: Record<string, string> = {
    ...baseHeaders,
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  return fetch(url, {
    ...options,
    headers,
  });
}

/**
 * Reads error details from a failed response.
 * Go API returns JSON `{ "error": "..." }`; proxies may return plain text ("Internal Server Error") or HTML.
 */
async function readErrorMessage(res: Response, fallback: string): Promise<string> {
  const text = await res.text();
  const trimmed = text.trim();
  if (!trimmed) {
    return `${fallback} (HTTP ${res.status})`;
  }
  if (trimmed.startsWith("{")) {
    try {
      const data = JSON.parse(trimmed) as { error?: string };
      if (typeof data.error === "string" && data.error.length > 0) {
        return data.error;
      }
    } catch {
      /* not valid JSON */
    }
  }
  const oneLine = trimmed.replace(/\s+/g, " ");
  const short = oneLine.length > 180 ? `${oneLine.slice(0, 180)}…` : oneLine;
  return short || `${fallback} (HTTP ${res.status})`;
}

/** Prevents infinite loading when /api/auth/me never completes (API down, bad proxy, etc.). */
const GET_ME_TIMEOUT_MS = 15_000;

// Auth API
export async function register(data: RegisterRequest): Promise<AuthResponse> {
  const res = await fetch("/api/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    const msg = await readErrorMessage(res, "Registration failed");
    throw new Error(msg);
  }
  const result = await res.json();
  setAuthToken(result.token);
  return result;
}

export async function login(data: LoginRequest): Promise<AuthResponse> {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    const msg = await readErrorMessage(res, "Login failed");
    throw new Error(msg);
  }
  const result = await res.json();
  setAuthToken(result.token);
  return result;
}

export async function getMe(): Promise<User> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), GET_ME_TIMEOUT_MS);
  try {
    const res = await authFetch("/api/auth/me", { signal: controller.signal });
    if (!res.ok) {
      const msg = await readErrorMessage(res, "Failed to get user");
      throw new Error(msg);
    }
    return res.json();
  } finally {
    clearTimeout(timeoutId);
  }
}

export function logout() {
  setAuthToken(null);
}

// Users API
export async function getUsers(): Promise<UsersListResponse> {
  const res = await authFetch("/api/users");
  if (!res.ok) throw new Error("Failed to fetch users");
  return res.json();
}

export async function getAvailableUsers(): Promise<UsersListResponse> {
  const res = await authFetch("/api/my/available-users");
  if (!res.ok) throw new Error("Failed to fetch available users");
  return res.json();
}

export async function getUser(id: string): Promise<User> {
  const res = await authFetch(`/api/users/${encodeURIComponent(id)}`);
  if (!res.ok) throw new Error("Failed to fetch user");
  return res.json();
}

export async function getUserSlots(
  userId: string,
  date: string
): Promise<SlotsListResponse> {
  const res = await authFetch(
    `/api/users/${encodeURIComponent(userId)}/slots?date=${encodeURIComponent(date)}`
  );
  if (!res.ok) throw new Error("Failed to fetch user slots");
  return res.json();
}

export async function getUserSlotsRange(
  userId: string,
  start: string, // YYYY-MM-DD
  end: string    // YYYY-MM-DD
): Promise<SlotsListResponse> {
  const res = await authFetch(
    `/api/users/${encodeURIComponent(userId)}/slots-range?start=${encodeURIComponent(start)}&end=${encodeURIComponent(end)}`
  );
  if (!res.ok) throw new Error("Failed to fetch user slots range");
  return res.json();
}

// Available dates types
export interface AvailableDate {
  date: string;
  availableSlots: number;
}

export interface AvailableDatesResponse {
  dates: AvailableDate[];
}

export async function getUserAvailableDates(
  userId: string,
  month: string
): Promise<AvailableDatesResponse> {
  const res = await authFetch(
    `/api/users/${encodeURIComponent(userId)}/available-dates?month=${encodeURIComponent(month)}`
  );
  if (!res.ok) throw new Error("Failed to fetch available dates");
  return res.json();
}

export async function getUserAvailableDatesRange(
  userId: string,
  start: string,  // YYYY-MM-DD
  end: string     // YYYY-MM-DD
): Promise<AvailableDatesResponse> {
  const res = await authFetch(
    `/api/users/${encodeURIComponent(userId)}/available-dates-range?start=${encodeURIComponent(start)}&end=${encodeURIComponent(end)}`
  );
  if (!res.ok) throw new Error("Failed to fetch available dates range");
  return res.json();
}

export async function updateMe(data: UpdateUserRequest): Promise<User> {
  const res = await authFetch("/api/users/me", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error("Failed to update profile");
  return res.json();
}

// Schedules API
export async function getMySchedules(): Promise<SchedulesListResponse> {
  const res = await authFetch("/api/my/schedules");
  if (!res.ok) throw new Error("Failed to fetch schedules");
  return res.json();
}

export async function createSchedule(
  data: CreateScheduleRequest
): Promise<Schedule> {
  const res = await authFetch("/api/my/schedules", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error("Failed to create schedule");
  return res.json();
}

export async function updateSchedule(
  id: string,
  data: CreateScheduleRequest
): Promise<Schedule> {
  const res = await authFetch(`/api/my/schedules/${encodeURIComponent(id)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error("Failed to update schedule");
  return res.json();
}

export async function deleteSchedule(id: string): Promise<void> {
  const res = await authFetch(`/api/my/schedules/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
  if (!res.ok) throw new Error("Failed to delete schedule");
}

// Groups API (fixed groups - only member management)
export async function getMyGroups(): Promise<GroupsListResponse> {
  const res = await authFetch("/api/my/groups");
  if (!res.ok) throw new Error("Failed to fetch groups");
  return res.json();
}

export async function getGroupMembers(
  groupId: string
): Promise<MembersListResponse> {
  const res = await authFetch(
    `/api/my/groups/${encodeURIComponent(groupId)}/members`
  );
  if (!res.ok) throw new Error("Failed to fetch group members");
  return res.json();
}

export async function addGroupMember(
  groupId: string,
  data: AddMemberRequest
): Promise<GroupMember> {
  const res = await authFetch(
    `/api/my/groups/${encodeURIComponent(groupId)}/members`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    }
  );
  if (!res.ok) throw new Error("Failed to add group member");
  return res.json();
}

export async function removeGroupMember(
  groupId: string,
  memberId: string
): Promise<void> {
  const res = await authFetch(
    `/api/my/groups/${encodeURIComponent(groupId)}/members/${encodeURIComponent(memberId)}`,
    {
      method: "DELETE",
    }
  );
  if (!res.ok) throw new Error("Failed to remove group member");
}

// Bookings API
export async function getMyBookings(): Promise<BookingsListResponse> {
  const res = await authFetch("/api/my/bookings");
  if (!res.ok) throw new Error("Failed to fetch bookings");
  return res.json();
}

export async function createBooking(
  data: CreateBookingRequest
): Promise<Booking> {
  const res = await authFetch("/api/my/bookings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    let errData: any;
    try {
      errData = await res.json();
    } catch (e) {
      errData = { error: `HTTP ${res.status}` };
    }
    throw new Error(errData.error || "Failed to create booking");
  }
  return res.json();
}

export async function cancelBooking(id: string): Promise<void> {
  const res = await authFetch(`/api/my/bookings/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
  if (!res.ok) throw new Error("Failed to cancel booking");
}
