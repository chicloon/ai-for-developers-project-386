namespace CallBooking;

// Common response model
model ErrorResponse {
  error: string;
}

// User models
model User {
  id: string;
  email: string;
  name: string;
  isPublic: boolean;
  createdAt?: string;
  updatedAt?: string;
}

model UpdateUserRequest {
  name?: string;
  isPublic?: boolean;
}

model RegisterRequest {
  email: string;
  password: string;
  name: string;
}

model LoginRequest {
  email: string;
  password: string;
}

model AuthResponse {
  token: string;
  user: User;
}

// Schedule models (replaces AvailabilityRule)
model Schedule {
  id: string;
  userId: string;
  type: "recurring" | "one-time";
  dayOfWeek?: int32;
  date?: string;
  startTime: string;
  endTime: string;
  isBlocked: boolean;
  groupIds?: string[];
  createdAt?: string;
}

model CreateScheduleRequest {
  type: "recurring" | "one-time";
  dayOfWeek?: int32;
  date?: string;
  startTime: string;
  endTime: string;
  isBlocked?: boolean;
  groupIds?: string[];
}

model SchedulesListResponse {
  schedules: Schedule[];
}

// Visibility Group models
// Groups are auto-created for each user: Family, Work, Friends
model VisibilityGroup {
  id: string;
  ownerId: string;
  name: string;
  visibilityLevel: "family" | "work" | "friends";
  createdAt?: string;
}

model GroupsListResponse {
  groups: VisibilityGroup[];
}

// Group Member models
model GroupMember {
  id: string;
  groupId: string;
  member: User;
  addedBy: string;
  addedAt: string;
}

model AddMemberRequest {
  email?: string;
  userId?: string;
}

model MembersListResponse {
  members: GroupMember[];
}

// Booking models
model Booking {
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
}

model CreateBookingRequest {
  ownerId: string;
  scheduleId: string;
}

model BookingsListResponse {
  bookings: Booking[];
}

// Slot model for public display
model Slot {
  id: string;
  date: string;
  startTime: string;
  endTime: string;
  isBooked: boolean;
}

model SlotsListResponse {
  slots: Slot[];
}

// Users list response
model UsersListResponse {
  users: User[];
}
