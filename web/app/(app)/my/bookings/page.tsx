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
  Badge,
  Tabs,
} from "@mantine/core";
import { Booking, getMyBookings, cancelBooking } from "@/lib/api";
import { useAuth } from "@/components/auth/AuthProvider";

function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleDateString("ru-RU", {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

function getStatusColor(status: string): string {
  switch (status) {
    case "active":
      return "green";
    case "cancelled":
      return "red";
    default:
      return "gray";
  }
}

function getStatusLabel(status: string): string {
  switch (status) {
    case "active":
      return "Активно";
    case "cancelled":
      return "Отменено";
    default:
      return status;
  }
}

interface BookingCardProps {
  booking: Booking;
  showCancel?: boolean;
  onCancel?: (id: string) => void;
}

function BookingCard({ booking, showCancel, onCancel }: BookingCardProps) {
  const isIncoming = new Date(booking.date) >= new Date();

  return (
    <Card withBorder>
      <Group justify="space-between" align="flex-start">
        <Stack gap="xs">
          <Text fw={500}>
            {formatDate(booking.date)} в {booking.startTime} - {booking.endTime}
          </Text>
          <Group gap="xs">
            <Badge color={getStatusColor(booking.status)}>
              {getStatusLabel(booking.status)}
            </Badge>
          </Group>
          <Text size="sm" c="dimmed">
            Клиент: {booking.booker.name} ({booking.booker.email})
          </Text>
          <Text size="sm" c="dimmed">
            Владелец: {booking.owner.name} ({booking.owner.email})
          </Text>
        </Stack>
        {showCancel && booking.status !== "cancelled" && onCancel && (
          <Button
            variant="light"
            color="red"
            size="sm"
            onClick={() => onCancel(booking.id)}
          >
            Отменить
          </Button>
        )}
      </Group>
    </Card>
  );
}

export default function MyBookingsPage() {
  const { user } = useAuth();
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<string>("incoming");
  const [cancellingId, setCancellingId] = useState<string | null>(null);

  useEffect(() => {
    loadBookings();
  }, []);

  const loadBookings = async () => {
    try {
      setLoading(true);
      const data = await getMyBookings();
      setBookings(data.bookings);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = async (id: string) => {
    try {
      setCancellingId(id);
      await cancelBooking(id);
      await loadBookings();
    } catch (e) {
      console.error(e);
    } finally {
      setCancellingId(null);
    }
  };

  if (loading) {
    return (
      <Center h="50vh">
        <Loader />
      </Center>
    );
  }

  // Filter bookings based on active tab
  const incomingBookings = bookings.filter(
    (b) => b.status !== "cancelled" && new Date(b.date) >= new Date()
  );
  const pastBookings = bookings.filter(
    (b) => b.status === "cancelled" || new Date(b.date) < new Date()
  );

  return (
    <Stack gap="md">
      <Title order={2}>Мои бронирования</Title>

      <Tabs
        value={activeTab}
        onChange={(value) => setActiveTab(value || "incoming")}
      >
        <Tabs.List>
          <Tabs.Tab value="incoming">
            Предстоящие ({incomingBookings.length})
          </Tabs.Tab>
          <Tabs.Tab value="past">История ({pastBookings.length})</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="incoming" pt="md">
          {incomingBookings.length === 0 ? (
            <Text c="dimmed">Нет предстоящих бронирований</Text>
          ) : (
            <Stack gap="md">
              {incomingBookings.map((booking) => (
                <BookingCard
                  key={booking.id}
                  booking={booking}
                  showCancel={booking.booker.id === user?.id}
                  onCancel={handleCancel}
                />
              ))}
            </Stack>
          )}
        </Tabs.Panel>

        <Tabs.Panel value="past" pt="md">
          {pastBookings.length === 0 ? (
            <Text c="dimmed">Нет прошедших бронирований</Text>
          ) : (
            <Stack gap="md">
              {pastBookings.map((booking) => (
                <BookingCard key={booking.id} booking={booking} showCancel={false} />
              ))}
            </Stack>
          )}
        </Tabs.Panel>
      </Tabs>
    </Stack>
  );
}
