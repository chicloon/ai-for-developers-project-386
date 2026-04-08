"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
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
  Alert,
} from "@mantine/core";
import { DatePickerInput } from "@mantine/dates";
import { User, Slot, getUser, getUserSlots, createBooking } from "@/lib/api";
import { useAuth } from "@/components/auth/AuthProvider";

export default function UserProfilePage() {
  const params = useParams();
  const userId = params.id as string;
  const { user: currentUser } = useAuth();

  const [user, setUser] = useState<User | null>(null);
  const [selectedDate, setSelectedDate] = useState<Date | null>(new Date());
  const [slots, setSlots] = useState<Slot[]>([]);
  const [loading, setLoading] = useState(true);
  const [slotsLoading, setSlotsLoading] = useState(false);
  const [bookingInProgress, setBookingInProgress] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    loadUser();
  }, [userId]);

  useEffect(() => {
    if (selectedDate && userId) {
      loadSlots();
    }
  }, [selectedDate, userId]);

  const loadUser = async () => {
    try {
      setLoading(true);
      const data = await getUser(userId);
      setUser(data);
    } catch (e) {
      console.error(e);
      setError("Не удалось загрузить профиль пользователя");
    } finally {
      setLoading(false);
    }
  };

  const loadSlots = async () => {
    if (!selectedDate) return;
    try {
      setSlotsLoading(true);
      const dateStr = selectedDate.toISOString().split("T")[0];
      const data = await getUserSlots(userId, dateStr);
      setSlots(data.slots);
    } catch (e) {
      console.error(e);
    } finally {
      setSlotsLoading(false);
    }
  };

  const handleBooking = async (slot: Slot) => {
    if (!currentUser) return;
    try {
      setBookingInProgress(slot.id);
      setError(null);
      setSuccess(null);

      // Extract schedule ID from slot ID (format: "scheduleId_startTime")
      const scheduleId = slot.id.split("_")[0];

      await createBooking({
        ownerId: userId,
        scheduleId: scheduleId,
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

  if (loading) {
    return (
      <Center h="50vh">
        <Loader />
      </Center>
    );
  }

  if (!user) {
    return (
      <Center h="50vh">
        <Text c="dimmed">Пользователь не найден</Text>
      </Center>
    );
  }

  const availableSlots = slots.filter((s) => !s.isBooked);

  return (
    <Stack gap="md">
      <Title order={2}>Профиль пользователя</Title>

      <Card withBorder>
        <Text fw={500} size="lg">
          {user.name}
        </Text>
        <Text c="dimmed">{user.email}</Text>
      </Card>

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
                    {slot.startTime} - {slot.endTime}
                  </Text>
                  <Button
                    size="sm"
                    onClick={() => handleBooking(slot)}
                    loading={bookingInProgress === slot.id}
                    disabled={userId === currentUser?.id}
                  >
                    {userId === currentUser?.id ? "Это вы" : "Записаться"}
                  </Button>
                </Group>
              </Card>
            ))}
          </Stack>
        )}
      </Paper>
    </Stack>
  );
}
