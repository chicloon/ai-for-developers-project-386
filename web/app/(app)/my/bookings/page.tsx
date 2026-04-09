"use client";

export const dynamic = 'force-dynamic';

import { useEffect, useState, useMemo, useRef } from "react";
import { Schedule, ScheduleEventData } from "@mantine/schedule";
import { Modal, Title, Stack, Text, Button, Group, Loader, Center, Badge, Divider } from "@mantine/core";
import { DatesProvider } from "@mantine/dates";
import { Booking, getMyBookings, cancelBooking } from "@/lib/api";
import { useAuth } from "@/components/auth/AuthProvider";
import dayjs from "dayjs";
import customParseFormat from "dayjs/plugin/customParseFormat";
import "dayjs/locale/ru";

// Настройка локали и плагинов для этого модуля
dayjs.locale("ru");
dayjs.extend(customParseFormat);

// Format time from HH:mm:ss or HH:mm:ss.ssssss or HH:mm to HH:mm
function formatTime(timeStr: string): string {
  // Remove microseconds if present
  const cleanStr = timeStr.split('.')[0];
  // Already in HH:mm format (HH:mm has 5 chars: "09:00")
  if (cleanStr.length === 5 && cleanStr.includes(':')) {
    return cleanStr;
  }
  // Has seconds - parse and format
  const parsed = dayjs(cleanStr, "HH:mm:ss");
  return parsed.isValid() ? parsed.format("HH:mm") : cleanStr;
}

// Цвета для групп
const GROUP_COLORS: Record<string, string> = {
  family: "green",
  work: "blue",
  friends: "orange",
  public: "gray"
};

function formatDate(dateStr: string): string {
  if (!dateStr) return "Повторяющееся";
  const date = new Date(dateStr);
  return date.toLocaleDateString("ru-RU", {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

/** Активное бронирование, интервал которого уже закончился (по локальному времени). */
function isBookingPast(booking: Booking): boolean {
  if (booking.status !== "active" || !booking.date) return false;
  const endTime = booking.endTime.split(".")[0];
  const end = dayjs(`${booking.date} ${endTime}`, "YYYY-MM-DD HH:mm:ss", true);
  return end.isValid() && end.isBefore(dayjs());
}

// Преобразование бронирования в событие календаря
function bookingToEvent(booking: Booking): ScheduleEventData {
  // Для повторяющихся бронирований без даты не создаем событие календаря
  if (!booking.date) {
    const startTime = booking.startTime.split('.')[0];
    const endTime = booking.endTime.split('.')[0];
    return {
      id: booking.id,
      title: `${booking.booker.name} → ${booking.owner.name}`,
      start: `${new Date().toISOString().split('T')[0]} ${startTime}`,
      end: `${new Date().toISOString().split('T')[0]} ${endTime}`,
      color: "gray",
    };
  }

  // Убираем микросекунды из времени (PostgreSQL возвращает HH:MM:SS.mmmmmm)
  const startTime = booking.startTime.split('.')[0];
  const endTime = booking.endTime.split('.')[0];
  const start = `${booking.date} ${startTime}`;
  const end = `${booking.date} ${endTime}`;

  // Определяем цвет по первой группе бронирования (используем groups с бэкенда)
  let color = "gray";
  if (booking.groups && booking.groups.length > 0) {
    const group = booking.groups[0];
    color = GROUP_COLORS[group.visibilityLevel] || "gray";
  }

  // Отмененные бронирования - красным
  if (booking.status === "cancelled") {
    color = "red";
  } else if (isBookingPast(booking)) {
    color = "gray";
  }

  return {
    id: booking.id,
    title: `${booking.booker.name} → ${booking.owner.name}`,
    start,
    end,
    color,
  };
}

// Компонент для рендера события
function BookingEvent({
  event,
}: {
  event: ScheduleEventData;
}) {
  return (
    <Text size="sm" fw={500} truncate>
      {event.title}
    </Text>
  );
}

// Компонент модалки с деталями бронирования
function BookingModal({
  booking,
  opened,
  onClose,
  onCancel,
}: {
  booking: Booking | null;
  opened: boolean;
  onClose: () => void;
  onCancel: (id: string) => void;
}) {
  const { user } = useAuth();

  if (!booking) return null;

  // Используем группы из бронирования (приходят с бэкенда с названиями)
  const bookingGroups = booking.groups || [];

  const past = isBookingPast(booking);
  const canCancel =
    booking.status === "active" &&
    !past &&
    (booking.booker.id === user?.id || booking.owner.id === user?.id);

  const statusBadge =
    booking.status === "cancelled"
      ? { color: "red" as const, label: "Отменено" }
      : past
        ? { color: "gray" as const, label: "Прошедшее" }
        : { color: "green" as const, label: "Активно" };

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title="Бронирование"
      centered
    >
      <Stack gap="xs">
        <Group justify="space-between">
          <Text fw={700}>Детали</Text>
          <Badge color={statusBadge.color}>{statusBadge.label}</Badge>
        </Group>

        <Text size="sm">
          <b>Дата:</b> {formatDate(booking.date)}
        </Text>
        <Text size="sm">
          <b>Время:</b> {formatTime(booking.startTime)} - {formatTime(booking.endTime)}
        </Text>

        <Divider />

        <Text size="sm" fw={500}>Клиент</Text>
        <Text size="sm">{booking.booker.name}</Text>
        <Text size="sm" c="dimmed">{booking.booker.email}</Text>

        <Text size="sm" fw={500}>Владелец</Text>
        <Text size="sm">{booking.owner.name}</Text>
        <Text size="sm" c="dimmed">{booking.owner.email}</Text>

        {canCancel && (
          <>
            <Divider />
            <Button
              color="red"
              size="sm"
              onClick={() => {
                onCancel(booking.id);
                onClose();
              }}
            >
              Отменить бронирование
            </Button>
          </>
        )}
      </Stack>
    </Modal>
  );
}

export default function MyBookingsPage() {
  const { user } = useAuth();
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [loading, setLoading] = useState(true);
  const [cancellingId, setCancellingId] = useState<string | null>(null);
  const [selectedBooking, setSelectedBooking] = useState<Booking | null>(null);
  const [modalOpened, setModalOpened] = useState(false);
  const [view, setView] = useState<'month' | 'week' | 'day' | 'year'>('month');

  const handleViewChange = (newView: 'month' | 'week' | 'day' | 'year') => {
    setView(newView);
  };
  const [date, setDate] = useState<Date>(new Date());
  const scheduleRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const bookingsData = await getMyBookings();
      setBookings(bookingsData.bookings);
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
      await loadData();
    } catch (e) {
      console.error(e);
    } finally {
      setCancellingId(null);
    }
  };

  const handleEventClick = (event: ScheduleEventData) => {
    const booking = bookings.find(b => b.id === event.id);
    if (booking) {
      setSelectedBooking(booking);
      setModalOpened(true);
    }
  };

  const handleCloseModal = () => {
    setModalOpened(false);
    setSelectedBooking(null);
  };

  // Intercept clicks on "+X more" button to switch to week view
  useEffect(() => {
    if (loading) return;

    const scheduleElement = scheduleRef.current;
    if (!scheduleElement) return;

    const handleClick = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      const moreButton = target.closest('button');

      // Check if the button text matches "+X more" pattern
      if (moreButton && moreButton.textContent?.match(/^\+\d+\s+more$/)) {
        e.preventDefault();
        e.stopPropagation();

        // Walk up the DOM to find a button with a date in aria-label
        let current: HTMLElement | null = moreButton as HTMLElement;
        let dayButton: Element | null = null;

        while (current && current !== scheduleElement) {
          const buttons = current.querySelectorAll('button');
          for (const btn of buttons) {
            const label = btn.getAttribute('aria-label');
            // Look for date pattern (contains 4-digit year and not "more")
            if (label && /\d{4}/.test(label) && !label.includes('more')) {
              dayButton = btn;
              break;
            }
          }
          if (dayButton) break;
          current = current.parentElement;
        }

        if (dayButton) {
          const label = dayButton.getAttribute('aria-label');
          if (label) {
            const clickedDate = new Date(label);
            if (!isNaN(clickedDate.getTime())) {
              setDate(clickedDate);
              setView('week');
              return;
            }
          }
        }

        // Fallback: just switch to week view
        setView('week');
      }
    };

    scheduleElement.addEventListener('click', handleClick, true);
    return () => {
      scheduleElement.removeEventListener('click', handleClick, true);
    };
  }, [loading]);

  // Преобразуем бронирования в события календаря
  const events = useMemo(() => {
    const filtered = bookings.filter(b => b.date);
    const mapped = filtered.map(b => bookingToEvent(b));
    return mapped;
  }, [bookings]);

  if (loading) {
    return (
      <Center h="50vh" data-testid="bookings-loading">
        <Loader />
      </Center>
    );
  }

  return (
    <Stack gap="md" data-testid="bookings-page">
      <Title order={2} data-testid="bookings-title">Мои бронирования</Title>

      {bookings.length === 0 && (
        <Text c="dimmed" size="sm" data-testid="bookings-empty">
          У вас пока нет бронирований
        </Text>
      )}

      <div ref={scheduleRef}>
        <DatesProvider settings={{ locale: "ru" }}>
          <Schedule
            events={events}
            view={view}
            onViewChange={handleViewChange}
            date={date}
            onDateChange={(newDate) => setDate(new Date(newDate))}
            onEventClick={handleEventClick}
            renderEventBody={(event) => (
              <BookingEvent event={event} />
            )}
            labels={{
              day: "День",
              week: "Неделя",
              month: "Месяц",
              year: "Год",
              today: "Сегодня",
              next: "Вперед",
              previous: "Назад",
              allDay: "Весь день",
              weekday: "Будний день",
              timeSlot: "Время",
              selectMonth: "Выбрать месяц",
              selectYear: "Выбрать год",
              switchToDayView: "Переключить на день",
              switchToWeekView: "Переключить на неделю",
              switchToMonthView: "Переключить на месяц",
              switchToYearView: "Переключить на год",
              viewSelectLabel: "Вид календаря",
              noEvents: "Нет событий",
              more: "Еще",
              moreLabel: (count) => `+${count} еще`,
            }}
            dayViewProps={{
              startTime: '08:00:00',
              endTime: '20:00:00',
              intervalMinutes: 30,
            }}
            weekViewProps={{
              startTime: '08:00:00',
              endTime: '20:00:00',
            }}
            monthViewProps={{
              withWeekNumbers: true,
              firstDayOfWeek: 1,
              maxEventsPerDay: 10,
            }}
          />
        </DatesProvider>
      </div>

      <BookingModal
        booking={selectedBooking}
        opened={modalOpened}
        onClose={handleCloseModal}
        onCancel={handleCancel}
      />
    </Stack>
  );
}
