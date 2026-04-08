"use client";

import { useEffect, useState } from "react";
import {
  Paper,
  Title,
  Stack,
  Table,
  Button,
  Loader,
  Center,
  Text,
  Modal,
  TextInput,
  Select,
  Group,
  ActionIcon,
  Checkbox,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { TimeInput } from "@mantine/dates";
import { IconTrash, IconEdit } from "@tabler/icons-react";
import {
  Schedule,
  getMySchedules,
  createSchedule,
  updateSchedule,
  deleteSchedule,
  CreateScheduleRequest,
} from "@/lib/api";

const DAYS_OF_WEEK = [
  { value: "1", label: "Понедельник" },
  { value: "2", label: "Вторник" },
  { value: "3", label: "Среда" },
  { value: "4", label: "Четверг" },
  { value: "5", label: "Пятница" },
  { value: "6", label: "Суббота" },
  { value: "0", label: "Воскресенье" },
];

export default function MySchedulePage() {
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [loading, setLoading] = useState(true);
  const [opened, { open, close }] = useDisclosure(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Form state
  const [type, setType] = useState<"recurring" | "one-time">("recurring");
  const [dayOfWeek, setDayOfWeek] = useState<string | null>("1");
  const [date, setDate] = useState<string>("");
  const [startTime, setStartTime] = useState<string>("09:00");
  const [endTime, setEndTime] = useState<string>("17:00");
  const [isBlocked, setIsBlocked] = useState<boolean>(false);

  useEffect(() => {
    loadSchedules();
  }, []);

  const loadSchedules = async () => {
    try {
      setLoading(true);
      const data = await getMySchedules();
      setSchedules(data.schedules);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const handleOpenCreate = () => {
    setEditingId(null);
    resetForm();
    open();
  };

  const handleOpenEdit = (schedule: Schedule) => {
    setEditingId(schedule.id);
    setType(schedule.type);
    setDayOfWeek(schedule.dayOfWeek?.toString() || "1");
    setDate(schedule.date || "");
    setStartTime(schedule.startTime);
    setEndTime(schedule.endTime);
    setIsBlocked(schedule.isBlocked);
    open();
  };

  const handleSubmit = async () => {
    try {
      setSubmitting(true);
      const data: CreateScheduleRequest = {
        type,
        dayOfWeek: type === "recurring" ? parseInt(dayOfWeek || "1") : undefined,
        date: type === "one-time" ? date : undefined,
        startTime,
        endTime,
        isBlocked,
      };

      if (editingId) {
        await updateSchedule(editingId, data);
      } else {
        await createSchedule(data);
      }
      close();
      resetForm();
      await loadSchedules();
    } catch (e) {
      console.error(e);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteSchedule(id);
      await loadSchedules();
    } catch (e) {
      console.error(e);
    }
  };

  const resetForm = () => {
    setType("recurring");
    setDayOfWeek("1");
    setDate("");
    setStartTime("09:00");
    setEndTime("17:00");
    setIsBlocked(false);
    setEditingId(null);
  };

  if (loading) {
    return (
      <Center h="50vh" data-testid="schedule-loading">
        <Loader />
      </Center>
    );
  }

  return (
    <Stack gap="md" data-testid="schedule-page">
      <Group justify="space-between">
        <Title order={2} data-testid="schedule-title">Моё расписание</Title>
        <Button onClick={handleOpenCreate} data-testid="schedule-add-button">Добавить расписание</Button>
      </Group>

      {schedules.length === 0 ? (
        <Text c="dimmed" data-testid="schedule-empty">У вас пока нет настроенных расписаний</Text>
      ) : (
        <Paper withBorder data-testid="schedule-table">
          <Table>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Тип</Table.Th>
                <Table.Th>День/дата</Table.Th>
                <Table.Th>Время</Table.Th>
                <Table.Th>Статус</Table.Th>
                <Table.Th></Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {schedules.map((schedule) => (
                <Table.Tr key={schedule.id} data-testid={`schedule-row-${schedule.id}`}>
                  <Table.Td data-testid={`schedule-type-${schedule.id}`}>
                    {schedule.type === "recurring" ? "Повторяющееся" : "Разовое"}
                  </Table.Td>
                  <Table.Td data-testid={`schedule-day-${schedule.id}`}>
                    {schedule.type === "recurring"
                      ? DAYS_OF_WEEK.find((d) => d.value === String(schedule.dayOfWeek))?.label
                      : schedule.date}
                  </Table.Td>
                  <Table.Td data-testid={`schedule-time-${schedule.id}`}>
                    {schedule.startTime} - {schedule.endTime}
                  </Table.Td>
                  <Table.Td data-testid={`schedule-status-${schedule.id}`}>
                    {schedule.isBlocked ? (
                      <Text c="red" size="sm">Заблокировано</Text>
                    ) : (
                      <Text c="green" size="sm">Доступно</Text>
                    )}
                  </Table.Td>
                  <Table.Td>
                    <Group gap="xs">
                      <ActionIcon
                        color="blue"
                        onClick={() => handleOpenEdit(schedule)}
                        data-testid={`schedule-edit-${schedule.id}`}
                      >
                        <IconEdit size={16} />
                      </ActionIcon>
                      <ActionIcon
                        color="red"
                        onClick={() => handleDelete(schedule.id)}
                        data-testid={`schedule-delete-${schedule.id}`}
                      >
                        <IconTrash size={16} />
                      </ActionIcon>
                    </Group>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </Paper>
      )}

      <Modal
        opened={opened}
        onClose={close}
        title={editingId ? "Редактировать расписание" : "Добавить расписание"}
        size="lg"
        data-testid="schedule-modal"
      >
        <Stack gap="md">
          <Select
            label="Тип расписания"
            value={type}
            onChange={(value) => setType(value as "recurring" | "one-time")}
            data={[
              { value: "recurring", label: "Повторяющееся" },
              { value: "one-time", label: "Разовое" },
            ]}
            data-testid="schedule-type-select"
          />

          {type === "recurring" ? (
            <Select
              label="День недели"
              value={dayOfWeek}
              onChange={setDayOfWeek}
              data={DAYS_OF_WEEK}
              data-testid="schedule-day-select"
            />
          ) : (
            <TextInput
              label="Дата"
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              data-testid="schedule-date-input"
            />
          )}

          <Group grow>
            <TimeInput
              label="Начало"
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
              data-testid="schedule-start-time"
            />
            <TimeInput
              label="Конец"
              value={endTime}
              onChange={(e) => setEndTime(e.target.value)}
              data-testid="schedule-end-time"
            />
          </Group>

          <Checkbox
            label="Заблокировать (недоступно для бронирования)"
            checked={isBlocked}
            onChange={(e) => setIsBlocked(e.currentTarget.checked)}
            data-testid="schedule-blocked-checkbox"
          />

          <Group justify="flex-end" mt="md">
            <Button variant="default" onClick={close} data-testid="schedule-cancel-button">
              Отмена
            </Button>
            <Button onClick={handleSubmit} loading={submitting} data-testid="schedule-submit-button">
              {editingId ? "Сохранить" : "Создать"}
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
