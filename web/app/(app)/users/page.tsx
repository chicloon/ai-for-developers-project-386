"use client";

import { useEffect, useState } from "react";
import {
  Paper,
  Title,
  Stack,
  Card,
  Group,
  Text,
  Button,
  Loader,
  Center,
  Select,
  Alert,
} from "@mantine/core";
import { DatePickerInput } from "@mantine/dates";
import {
  User,
  Slot,
  getUsers,
  getUser,
  getUserSlots,
  createBooking,
} from "@/lib/api";
import { useAuth } from "@/components/auth/AuthProvider";
import dayjs from "dayjs";
import customParseFormat from "dayjs/plugin/customParseFormat";

dayjs.extend(customParseFormat);

// Format time from HH:mm:ss or HH:mm:ss.ssssss or HH:mm to HH:mm
function formatTime(timeStr: string): string {
  const cleanStr = timeStr.split('.')[0];
  if (cleanStr.length === 5 && cleanStr.includes(':')) {
    return cleanStr;
  }
  const parsed = dayjs(cleanStr, "HH:mm:ss");
  return parsed.isValid() ? parsed.format("HH:mm") : cleanStr;
}

export default function UsersCatalogPage() {
  const { user: currentUser } = useAuth();
  
  // Список всех пользователей для Select
  const [users, setUsers] = useState<User[]>([]);
  const [usersLoading, setUsersLoading] = useState(true);
  
  // Выбранный пользователь
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [userLoading, setUserLoading] = useState(false);
  
  // Дата и слоты
  const [selectedDate, setSelectedDate] = useState<Date | null>(new Date());
  const [slots, setSlots] = useState<Slot[] | null>(null);
  const [slotsLoading, setSlotsLoading] = useState(false);
  
  // Бронирование
  const [bookingInProgress, setBookingInProgress] = useState<string | null>(null);
  
  // Уведомления
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Загрузка списка пользователей при монтировании
  useEffect(() => {
    loadUsers();
  }, []);

  const loadUsers = async () => {
    try {
      setUsersLoading(true);
      const data = await getUsers();
      setUsers(data.users);
    } catch (e) {
      console.error(e);
      setError("Не удалось загрузить список пользователей");
    } finally {
      setUsersLoading(false);
    }
  };

  // Загрузка выбранного пользователя
  useEffect(() => {
    if (selectedUserId) {
      loadSelectedUser();
    } else {
      setSelectedUser(null);
      setSlots(null);
    }
  }, [selectedUserId]);

  const loadSelectedUser = async () => {
    if (!selectedUserId) return;
    try {
      setUserLoading(true);
      const data = await getUser(selectedUserId);
      setSelectedUser(data);
    } catch (e) {
      console.error(e);
      setError("Не удалось загрузить профиль пользователя");
    } finally {
      setUserLoading(false);
    }
  };

  // Загрузка слотов при изменении пользователя или даты
  useEffect(() => {
    if (selectedUserId && selectedDate) {
      loadSlots();
    }
  }, [selectedUserId, selectedDate]);

  const loadSlots = async () => {
    if (!selectedUserId || !selectedDate) return;
    try {
      setSlotsLoading(true);
      const dateStr = selectedDate.toISOString().split("T")[0];
      const data = await getUserSlots(selectedUserId, dateStr);
      setSlots(data.slots ?? []);
    } catch (e) {
      console.error(e);
      setError("Не удалось загрузить слоты");
    } finally {
      setSlotsLoading(false);
    }
  };

  const handleBooking = async (slot: Slot) => {
    if (!currentUser || !selectedUserId) return;
    try {
      setBookingInProgress(slot.id);
      setError(null);
      setSuccess(null);

      const [scheduleId, slotStartTime] = slot.id.split("_");

      await createBooking({
        ownerId: selectedUserId,
        scheduleId: scheduleId,
        slotStartTime: slotStartTime,
        slotDate: slot.date,
      });

      setSuccess("Запись успешно создана!");
      await loadSlots();
    } catch (e: any) {
      console.error(e);
      setError(e.message || "Не удалось создать запись");
    } finally {
      setBookingInProgress(null);
    }
  };

  // Формирование данных для Select
  const selectData = users.map((user) => ({
    value: user.id,
    label: `${user.name} (${user.email})`,
  }));

  const availableSlots = slots?.filter((s) => !s.isBooked) ?? [];

  return (
    <Stack gap="md" data-testid="users-page">
      <Title order={2} data-testid="users-title">
        Каталог пользователей
      </Title>

      {error && (
        <Alert color="red" onClose={() => setError(null)} withCloseButton>
          {error}
        </Alert>
      )}

      {success && (
        <Alert color="green" onClose={() => setSuccess(null)} withCloseButton>
          {success}
        </Alert>
      )}

      <Select
        label="Выберите пользователя"
        placeholder="Начните вводить имя или email"
        data={selectData}
        value={selectedUserId}
        onChange={setSelectedUserId}
        searchable
        clearable
        disabled={usersLoading}
        data-testid="users-select"
      />

      {usersLoading && (
        <Center h="100px">
          <Loader />
        </Center>
      )}

      {userLoading && (
        <Center h="200px">
          <Loader />
        </Center>
      )}

      {selectedUser && !userLoading && (
        <>
          <Card withBorder data-testid={`user-card-${selectedUser.id}`}>
            <Text fw={500} size="lg" data-testid={`user-name-${selectedUser.id}`}>
              {selectedUser.name}
            </Text>
            <Text c="dimmed" data-testid={`user-email-${selectedUser.id}`}>
              {selectedUser.email}
            </Text>
          </Card>

          <Paper p="md" withBorder>
            <Title order={4} mb="md">
              Выберите дату
            </Title>
            <DatePickerInput
              value={selectedDate}
              onChange={(value) => setSelectedDate(value ? new Date(value) : null)}
              locale="ru"
              minDate={new Date()}
            />
          </Paper>

          <Paper p="md" withBorder>
            <Title order={4} mb="md">
              Доступное время
            </Title>
            {slotsLoading ? (
              <Center>
                <Loader />
              </Center>
            ) : availableSlots.length === 0 ? (
              <Text c="dimmed">Нет доступного времени на выбранную дату</Text>
            ) : (
              <Stack gap="xs">
                {availableSlots.map((slot) => (
                  <Card key={slot.id} withBorder padding="sm">
                    <Group justify="space-between">
                      <Text>
                        {formatTime(slot.startTime)} - {formatTime(slot.endTime)}
                      </Text>
                      <Button
                        size="sm"
                        onClick={() => handleBooking(slot)}
                        loading={bookingInProgress === slot.id}
                        disabled={selectedUserId === currentUser?.id}
                      >
                        {selectedUserId === currentUser?.id ? "Это вы" : "Записаться"}
                      </Button>
                    </Group>
                  </Card>
                ))}
              </Stack>
            )}
          </Paper>
        </>
      )}
    </Stack>
  );
}
